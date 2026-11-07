package usenet_pool

import (
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/nntp/nntptest"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestNZB(files ...nzb.File) *nzb.NZB {
	if len(files) == 0 {
		return &nzb.NZB{Files: []nzb.File{}}
	}

	nzbDoc := &nzb.NZB{
		Files: make([]nzb.File, len(files)),
	}

	for i, f := range files {
		if len(f.Groups) == 0 {
			f.Groups = []string{"alt.binaries.test"}
		}
		if f.Date == 0 {
			f.Date = time.Now().Unix()
		}
		if f.Poster == "" {
			f.Poster = "test@example.com"
		}

		nzbDoc.Files[i] = f
	}

	nzbDoc.ParseFileSubject()

	return nzbDoc
}

func createTestPool(t *testing.T, server *nntptest.Server) *Pool {
	t.Helper()

	return createTestPoolWithSize(t, server, 0)
}

// createTestPoolWithSize is createTestPool with maxSize connections to the
// server, or the nntp default when maxSize is 0.
func createTestPoolWithSize(t *testing.T, server *nntptest.Server, maxSize int32) *Pool {
	t.Helper()

	pool, err := NewPool(&Config{
		Providers: []ProviderConfig{
			{
				PoolConfig: nntp.PoolConfig{
					ConnectionConfig: nntp.ConnectionConfig{
						Host: server.Host(),
						Port: server.Port(),
					},
					MaxSize: maxSize,
				},
			},
		},
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// makeSegmentBytes returns size bytes that differ from segment to segment, so
// a read served from the wrong segment fails the comparison. makeTestBytes
// gives every segment the same bytes.
func makeSegmentBytes(num, size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(num*37 + i)
	}
	return data
}

// segmentedFile describes the test.bin that setupSegmentedFile serves.
type segmentedFile struct {
	segments    int                   // numbered 1..segments
	segmentSize int                   // bytes per segment, 50 if zero
	bufferSize  int64                 // SegmentBufferSize, 1 MB if zero
	connections int32                 // connections to the server, the nntp default if zero
	missing     []int                 // segment numbers that answer 430 No Such Article
	delayed     map[int]time.Duration // segment numbers that answer late
}

// setupSegmentedFile serves sf as test.bin, with 8 connections per stream
// whatever the environment sets.
func setupSegmentedFile(t *testing.T, sf segmentedFile) (*UsenetFile, []byte, *nntptest.Server) {
	t.Helper()

	if sf.segmentSize == 0 {
		sf.segmentSize = 50
	}
	if sf.bufferSize == 0 {
		sf.bufferSize = 1 << 20
	}

	maxConnectionPerStream := config.Newz.MaxConnectionPerStream
	config.Newz.MaxConnectionPerStream = 8
	t.Cleanup(func() {
		config.Newz.MaxConnectionPerStream = maxConnectionPerStream
	})

	totalSize := int64(sf.segments * sf.segmentSize)

	server := nntptest.NewServer(t, "200 NNTP Service Ready")
	server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")

	data := make([]byte, 0, totalSize)
	segments := make([]nzb.Segment, sf.segments)
	for i := range sf.segments {
		num := i + 1
		messageId := fmt.Sprintf("msg%d@test", num)
		segmentData := makeSegmentBytes(num, sf.segmentSize)
		data = append(data, segmentData...)
		segments[i] = nzb.Segment{MessageId: messageId, Bytes: int64(sf.segmentSize), Number: num}

		if slices.Contains(sf.missing, num) {
			setupServerArticleNotFound(server, messageId)
			continue
		}
		yencData := encodeYenc(segmentData, "test.bin", num, sf.segments, totalSize, int64(i*sf.segmentSize)+1)
		server.SetResponseWithDelay("BODY <"+messageId+">", sf.delayed[num], "222 0 <"+messageId+">", []string{string(yencData)})
	}
	server.Start(t)

	pool := createTestPoolWithSize(t, server, sf.connections)

	nzbDoc := createTestNZB(nzb.File{
		Subject:  fmt.Sprintf(`Test - "test.bin" yEnc (1/%d)`, sf.segments),
		Segments: segments,
		Groups:   []string{"alt.binaries.test"},
	})

	ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
		NZB:               nzbDoc,
		Pool:              pool,
		SegmentBufferSize: sf.bufferSize,
	})
	f, err := ufs.Open("test.bin")
	require.NoError(t, err)
	t.Cleanup(func() {
		f.Close()
	})

	uf, ok := f.(*UsenetFile)
	require.True(t, ok)

	return uf, data, server
}

func TestUsenetFS_Open(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "existing.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB: nzbDoc,
		})

		f, err := ufs.Open("nonexistent.txt")
		assert.Nil(t, f)
		assert.Error(t, err)

		var pathErr *fs.PathError
		require.ErrorAs(t, err, &pathErr)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "nonexistent.txt", pathErr.Path)
		assert.ErrorIs(t, pathErr.Err, fs.ErrNotExist)
	})

	t.Run("EmptyName", func(t *testing.T) {
		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "file.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB: nzbDoc,
		})

		f, err := ufs.Open("")
		assert.Nil(t, f)
		assert.Error(t, err)

		var pathErr *fs.PathError
		require.ErrorAs(t, err, &pathErr)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, ".", pathErr.Path)
		assert.ErrorIs(t, pathErr.Err, fs.ErrNotExist)
	})

	t.Run("ValidFile", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		pool := createTestPool(t, server)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		require.NotNil(t, f)
		f.Close()

		uf, ok := f.(*UsenetFile)
		assert.True(t, ok)
		assert.NotNil(t, uf)
	})

	t.Run("PathCleaning", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "file.txt" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
		})

		pool := createTestPool(t, server)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		testCases := []struct {
			path string
		}{
			{"./file.txt"},
			{"dir/../file.txt"},
			{"./dir/../file.txt"},
		}

		for _, tc := range testCases {
			t.Run(tc.path, func(t *testing.T) {
				f, err := ufs.Open(tc.path)
				require.NoError(t, err, "path: %s", tc.path)
				require.NotNil(t, f)
				f.Close()
			})
		}
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencEncoded)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(
			nzb.File{
				Subject: `Test - "file1.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg1@test", Bytes: 100, Number: 1},
				},
			},
			nzb.File{
				Subject: `Test - "file2.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg2@test", Bytes: 200, Number: 1},
				},
			},
			nzb.File{
				Subject: `Test - "file3.bin" yEnc (1/1)`,
				Segments: []nzb.Segment{
					{MessageId: "msg3@test", Bytes: 300, Number: 1},
				},
			},
		)

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		for _, name := range []string{"file1.bin", "file2.bin", "file3.bin"} {
			f, err := ufs.Open(name)
			require.NoError(t, err, "file: %s", name)
			require.NotNil(t, f)
			f.Close()
		}
	})
}

func TestUsenetFile_Stat(t *testing.T) {
	t.Run("ReturnsFileInfo", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 2, 3000, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/2)`,
			Date:    1234567890,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 1000, Number: 1},
				{MessageId: "msg2@test", Bytes: 2000, Number: 2},
			},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		fi, err := f.Stat()
		require.NoError(t, err)
		assert.Equal(t, "test.bin", fi.Name())
		assert.Equal(t, int64(3000), fi.Size())
		assert.Equal(t, fs.FileMode(0644), fi.Mode())
		assert.Equal(t, time.Unix(1234567890, 0), fi.ModTime())
		assert.False(t, fi.IsDir())
		assert.Nil(t, fi.Sys())
	})
}

func TestUsenetFile_Read(t *testing.T) {
	t.Run("SingleSegment", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		buf := make([]byte, 200)
		n, err := f.Read(buf)
		assert.Equal(t, 100, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData, buf[:n])

		n, err = f.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("MultipleReads", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		originalData := makeTestBytes(100)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, 100, 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})

		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		// Read in small chunks
		buf := make([]byte, 30)
		var result []byte

		for {
			n, err := f.Read(buf)
			if n > 0 {
				result = append(result, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}

		assert.Equal(t, originalData, result)
	})

	t.Run("ServesBytesBeforeAMissingArticle", func(t *testing.T) {
		// Segment 8, bytes [350, 400), is missing. Segment 7 answers late,
		// so the 430 always arrives first.
		uf, data, _ := setupSegmentedFile(t, segmentedFile{
			segments: 10,
			missing:  []int{8},
			delayed:  map[int]time.Duration{7: 300 * time.Millisecond},
		})

		buf := make([]byte, 10)
		n, err := uf.Read(buf)
		require.NoError(t, err)
		require.Equal(t, data[:10], buf[:n])

		rest, err := io.ReadAll(uf)
		require.ErrorIs(t, err, ErrArticleNotFound)
		require.Equal(t, data[10:350], rest)

		// The error stays rather than turning into EOF.
		_, err = uf.Read(buf)
		require.ErrorIs(t, err, ErrArticleNotFound)
	})
}

func TestUsenetFile_Seek(t *testing.T) {
	setupSeekTest := func(t *testing.T, dataSize int) (*UsenetFile, []byte, func()) {
		t.Helper()

		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		originalData := makeTestBytes(dataSize)
		yencEncoded := encodeYenc(originalData, "test.bin", 1, 1, int64(dataSize), 1)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencEncoded)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/1)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: int64(dataSize), Number: 1},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		cleanup := func() {
			f.Close()
		}

		return uf, originalData, cleanup
	}

	t.Run("SeekStart", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Seek to position 50
		pos, err := uf.Seek(50, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(50), pos)

		// Read from that position
		buf := make([]byte, 10)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[50:60], buf[:n])
	})

	t.Run("SeekCurrent", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Read 20 bytes to position at 20
		buf := make([]byte, 20)
		_, err := uf.Read(buf)
		require.NoError(t, err)

		// Seek forward 10 from current (20 + 10 = 30)
		pos, err := uf.Seek(10, io.SeekCurrent)
		require.NoError(t, err)
		assert.Equal(t, int64(30), pos)

		// Read from that position
		n, err := uf.Read(buf[:10])
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[30:40], buf[:n])
	})

	t.Run("SeekEnd", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		// Seek to 10 bytes before end
		pos, err := uf.Seek(-10, io.SeekEnd)
		require.NoError(t, err)
		assert.Equal(t, int64(90), pos)

		// Read from that position
		buf := make([]byte, 20)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, originalData[90:], buf[:n])
	})

	t.Run("SeekThenRead", func(t *testing.T) {
		uf, originalData, cleanup := setupSeekTest(t, 200)
		defer cleanup()

		// Test multiple seek and read operations
		// Each test seeks to a position and reads 10 bytes
		testCases := []struct {
			offset int64
			whence int
			want   int64
		}{
			{0, io.SeekStart, 0},
			{50, io.SeekStart, 50},
			{100, io.SeekStart, 100},
			{-50, io.SeekEnd, 150},
		}

		for _, tc := range testCases {
			pos, err := uf.Seek(tc.offset, tc.whence)
			require.NoError(t, err)
			assert.Equal(t, tc.want, pos)

			// Verify read returns correct data
			buf := make([]byte, 10)
			n, err := uf.Read(buf)
			assert.Equal(t, 10, n)
			assert.NoError(t, err)
			assert.Equal(t, originalData[tc.want:tc.want+10], buf[:n])
		}
	})

	t.Run("SeekNegativePosition", func(t *testing.T) {
		uf, _, cleanup := setupSeekTest(t, 100)
		defer cleanup()

		_, err := uf.Seek(-10, io.SeekStart)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "negative position")
	})

	t.Run("SeekSkipsIntermediateSegments", func(t *testing.T) {
		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		segment1Data := makeTestBytes(100)
		segment2Data := makeTestBytes(100)
		segment3Data := makeTestBytes(100)
		segment4Data := makeTestBytes(100)

		yencSeg1 := encodeYenc(segment1Data, "test.bin", 1, 4, 400, 1)
		yencSeg2 := encodeYenc(segment2Data, "test.bin", 2, 4, 400, 101)
		yencSeg3 := encodeYenc(segment3Data, "test.bin", 3, 4, 400, 201)
		yencSeg4 := encodeYenc(segment4Data, "test.bin", 4, 4, 400, 301)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencSeg1)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencSeg2)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencSeg3)})
		server.SetResponse("BODY <msg4@test>", "222 0 <msg4@test>", []string{string(yencSeg4)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/4)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 100, Number: 1},
				{MessageId: "msg2@test", Bytes: 100, Number: 2},
				{MessageId: "msg3@test", Bytes: 100, Number: 3},
				{MessageId: "msg4@test", Bytes: 100, Number: 4},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:               nzbDoc,
			Pool:              pool,
			SegmentBufferSize: 1,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		// Step 1: Read 10 bytes from start (should fetch segment 1)
		buf := make([]byte, 10)
		n, err := uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, segment1Data[:10], buf[:n])

		// Step 2: Seek to position 350 (inside segment 4, skipping segment 2,3)
		pos, err := uf.Seek(350, io.SeekStart)
		require.NoError(t, err)
		assert.Equal(t, int64(350), pos)

		// Step 3: Read 10 bytes from position 350 (should fetch segment 4)
		n, err = uf.Read(buf)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		// Position 350 is offset 50 in segment 4 (350 - 300 = 50)
		assert.Equal(t, segment4Data[50:60], buf[:n])

		// Assert segment 1, 2 (buffered) and 4 were fetched, but not segment 3
		cmds := server.GetRequestCommands()
		assert.True(t, cmds.HasCommand("BODY <msg1@test>"), "segment 1 should be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg2@test>"), "segment 2 should not be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg3@test>"), "segment 3 should not be fetched")
		assert.True(t, cmds.HasCommand("BODY <msg4@test>"), "segment 4 should be fetched")
	})
}

func TestUsenetFile_ReadAt(t *testing.T) {
	setup := func(t *testing.T) (*UsenetFile, []byte, *nntptest.Server) {
		t.Helper()

		server := nntptest.NewServer(t, "200 NNTP Service Ready")

		// Create 4 segments of 50 bytes each (total: 200 bytes)
		segment1Data := makeTestBytes(50)
		segment2Data := makeTestBytes(50)
		segment3Data := makeTestBytes(50)
		segment4Data := makeTestBytes(50)

		// Combine all segments to create the complete file data
		data := make([]byte, 0, 200)
		data = append(data, segment1Data...)
		data = append(data, segment2Data...)
		data = append(data, segment3Data...)
		data = append(data, segment4Data...)

		// Encode each segment as yEnc
		yencSeg1 := encodeYenc(segment1Data, "test.bin", 1, 4, 200, 1)
		yencSeg2 := encodeYenc(segment2Data, "test.bin", 2, 4, 200, 51)
		yencSeg3 := encodeYenc(segment3Data, "test.bin", 3, 4, 200, 101)
		yencSeg4 := encodeYenc(segment4Data, "test.bin", 4, 4, 200, 151)

		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")
		server.SetResponse("BODY <msg1@test>", "222 0 <msg1@test>", []string{string(yencSeg1)})
		server.SetResponse("BODY <msg2@test>", "222 0 <msg2@test>", []string{string(yencSeg2)})
		server.SetResponse("BODY <msg3@test>", "222 0 <msg3@test>", []string{string(yencSeg3)})
		server.SetResponse("BODY <msg4@test>", "222 0 <msg4@test>", []string{string(yencSeg4)})
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/4)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 50, Number: 1},
				{MessageId: "msg2@test", Bytes: 50, Number: 2},
				{MessageId: "msg3@test", Bytes: 50, Number: 3},
				{MessageId: "msg4@test", Bytes: 50, Number: 4},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		t.Cleanup(func() {
			f.Close()
		})

		return uf, data, server
	}

	t.Run("ReadAtOffset", func(t *testing.T) {
		uf, data, _ := setup(t)

		buf := make([]byte, 20)
		n, err := uf.ReadAt(buf, 50)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[50:70], buf[:n])

		n, err = uf.ReadAt(buf, 100)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[100:120], buf[:n])
	})

	t.Run("ReadAtDoesNotAffectPosition", func(t *testing.T) {
		uf, data, _ := setup(t)

		buf := make([]byte, 20)
		n, err := uf.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 20, n)
		assert.Equal(t, data[0:20], buf[:n])

		n, err = uf.ReadAt(buf, 100)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[100:120], buf[:n])

		n, err = uf.Read(buf)
		assert.Equal(t, 20, n)
		assert.NoError(t, err)
		assert.Equal(t, data[20:40], buf[:n])
	})

	t.Run("ReadAtSkipsIntermediateSegments", func(t *testing.T) {
		uf, data, server := setup(t)

		buf := make([]byte, 10)
		n, err := uf.ReadAt(buf, 160)
		assert.Equal(t, 10, n)
		assert.NoError(t, err)
		assert.Equal(t, data[160:170], buf[:n])

		cmds := server.GetRequestCommands()
		assert.True(t, cmds.HasCommand("BODY <msg1@test>"), "segment 1 should be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg2@test>"), "segment 2 should NOT be fetched")
		assert.False(t, cmds.HasCommand("BODY <msg3@test>"), "segment 3 should NOT be fetched")
		assert.True(t, cmds.HasCommand("BODY <msg4@test>"), "segment 4 should be fetched")
	})

	t.Run("SequentialReadAtFetchesEachSegmentOnce", func(t *testing.T) {
		uf, data, server := setupSegmentedFile(t, segmentedFile{segments: 4})

		buf := make([]byte, 10)
		for off := int64(0); off < 200; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, 10, n)
			require.Equal(t, data[off:off+10], buf, "offset %d", off)
		}

		// NewFileStream already fetched segment 1 once, to learn the file size.
		assert.LessOrEqual(t, countBodyRequests(server, "msg1@test"), 2)
		for _, messageId := range []string{"msg2@test", "msg3@test", "msg4@test"} {
			assert.Equal(t, 1, countBodyRequests(server, messageId), messageId)
		}
	})

	t.Run("SequentialReadAtReadsAhead", func(t *testing.T) {
		uf, data, server := setupSegmentedFile(t, segmentedFile{segments: 12})

		// The calls after the first one read two segments' worth, 100 bytes.
		buf := make([]byte, 10)
		for off := int64(0); off < 110; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, 10, n)
			require.Equal(t, data[off:off+10], buf, "offset %d", off)
		}

		// The stream now reads ahead to the end of the file without further calls.
		require.Eventually(t, func() bool {
			return countBodyRequests(server, "msg12@test") > 0
		}, 3*time.Second, 10*time.Millisecond, "segment 12 should be fetched ahead of the reader")
	})

	t.Run("ReadAtReadAheadStaysWithinBufferSize", func(t *testing.T) {
		// A 150-byte buffer holds three 50-byte segments.
		uf, _, server := setupSegmentedFile(t, segmentedFile{segments: 12, bufferSize: 150})

		buf := make([]byte, 10)
		for off := int64(0); off < 110; off += 10 {
			_, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
		}

		// The reader has taken segment 3, and segments 4-6 fill the buffer.
		require.Eventually(t, func() bool {
			return countBodyRequests(server, "msg6@test") > 0
		}, 3*time.Second, 10*time.Millisecond, "segment 6 should be fetched ahead of the reader")
		time.Sleep(200 * time.Millisecond)
		assert.Zero(t, countBodyRequests(server, "msg7@test"), "segment 7 is beyond the buffer")
	})

	t.Run("HeaderReadsDoNotReadAhead", func(t *testing.T) {
		uf, data, server := setupSegmentedFile(t, segmentedFile{segments: 12, segmentSize: 5000})

		// Opening a 7z archive reads 4102 bytes at 0 to find the signature,
		// then the 12-byte signature header at 0 and the 20-byte start
		// header at 12 (sevenzip reader.go, findSignature and init).
		for _, r := range []struct{ off, size int64 }{{0, 4102}, {0, 12}, {12, 20}} {
			buf := make([]byte, r.size)
			n, err := uf.ReadAt(buf, r.off)
			require.NoError(t, err, "offset %d", r.off)
			require.Equal(t, data[r.off:r.off+r.size], buf[:n], "offset %d", r.off)
		}

		time.Sleep(200 * time.Millisecond)
		assert.Zero(t, countBodyRequests(server, "msg4@test"), "segment 4 should not be fetched")
	})

	t.Run("OneLargeReadAtDoesNotReadAhead", func(t *testing.T) {
		uf, data, server := setupSegmentedFile(t, segmentedFile{segments: 12})

		// 200 bytes is four segments, but in a single call.
		buf := make([]byte, 200)
		n, err := uf.ReadAt(buf, 0)
		require.NoError(t, err)
		require.Equal(t, data[:200], buf[:n])

		time.Sleep(200 * time.Millisecond)
		assert.Zero(t, countBodyRequests(server, "msg10@test"), "segment 10 should not be fetched")
	})

	t.Run("ReadAtServesBytesBeforeAMissingArticle", func(t *testing.T) {
		// Segment 8, bytes [350, 400), is missing. Segment 7 answers late,
		// so the 430 always arrives first.
		uf, data, _ := setupSegmentedFile(t, segmentedFile{
			segments: 10,
			missing:  []int{8},
			delayed:  map[int]time.Duration{7: 300 * time.Millisecond},
		})

		buf := make([]byte, 10)
		for off := int64(0); off < 350; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, data[off:off+10], buf[:n], "offset %d", off)
		}

		_, err := uf.ReadAt(buf, 350)
		require.ErrorIs(t, err, ErrArticleNotFound)

		// The next call at that offset fails the same way.
		_, err = uf.ReadAt(buf, 350)
		require.ErrorIs(t, err, ErrArticleNotFound)
	})

	t.Run("ConcurrentReadAt", func(t *testing.T) {
		// More connections than articles, so no fetch waits for one. The
		// pool shares a fetch of one article between readers, on the
		// context of the reader that started it: a fetch still waiting for
		// a connection when that reader's stream closes fails for all of
		// them. ReadAt serves such a call again from a stream of its own;
		// this test checks the bytes, not that race.
		uf, data, _ := setupSegmentedFile(t, segmentedFile{segments: 12, connections: 16})

		// Four overlapping sequential runs: [0, 300), [100, 400), [200, 500)
		// and [300, 600).
		var wg sync.WaitGroup
		for g := range 4 {
			wg.Go(func() {
				buf := make([]byte, 10)
				for off := int64(g * 100); off < int64(g*100+300); off += 10 {
					n, err := uf.ReadAt(buf, off)
					if !assert.NoError(t, err, "offset %d", off) {
						return
					}
					assert.Equal(t, data[off:off+10], buf[:n], "offset %d", off)
				}
			})
		}
		wg.Wait()
	})

	t.Run("CloseDoesNotWaitForReadAt", func(t *testing.T) {
		// Segment 3 holds bytes [100, 150) and answers late.
		uf, data, server := setupSegmentedFile(t, segmentedFile{
			segments: 4,
			delayed:  map[int]time.Duration{3: 3 * time.Second},
		})

		// Read [0, 100), so the next call at 100 continues where these
		// ended, and the read-ahead has already asked for segment 3.
		buf := make([]byte, 10)
		for off := int64(0); off < 100; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, data[off:off+10], buf[:n], "offset %d", off)
		}
		require.Eventually(t, func() bool {
			return countBodyRequests(server, "msg3@test") > 0
		}, 3*time.Second, 10*time.Millisecond)

		done := make(chan error, 1)
		go func() {
			_, err := uf.ReadAt(make([]byte, 10), 100)
			done <- err
		}()
		// Let the call start waiting for segment 3. Had Close come first,
		// the call would still have to fail at once.
		time.Sleep(100 * time.Millisecond)

		start := time.Now()
		require.NoError(t, uf.Close())
		assert.Less(t, time.Since(start), 500*time.Millisecond, "Close should not wait for the ReadAt call")

		select {
		case err := <-done:
			require.Error(t, err)
			assert.NotErrorIs(t, err, io.EOF)
			assert.Less(t, time.Since(start), 2*time.Second, "the ReadAt call should not wait for segment 3 after Close")
		case <-time.After(5 * time.Second):
			t.Fatal("ReadAt did not return after Close")
		}

		_, err := uf.ReadAt(make([]byte, 10), 0)
		require.Error(t, err)
		assert.NotErrorIs(t, err, io.EOF)
	})

	t.Run("ReadAtLocatesEachCallForDuplicateSegmentNumbers", func(t *testing.T) {
		// Part 2 was posted twice, so the NZB lists segment number 2 twice.
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")

		parts := [][]byte{makeSegmentBytes(1, 50), makeSegmentBytes(2, 50), makeSegmentBytes(3, 50)}
		data := slices.Concat(parts...)
		for _, seg := range []struct {
			messageId string
			part      int
		}{{"msg1@test", 1}, {"msg2@test", 2}, {"msg2b@test", 2}, {"msg3@test", 3}} {
			yencData := encodeYenc(parts[seg.part-1], "test.bin", seg.part, 3, 150, int64(seg.part-1)*50+1)
			server.SetResponse("BODY <"+seg.messageId+">", "222 0 <"+seg.messageId+">", []string{string(yencData)})
		}
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/3)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 50, Number: 1},
				{MessageId: "msg2@test", Bytes: 50, Number: 2},
				{MessageId: "msg2b@test", Bytes: 50, Number: 2},
				{MessageId: "msg3@test", Bytes: 50, Number: 3},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		// A stream running from offset 0 would serve part 2 twice.
		buf := make([]byte, 10)
		for off := int64(0); off < 150; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, data[off:off+10], buf[:n], "offset %d", off)
		}

		// After Close, a call fails rather than reporting the end of the file.
		require.NoError(t, f.Close())
		_, err = uf.ReadAt(buf, 0)
		require.Error(t, err)
		assert.NotErrorIs(t, err, io.EOF)
	})

	t.Run("ReadAtFallsBackWhenASegmentBodyDoesNotMatchItsRange", func(t *testing.T) {
		// Segment 1's "=ypart" line claims bytes [0, 40000), but its body
		// decodes to 50000 bytes, as its "=yend size" says. The header is
		// read from the first 32 KB, before the trailer, so the segment's
		// range comes from "=ypart". A stream per call locates every offset
		// by those ranges, so bytes [40000, 50000) come from segment 2.
		const size = 40000
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")

		parts := [][]byte{makeSegmentBytes(1, size+10000), makeSegmentBytes(2, size), makeSegmentBytes(3, size)}
		for i, part := range parts {
			messageId := fmt.Sprintf("msg%d@test", i+1)
			yencData := string(encodeYenc(part, "test.bin", i+1, 3, 3*size, int64(i*size)+1))
			if i == 0 {
				require.Contains(t, yencData, "end=50000")
				yencData = strings.Replace(yencData, "end=50000", "end=40000", 1)
			}
			server.SetResponse("BODY <"+messageId+">", "222 0 <"+messageId+">", []string{yencData})
		}
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/3)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: size, Number: 1},
				{MessageId: "msg2@test", Bytes: size, Number: 2},
				{MessageId: "msg3@test", Bytes: size, Number: 3},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		want := slices.Concat(parts[0][:size], parts[1], parts[2])
		buf := make([]byte, 1000)
		for off := int64(0); off < 3*size; off += 1000 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, want[off:off+1000], buf[:n], "offset %d", off)
		}
	})

	t.Run("ReadAtFallsBackWhenSegmentRangesDoNotJoin", func(t *testing.T) {
		// Segment 2's yEnc range starts at 60, not 50, so no segment holds
		// bytes [50, 60), although the numbers run 1..3.
		server := nntptest.NewServer(t, "200 NNTP Service Ready")
		server.SetResponse("GROUP alt.binaries.test", "211 1 1 1 alt.binaries.test")

		parts := [][]byte{makeSegmentBytes(1, 50), makeSegmentBytes(2, 50), makeSegmentBytes(3, 50)}
		begins := []int64{1, 61, 111}
		for i, part := range parts {
			messageId := fmt.Sprintf("msg%d@test", i+1)
			yencData := encodeYenc(part, "test.bin", i+1, 3, 160, begins[i])
			server.SetResponse("BODY <"+messageId+">", "222 0 <"+messageId+">", []string{string(yencData)})
		}
		server.Start(t)

		pool := createTestPool(t, server)

		nzbDoc := createTestNZB(nzb.File{
			Subject: `Test - "test.bin" yEnc (1/3)`,
			Segments: []nzb.Segment{
				{MessageId: "msg1@test", Bytes: 50, Number: 1},
				{MessageId: "msg2@test", Bytes: 50, Number: 2},
				{MessageId: "msg3@test", Bytes: 50, Number: 3},
			},
			Groups: []string{"alt.binaries.test"},
		})

		ufs := NewUsenetFS(t.Context(), &UsenetFSConfig{
			NZB:  nzbDoc,
			Pool: pool,
		})
		f, err := ufs.Open("test.bin")
		require.NoError(t, err)
		defer f.Close()

		uf, ok := f.(*UsenetFile)
		require.True(t, ok)

		buf := make([]byte, 10)
		for off := int64(0); off < 50; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			require.Equal(t, parts[0][off:off+10], buf[:n], "offset %d", off)
		}

		// A stream running on from offset 0 would serve segment 2's first
		// bytes here. A call located on its own finds no segment.
		_, err = uf.ReadAt(buf, 50)
		require.Error(t, err)

		for off := int64(60); off < 160; off += 10 {
			n, err := uf.ReadAt(buf, off)
			require.NoError(t, err, "offset %d", off)
			part := parts[1+(off-60)/50]
			require.Equal(t, part[(off-60)%50:][:10], buf[:n], "offset %d", off)
		}
	})
}
