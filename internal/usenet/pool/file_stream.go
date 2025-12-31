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

var (
	_ io.Reader   = (*FileStream)(nil)
	_ io.ReaderAt = (*FileStream)(nil)
	_ io.Seeker   = (*FileStream)(nil)
	_ io.Closer   = (*FileStream)(nil)
)

var fileLog = logger.Scoped("nntp/file_stream")

type FileStream struct {
	file               *nzb.File
	fileSize           int64
	segmentSize        int64
	segmentBytesRatio  float64 // fileSize / totalSegmentBytes
	pool               *Pool
	bufferSegmentCount int
	cache              *SegmentCache

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	position   int64
	partStream *FilePartStream

	closed bool
}

func NewFileStream(
	ctx context.Context,
	pool *Pool,
	file *nzb.File,
	fileSize int64,
	bufferSegmentCount int,
	cache *SegmentCache,
) *FileStream {
	fileLog.Trace("file stream created", "segment_count", file.SegmentCount(), "file_size", fileSize, "buffer_segment_count", bufferSegmentCount)

	ctx, cancel := context.WithCancel(ctx)

	segmentSize := int64(0)
	if file.SegmentCount() > 0 {
		segmentSize = fileSize / int64(file.SegmentCount())
	}

	totalSegmentBytes := file.TotalSize()
	segmentBytesRatio := float64(1)
	if totalSegmentBytes > 0 {
		segmentBytesRatio = float64(fileSize) / float64(totalSegmentBytes)
	}

	return &FileStream{
		file:               file,
		fileSize:           fileSize,
		segmentSize:        segmentSize,
		segmentBytesRatio:  segmentBytesRatio,
		pool:               pool,
		bufferSegmentCount: bufferSegmentCount,
		cache:              cache,

		ctx:    ctx,
		cancel: cancel,
	}
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

	bufferSegmentCount := max(int(math.Ceil(float64(len(p))/float64(s.segmentSize))), 1)
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

	fileLog.Trace("Seek called", "offset", offset, "whence", whence)

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
		fileLog.Trace("Seek changing position", "old_position", s.position, "new_position", newPos, "whence", whence)
		// Close current stream and reset
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

func (s *FileStream) findSegmentByOffset(ctx context.Context, targetByte int64) (int, ByteRange, error) {
	var offset int64
	for i := range s.file.Segments {
		segBytes := s.file.Segments[i].Bytes
		if segBytes <= 0 {
			return 0, ByteRange{}, fmt.Errorf("segment %d has no bytes", i)
		}

		// Scale segment bytes by ratio to estimate decoded size
		estimatedDecodedBytes := int64(float64(segBytes) * s.segmentBytesRatio)
		nextOffset := offset + estimatedDecodedBytes

		if targetByte >= offset && targetByte < nextOffset {
			// Found estimated segment, now get actual byte range
			byteRange, err := s.getSegmentByteRange(ctx, i)
			if err != nil {
				return 0, ByteRange{}, err
			}

			// Verify actual range contains target
			if byteRange.Contains(targetByte) {
				return i, byteRange, nil
			}

			// Estimate was wrong, return error to trigger InterpolationSearch
			return 0, ByteRange{}, fmt.Errorf("estimate incorrect: segment %d range [%d, %d) does not contain %d", i, byteRange.Start, byteRange.End, targetByte)
		}
		offset = nextOffset
	}

	return 0, ByteRange{}, fmt.Errorf("no segment found for byte %d", targetByte)
}

func (s *FileStream) createPartStream(startPos int64, bufferSegmentCount int) (*FilePartStream, error) {
	fileLog.Trace("create part stream - start", "position", startPos)

	if startPos == 0 {
		return NewFilePartStream(s.ctx, s.pool, s.file.Segments, s.file.Groups, bufferSegmentCount, s.cache), nil
	}

	var startIndex int
	var byteRange ByteRange

	if idx, br, err := s.findSegmentByOffset(s.ctx, startPos); err == nil {
		fileLog.Trace("create part stream - using estimate", "start_position", startPos)
		startIndex = idx
		byteRange = br
		fileLog.Trace("create part stream - found segment", "segment_index", startIndex, "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))
	} else {
		fileLog.Trace("create part stream - using interpolation search", "start_position", startPos)

		result, err := InterpolationSearch(
			s.ctx,
			startPos,
			s.file.SegmentCount(),
			s.fileSize,
			func(ctx context.Context, index int) (ByteRange, error) {
				return s.getSegmentByteRange(ctx, index)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find segment for position %d: %w", startPos, err)
		}

		startIndex = result.Index
		byteRange = result.ByteRange
		fileLog.Trace("createStream found segment", "segment_index", startIndex, "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))
	}

	stream := NewFilePartStream(s.ctx, s.pool, s.file.Segments[startIndex:], s.file.Groups, s.bufferSegmentCount, s.cache)

	skipBytes := startPos - byteRange.Start
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

	fileLog.Trace("getSegmentByteRange", "n", segment.Number, "message_id", segment.MessageId)

	data, err := s.pool.fetchSegment(ctx, segment, s.file.Groups, s.cache)
	if err != nil {
		return ByteRange{}, err
	}

	byteRange := data.Header.ByteRange()
	fileLog.Trace("getSegmentByteRange result", "index", index, "byte_range", fmt.Sprintf("[%d, %d)", byteRange.Start, byteRange.End))

	return byteRange, nil
}
