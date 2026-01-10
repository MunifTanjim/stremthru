package usenet_pool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/mnightingale/rapidyenc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// generateTestData creates deterministic test data of the specified size
func generateTestData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		// Create a pattern that's easy to verify
		data[i] = byte(i % 256)
	}
	return data
}

// encodeYenc encodes data as a yEnc article with proper headers
// Uses rapidyenc for proper encoding compatible with the decoder
func encodeYenc(data []byte, filename string, partNum, totalParts int, begin, end, total int64) []byte {
	var buf bytes.Buffer

	meta := rapidyenc.Meta{
		FileName:   filename,
		FileSize:   total,
		PartNumber: int64(partNum),
		TotalParts: int64(totalParts),
		Offset:     begin - 1, // Convert 1-based to 0-based offset
		PartSize:   int64(len(data)),
	}

	encoder, err := rapidyenc.NewEncoder(&buf, meta)
	if err != nil {
		panic(fmt.Sprintf("failed to create yEnc encoder: %v", err))
	}

	_, err = encoder.Write(data)
	if err != nil {
		panic(fmt.Sprintf("failed to encode yEnc data: %v", err))
	}

	err = encoder.Close()
	if err != nil {
		panic(fmt.Sprintf("failed to close yEnc encoder: %v", err))
	}

	return buf.Bytes()
}

// mockNZBFile represents a file for creating mock NZB documents
type mockNZBFile struct {
	Name     string
	Segments []mockSegment
	Groups   []string
}

// mockSegment represents a segment for creating mock NZB documents
type mockSegment struct {
	MessageId string
	Bytes     int64
	Number    int
}

// createMockNZB creates an NZB document from the provided files
func createMockNZB(files []mockNZBFile) *nzb.NZB {
	nzbDoc := &nzb.NZB{
		Files: make([]nzb.File, len(files)),
	}

	for i, f := range files {
		segments := make([]nzb.Segment, len(f.Segments))
		for j, s := range f.Segments {
			segments[j] = nzb.Segment{
				MessageId: s.MessageId,
				Bytes:     s.Bytes,
				Number:    s.Number,
			}
		}

		nzbDoc.Files[i] = nzb.File{
			Subject:  fmt.Sprintf("Test - \"%s\" yEnc (1/1)", f.Name),
			Groups:   f.Groups,
			Segments: segments,
		}
	}

	return nzbDoc
}

// =============================================================================
// TestFileTypeDetection
// =============================================================================

func TestFileTypeDetection(t *testing.T) {
	t.Run("MagicBytes", func(t *testing.T) {
		t.Run("RAR4", func(t *testing.T) {
			data := append(magicBytesRAR4, make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.rar")
			assert.Equal(t, FileTypeRar, ft)
		})

		t.Run("RAR5", func(t *testing.T) {
			data := append(magicBytesRAR5, make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.rar")
			assert.Equal(t, FileTypeRar, ft)
		})

		t.Run("7z", func(t *testing.T) {
			data := append(magicBytes7Zip, make([]byte, 1000)...)
			ft := DetectFileType(data, "archive.7z")
			assert.Equal(t, FileType7z, ft)
		})

		t.Run("DirectWithRarExtension", func(t *testing.T) {
			// Magic bytes take precedence - this is a video with .rar extension
			data := []byte{0x1A, 0x45, 0xDF, 0xA3} // MKV magic
			data = append(data, make([]byte, 1000)...)
			ft := DetectFileType(data, "video.rar")
			// Without RAR magic, falls back to extension
			assert.Equal(t, FileTypeRar, ft)
		})
	})

	t.Run("ExtensionBased", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected FileType
		}{
			{"movie.mkv", FileTypePlain},
			{"movie.mp4", FileTypePlain},
			{"movie.avi", FileTypePlain},
			{"movie.mov", FileTypePlain},
			{"movie.webm", FileTypePlain},
			{"archive.rar", FileTypeRar},
			{"archive.r00", FileTypeRar},
			{"archive.r01", FileTypeRar},
			{"archive.r99", FileTypeRar},
			{"archive.part01.rar", FileTypeRar},
			{"archive.part99.rar", FileTypeRar},
			{"archive.7z", FileType7z},
			{"archive.7z.001", FileType7z},
			{"archive.7z.002", FileType7z},
			{"unknown.txt", FileTypePlain},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				// Use non-magic bytes
				data := []byte("some random data")
				ft := DetectFileType(data, tc.filename)
				assert.Equal(t, tc.expected, ft, "filename: %s", tc.filename)
			})
		}
	})

	t.Run("IsVideoFile", func(t *testing.T) {
		videoFiles := []string{
			"movie.mkv", "MOVIE.MKV", "movie.mp4", "movie.avi",
			"movie.mov", "movie.wmv", "movie.flv", "movie.webm",
			"movie.m4v", "movie.mpg", "movie.mpeg", "movie.ts",
		}
		for _, f := range videoFiles {
			assert.True(t, isVideoFile(f), "should be video: %s", f)
		}

		nonVideoFiles := []string{
			"file.rar", "file.7z", "file.txt", "file.exe", "file.zip",
		}
		for _, f := range nonVideoFiles {
			assert.False(t, isVideoFile(f), "should not be video: %s", f)
		}
	})

	t.Run("GetRarPartNumber", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected int
		}{
			{"archive.rar", 0},
			{"archive.r00", 1},
			{"archive.r01", 2},
			{"archive.r99", 100},
			{"archive.part01.rar", 1},
			{"archive.part02.rar", 2},
			{"archive.part99.rar", 99},
			{"notrar.txt", -1},
			{"archive.zip", -1},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := GetRARVolumeNumber(tc.filename)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Get7zPartNumber", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected int
		}{
			{"archive.7z", 0},
			{"archive.7z.001", 1},
			{"archive.7z.002", 2},
			{"archive.7z.099", 99},
			{"not7z.txt", -1},
			{"archive.zip", -1},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := Get7zVolumeNumber(tc.filename)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("GetContentType", func(t *testing.T) {
		testCases := []struct {
			filename    string
			contentType string
		}{
			{"movie.mkv", "video/x-matroska"},
			{"movie.mp4", "video/mp4"},
			{"movie.avi", "video/x-msvideo"},
			{"movie.webm", "video/webm"},
			{"movie.mov", "video/quicktime"},
			{"movie.wmv", "video/x-ms-wmv"},
			{"movie.flv", "video/x-flv"},
			{"movie.ts", "video/mp2t"},
			{"movie.m2ts", "video/mp2t"},
			{"movie.mpg", "video/mpeg"},
			{"movie.mpeg", "video/mpeg"},
			{"movie.m4v", "video/x-m4v"},
			{"unknown.xyz", "application/octet-stream"},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := GetContentType(tc.filename)
				assert.Equal(t, tc.contentType, result)
			})
		}
	})

	t.Run("GetBaseArchiveName", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected string
		}{
			{"archive.rar", "archive.rar"},
			{"archive.r00", "archive.rar"},
			{"archive.r01", "archive.rar"},
			{"archive.part01.rar", "archive.rar"},
			{"archive.part02.rar", "archive.rar"},
			{"archive.7z", "archive.7z"},
			{"archive.7z.001", "archive.7z"},
			{"archive.7z.002", "archive.7z"},
		}

		for _, tc := range testCases {
			t.Run(tc.filename, func(t *testing.T) {
				result := GetBaseArchiveName(tc.filename)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// =============================================================================
// TestLongRangeOperations
// =============================================================================

func TestLongRangeOperations(t *testing.T) {
	t.Run("Count", func(t *testing.T) {
		r := ByteRange{Start: 0, End: 100}
		assert.Equal(t, int64(100), r.Count())

		r = ByteRange{Start: 50, End: 150}
		assert.Equal(t, int64(100), r.Count())

		r = ByteRange{Start: 0, End: 0}
		assert.Equal(t, int64(0), r.Count())
	})

	t.Run("Contains", func(t *testing.T) {
		r := ByteRange{Start: 10, End: 20}

		// Inside
		assert.True(t, r.Contains(10))
		assert.True(t, r.Contains(15))
		assert.True(t, r.Contains(19))

		// Outside
		assert.False(t, r.Contains(9))
		assert.False(t, r.Contains(20)) // End is exclusive
		assert.False(t, r.Contains(21))
	})

	t.Run("ContainsRange", func(t *testing.T) {
		r := ByteRange{Start: 0, End: 100}

		// Fully contained
		assert.True(t, r.ContainsRange(ByteRange{Start: 0, End: 100}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 10, End: 90}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 0, End: 50}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 50, End: 100}))

		// Not contained
		assert.False(t, r.ContainsRange(ByteRange{Start: -10, End: 50}))
		assert.False(t, r.ContainsRange(ByteRange{Start: 50, End: 150}))
		assert.False(t, r.ContainsRange(ByteRange{Start: -10, End: 150}))
	})

	t.Run("NewLongRangeFromSize", func(t *testing.T) {
		r := NewByteRangeFromSize(0, 100)
		assert.Equal(t, int64(0), r.Start)
		assert.Equal(t, int64(100), r.End)

		r = NewByteRangeFromSize(50, 100)
		assert.Equal(t, int64(50), r.Start)
		assert.Equal(t, int64(150), r.End)
	})
}

// =============================================================================
// TestInterpolationSearch
// =============================================================================

func TestInterpolationSearch(t *testing.T) {
	// Helper to create a mock getByteRange function for uniform segments
	createUniformSegments := func(count int, segmentSize int64) GetByteRangeFunc {
		return func(ctx context.Context, index int) (ByteRange, error) {
			start := int64(index) * segmentSize
			end := start + segmentSize
			return ByteRange{Start: start, End: end}, nil
		}
	}

	t.Run("FindAtStart", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 0, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: -1}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 0, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(0))
	})

	t.Run("FindInMiddle", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 5500, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: -1}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 5, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(5500))
	})

	t.Run("FindAtEnd", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 9999, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: -1}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 9, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(9999))
	})

	t.Run("SingleSegment", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(1, 1000)

		result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 500, SegmentCount: 1, FileSize: 1000, EstimatedSegmentIndex: -1}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 0, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(500))
	})

	t.Run("ManySegments", func(t *testing.T) {
		ctx := context.Background()
		segmentCount := 1000
		segmentSize := int64(1024)
		fileSize := int64(segmentCount) * segmentSize
		getByteRange := createUniformSegments(segmentCount, segmentSize)

		// Test finding various positions
		testPositions := []int64{0, 512, fileSize / 2, fileSize - 100}
		for _, pos := range testPositions {
			result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: pos, SegmentCount: segmentCount, FileSize: fileSize, EstimatedSegmentIndex: -1}, getByteRange)
			require.NoError(t, err, "position: %d", pos)
			assert.True(t, result.ByteRange.Contains(pos), "position %d not in range [%d, %d)", pos, result.ByteRange.Start, result.ByteRange.End)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		getByteRange := createUniformSegments(100, 1000)

		_, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 50000, SegmentCount: 100, FileSize: 100000, EstimatedSegmentIndex: -1}, getByteRange)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("OutOfBoundsNegative", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		_, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: -1, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: -1}, getByteRange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of bounds")
	})

	t.Run("OutOfBoundsPositive", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		_, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 10000, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: -1}, getByteRange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of bounds")
	})

	t.Run("EmptySegmentList", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(0, 1000)

		_, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 0, SegmentCount: 0, FileSize: 0, EstimatedSegmentIndex: -1}, getByteRange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no segments")
	})

	t.Run("WithEstimatedSegmentIndex", func(t *testing.T) {
		ctx := context.Background()
		getByteRange := createUniformSegments(10, 1000)

		// Correct estimate
		result, err := InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 5500, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: 5}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 5, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(5500))

		// Incorrect estimate (too low)
		result, err = InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 5500, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: 2}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 5, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(5500))

		// Incorrect estimate (too high)
		result, err = InterpolationSearch(ctx, InterpolationSearchParams{TargetByte: 5500, SegmentCount: 10, FileSize: 10000, EstimatedSegmentIndex: 8}, getByteRange)
		require.NoError(t, err)
		assert.Equal(t, 5, result.SegmentIndex)
		assert.True(t, result.ByteRange.Contains(5500))
	})
}

// =============================================================================
// TestYencDecoder
// =============================================================================

func TestYencDecoder(t *testing.T) {
	t.Run("DecodeSinglePart", func(t *testing.T) {
		originalData := []byte("Hello, World! This is test data for yEnc encoding.")
		encoded := encodeYenc(originalData, "test.txt", 1, 1, 1, int64(len(originalData)), int64(len(originalData)))

		decoder := NewYEncDecoder(bytes.NewReader(encoded))
		header, err := decoder.Header()
		require.NoError(t, err)

		assert.Equal(t, "test.txt", header.FileName)
		assert.Equal(t, int64(len(originalData)), header.FileSize)

		// Read all decoded data
		decoded, err := io.ReadAll(decoder)
		require.NoError(t, err)
		assert.Equal(t, originalData, decoded)
	})

	t.Run("DecodeMultiPart", func(t *testing.T) {
		totalSize := int64(100)
		part1Data := generateTestData(50)
		part2Data := generateTestData(50)

		// Part 1: bytes 1-50 (1-based)
		encoded1 := encodeYenc(part1Data, "test.bin", 1, 2, 1, 50, totalSize)
		decoder1 := NewYEncDecoder(bytes.NewReader(encoded1))
		header1, err := decoder1.Header()
		require.NoError(t, err)

		assert.Equal(t, int64(1), header1.PartNumber)
		assert.Equal(t, int64(2), header1.TotalParts)
		assert.Equal(t, int64(1), header1.Begin())
		assert.Equal(t, int64(50), header1.End())

		// Verify ByteRange converts correctly
		br := header1.ByteRange()
		assert.Equal(t, int64(0), br.Start) // 1-based to 0-based
		assert.Equal(t, int64(50), br.End)

		decoded1, err := io.ReadAll(decoder1)
		require.NoError(t, err)
		assert.Equal(t, part1Data, decoded1)

		// Part 2: bytes 51-100 (1-based)
		encoded2 := encodeYenc(part2Data, "test.bin", 2, 2, 51, 100, totalSize)
		decoder2 := NewYEncDecoder(bytes.NewReader(encoded2))
		header2, err := decoder2.Header()
		require.NoError(t, err)

		assert.Equal(t, int64(2), header2.PartNumber)
		assert.Equal(t, int64(51), header2.Begin())
		assert.Equal(t, int64(100), header2.End())
	})

	t.Run("ByteRangeConversion", func(t *testing.T) {
		header := &YEncHeader{}
		header.Offset = 0
		header.PartSize = 100

		br := header.ByteRange()
		assert.Equal(t, int64(0), br.Start, "Begin 1 should convert to Start 0")
		assert.Equal(t, int64(100), br.End, "End 100 should stay 100 (exclusive)")
		assert.Equal(t, int64(100), br.Count())
	})

	t.Run("HandleEOF", func(t *testing.T) {
		data := []byte("short")
		encoded := encodeYenc(data, "short.txt", 1, 1, 1, int64(len(data)), int64(len(data)))

		decoder := NewYEncDecoder(bytes.NewReader(encoded))
		_, err := decoder.Header()
		require.NoError(t, err)

		// Read beyond EOF
		buf := make([]byte, 1000)
		n, err := decoder.Read(buf)
		assert.Equal(t, len(data), n)

		// Next read should return EOF
		n, err = decoder.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})
}

// =============================================================================
// TestDirectFileStreaming
// =============================================================================

func TestDirectFileStreaming(t *testing.T) {
	// Note: Full streaming tests that read actual data are skipped because
	// they depend on the NNTP body reading code which has known issues
	// with dot-terminated multi-line response handling.
	// See TestBody_ReaderContent in commands_test.go for the underlying issue.

	t.Run("StreamingWithRealData", func(t *testing.T) {
		t.Skip("Skipped: depends on NNTP body reading which has known issues")
	})

	t.Run("EmptySegmentsError", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.Start(t)

		pool, err := nntp.NewPool(&nntp.PoolConfig{
			ConnectionConfig: nntp.ConnectionConfig{
				Host: server.Host(),
				Port: server.Port(),
			},
		})
		require.NoError(t, err)
		defer pool.Close()

		usenetPool := &Pool{
			providers: []*providerPool{{Pool: pool}},
		}

		ctx := t.Context()
		_, err = usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: []nzb.Segment{},
			Groups:   []string{"alt.test"},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no segments")
	})
}

// =============================================================================
// TestStreamSegmentsAPI
// =============================================================================

func TestStreamSegmentsAPI(t *testing.T) {
	t.Run("CalculatesSizeCorrectly", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.Start(t)

		pool, err := nntp.NewPool(&nntp.PoolConfig{
			ConnectionConfig: nntp.ConnectionConfig{
				Host: server.Host(),
				Port: server.Port(),
			},
		})
		require.NoError(t, err)
		defer pool.Close()

		usenetPool := &Pool{
			providers: []*providerPool{{Pool: pool}},
		}

		segments := []nzb.Segment{
			{MessageId: "seg1@test.com", Bytes: 1000, Number: 1},
			{MessageId: "seg2@test.com", Bytes: 2000, Number: 2},
			{MessageId: "seg3@test.com", Bytes: 500, Number: 3},
		}

		ctx := t.Context()
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: segments,
			Groups:   []string{"alt.test"},
		})
		require.NoError(t, err)
		defer result.Close()

		// Size should be sum of all segment bytes
		assert.Equal(t, int64(3500), result.Size)
	})

	t.Run("DefaultBufferSize", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.Start(t)

		pool, err := nntp.NewPool(&nntp.PoolConfig{
			ConnectionConfig: nntp.ConnectionConfig{
				Host: server.Host(),
				Port: server.Port(),
			},
		})
		require.NoError(t, err)
		defer pool.Close()

		usenetPool := &Pool{
			providers: []*providerPool{{Pool: pool}},
		}

		segments := []nzb.Segment{
			{MessageId: "seg1@test.com", Bytes: 1000, Number: 1},
		}

		ctx := t.Context()

		// Buffer of 0 should use default
		result, err := usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: segments,
			Groups:   []string{"alt.test"},
			Buffer:   0,
		})
		require.NoError(t, err)
		result.Close()

		// Negative buffer should use default
		result, err = usenetPool.StreamSegments(ctx, StreamSegmentsConfig{
			Segments: segments,
			Groups:   []string{"alt.test"},
			Buffer:   -1,
		})
		require.NoError(t, err)
		result.Close()
	})
}
