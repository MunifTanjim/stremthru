package usenet_pool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

type ByteRange struct {
	Start int64 // inclusive
	End   int64 // exclusive
}

func (r ByteRange) Count() int64 {
	return r.End - r.Start
}

func (r ByteRange) Contains(byteIdx int64) bool {
	return r.Start <= byteIdx && byteIdx < r.End
}

func (r ByteRange) ContainsRange(other ByteRange) bool {
	return r.Start <= other.Start && other.End <= r.End
}

func NewByteRangeFromSize(start, size int64) ByteRange {
	return ByteRange{Start: start, End: start + size}
}

var (
	_ io.Reader   = (*FileStream)(nil)
	_ io.ReaderAt = (*FileStream)(nil)
	_ io.Seeker   = (*FileStream)(nil)
	_ io.Closer   = (*FileStream)(nil)
)

var fileLog = logger.Scoped("usenet/pool/file_stream")

type FileStream struct {
	file             *nzb.File
	fileSize         int64
	avgSegmentSize   int64
	segmentSizeRatio float64

	pool               *Pool
	bufferSegmentCount int

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	position   int64
	partStream *FilePartStream

	closed bool
}

func NewFileStream(
	pool *Pool,
	file *nzb.File,
	bufferSegmentCount int,
) (*FileStream, error) {
	ctx := context.Background()

	firstSegment, err := pool.fetchFirstSegment(ctx, file)
	if err != nil {
		return nil, err
	}
	fileSize := firstSegment.FileSize()

	fileLog.Trace("file stream - created", "segment_count", file.SegmentCount(), "file_size", fileSize, "buffer_segment_count", bufferSegmentCount)

	avgSegmentSize := int64(0)
	if file.SegmentCount() > 0 {
		avgSegmentSize = fileSize / int64(file.SegmentCount())
	}

	segmentSizeRatio := float64(1)
	if totalSegmentBytes := file.TotalSize(); totalSegmentBytes > 0 {
		segmentSizeRatio = float64(fileSize) / float64(totalSegmentBytes)
	}

	ctx, cancel := context.WithCancel(ctx)

	return &FileStream{
		file:             file,
		fileSize:         fileSize,
		avgSegmentSize:   avgSegmentSize,
		segmentSizeRatio: segmentSizeRatio,

		pool:               pool,
		bufferSegmentCount: bufferSegmentCount,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (s *FileStream) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, errors.New("file stream is closed")
	}

	if s.position >= s.fileSize {
		return 0, io.EOF
	}

	if s.partStream == nil {
		stream, err := s.createPartStream(s.position, s.bufferSegmentCount)
		if err != nil {
			return 0, err
		}
		s.partStream = stream
	}

	n, err = s.partStream.Read(p)
	s.position += int64(n)
	return n, err
}

func (s *FileStream) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.fileSize {
		return 0, io.EOF
	}

	bufferSegmentCount := max(int(math.Ceil(float64(len(p))/float64(s.avgSegmentSize))), 1)
	stream, err := s.createPartStream(off, bufferSegmentCount)
	if err != nil {
		return 0, err
	}
	defer stream.Close()

	return io.ReadFull(stream, p)
}

func (s *FileStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileLog.Trace("file stream - seek", "offset", offset, "whence", whence)

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.position + offset
	case io.SeekEnd:
		newPos = s.fileSize + offset
	default:
		return s.position, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return s.position, fmt.Errorf("negative position: %d", newPos)
	}
	if newPos > s.fileSize {
		newPos = s.fileSize
	}

	if newPos != s.position {
		fileLog.Trace("file stream - seek position changed", "old_position", s.position, "new_position", newPos, "whence", whence)
		if s.partStream != nil {
			s.partStream.Close()
			s.partStream = nil
		}
		s.position = newPos
	}

	return s.position, nil
}

func (s *FileStream) Size() int64 {
	return s.fileSize
}

func (s *FileStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	s.cancel()
	if s.partStream != nil {
		return s.partStream.Close()
	}
	return nil
}

func (s *FileStream) createPartStream(startPos int64, bufferSegmentCount int) (*FilePartStream, error) {
	fileLog.Trace("create part stream - start", "position", startPos)

	if startPos == 0 {
		return NewFilePartStream(s.pool, s.file.Segments, s.file.Groups, bufferSegmentCount), nil
	}

	result, err := InterpolationSearch(
		s.ctx,
		InterpolationSearchParams{
			TargetByte:            startPos,
			SegmentCount:          s.file.SegmentCount(),
			FileSize:              s.fileSize,
			EstimatedSegmentIndex: s.estimateSegmentIndex(startPos),
		},
		func(ctx context.Context, index int) (ByteRange, error) {
			return s.getSegmentByteRange(ctx, index)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find segment for position %d: %w", startPos, err)
	}

	fileLog.Trace("create part stream - found segment", "segment_idx", result.SegmentIndex, "byte_range", fmt.Sprintf("[%d, %d)", result.ByteRange.Start, result.ByteRange.End))

	stream := NewFilePartStream(s.pool, s.file.Segments[result.SegmentIndex:], s.file.Groups, s.bufferSegmentCount)

	skipBytes := startPos - result.ByteRange.Start
	if skipBytes > 0 {
		fileLog.Trace("create part stream - skipping bytes", "skip_bytes", skipBytes)
		if _, err := io.CopyN(io.Discard, stream, skipBytes); err != nil {
			if err == io.EOF {
				return stream, nil
			}
			stream.Close()
			return nil, fmt.Errorf("failed to skip %d bytes: %w", skipBytes, err)
		}
	}

	return stream, nil
}

func (s *FileStream) getSegmentByteRange(ctx context.Context, index int) (ByteRange, error) {
	segment := &s.file.Segments[index]

	fileLog.Trace("file stream - get segment byte range", "segment_num", segment.Number, "message_id", segment.MessageId)

	data, err := s.pool.fetchSegment(ctx, segment, s.file.Groups)
	if err != nil {
		return ByteRange{}, err
	}

	byteRange := data.ByteRange()
	fileLog.Trace("file stream - segment byte range", "segment_num", segment.Number, "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))

	return byteRange, nil
}

func (s *FileStream) estimateSegmentIndex(targetByte int64) int {
	var offset int64
	for i := range s.file.Segments {
		segBytes := s.file.Segments[i].Bytes
		if segBytes <= 0 {
			continue
		}
		estimatedDecodedBytes := int64(float64(segBytes) * s.segmentSizeRatio)
		if targetByte < offset+estimatedDecodedBytes {
			return i
		}
		offset += estimatedDecodedBytes
	}
	return len(s.file.Segments) - 1
}
