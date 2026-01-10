package usenet_pool

import (
	"context"
	"io"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var (
	_ io.Reader = (*FilePartStream)(nil)
	_ io.Closer = (*FilePartStream)(nil)
)

var partLog = logger.Scoped("usenet/pool/file_part_stream")

type SegmentData interface {
	Body() []byte
	ByteRange() ByteRange
	FileSize() int64
	Size() int64
}

type FilePartStream struct {
	segments []nzb.Segment
	groups   []string
	pool     *Pool

	ctx    context.Context
	cancel context.CancelFunc
	dataCh chan SegmentData
	errCh  chan error

	mu       sync.Mutex
	currData []byte // Current segment's remaining data
	currPos  int    // Position within currentData
	closed   bool
}

func NewFilePartStream(
	pool *Pool,
	segments []nzb.Segment,
	groups []string,
	bufferSize int,
) *FilePartStream {
	ctx, cancel := context.WithCancel(context.Background())

	s := &FilePartStream{
		segments: segments,
		groups:   groups,
		pool:     pool,
		ctx:      ctx,
		cancel:   cancel,
		dataCh:   make(chan SegmentData, bufferSize),
		errCh:    make(chan error, 1),
	}

	partLog.Trace("part stream - created", "segment_count", len(segments), "buffer_size", bufferSize)

	go s.scheduleSegmentFetcher()

	return s
}

func (s *FilePartStream) scheduleSegmentFetcher() {
	defer close(s.dataCh)

	partLog.Trace("part stream - fetcher started", "segment_count", len(s.segments))

	for i := range s.segments {
		segment := &s.segments[i]
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		data, err := s.pool.fetchSegment(s.ctx, segment, s.groups)
		if err != nil {
			select {
			case s.errCh <- err:
			default:
			}
			return
		}

		select {
		case s.dataCh <- data:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *FilePartStream) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.EOF
	}

	for n < len(p) {
		select {
		case err := <-s.errCh:
			return n, err
		default:
		}

		if s.currPos < len(s.currData) {
			copied := copy(p[n:], s.currData[s.currPos:])
			s.currPos += copied
			n += copied
			continue
		}

		partLog.Trace("part stream - waiting for segment")

		data, ok := <-s.dataCh
		if !ok {
			partLog.Trace("part stream - no more segments", "segment_count", len(s.segments))
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		}

		partLog.Trace("part stream - segment received", "size", len(data.Body()))

		s.currData = data.Body()
		s.currPos = 0
	}

	return n, nil
}

func (s *FilePartStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.cancel()

	for range s.dataCh {
		// drain
	}

	return nil
}
