package usenet_pool

import (
	"context"
	"io"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var (
	_ io.ReadCloser = (*SegmentsStream)(nil)
)

var segmentLog = logger.Scoped("usenet/pool/segments_stream")

type SegmentsStream struct {
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

func NewSegmentsStream(
	pool *Pool,
	segments []nzb.Segment,
	groups []string,
	bufferCount int,
) *SegmentsStream {
	ctx, cancel := context.WithCancel(context.Background())

	s := &SegmentsStream{
		segments: segments,
		groups:   groups,
		pool:     pool,
		ctx:      ctx,
		cancel:   cancel,
		dataCh:   make(chan SegmentData, bufferCount),
		errCh:    make(chan error, 1),
	}

	segmentLog.Trace("segments stream - created", "segment_count", len(segments), "buffer_count", bufferCount)

	go s.scheduleSegmentFetcher()

	return s
}

func (s *SegmentsStream) scheduleSegmentFetcher() {
	defer close(s.dataCh)

	segmentLog.Trace("segments stream - fetcher started", "segment_count", len(s.segments))

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

func (s *SegmentsStream) Read(p []byte) (n int, err error) {
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

		segmentLog.Trace("segments stream - waiting for segment")

		data, ok := <-s.dataCh
		if !ok {
			segmentLog.Trace("segments stream - no more segments", "segment_count", len(s.segments))
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		}

		segmentLog.Trace("segments stream - segment received", "size", len(data.Body()))

		s.currData = data.Body()
		s.currPos = 0
	}

	return n, nil
}

func (s *SegmentsStream) Close() error {
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
