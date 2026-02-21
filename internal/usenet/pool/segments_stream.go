package usenet_pool

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
)

var (
	_ io.ReadCloser = (*SegmentsStream)(nil)
)

var segmentLog = logger.Scoped("usenet/pool/segments_stream")

type segmentResult struct {
	idx  int
	data *SegmentData
	err  error
}

type segmentWithIdx struct {
	*nzb.Segment
	idx int
}

type SegmentsStream struct {
	segments []nzb.Segment
	groups   []string
	pool     *Pool

	ctx      context.Context
	cancel   context.CancelFunc
	dataChan chan *SegmentData
	errChan  chan error

	bufferCond          *sync.Cond   // signals when buffer space available
	bufferSizeRemaining atomic.Int64 // remaining buffer space

	mu       sync.Mutex
	currData []byte // Current segment's remaining data
	currPos  int    // Position within currentData
	closed   bool

	workerCount int
}

func NewSegmentsStream(
	pool *Pool,
	segments []nzb.Segment,
	groups []string,
	bufferSize int64,
) *SegmentsStream {
	ctx, cancel := context.WithCancel(context.Background())

	workerCount := max(min(len(segments), config.Newz.MaxConnectionPerStream), 1)

	s := &SegmentsStream{
		segments:    segments,
		groups:      groups,
		pool:        pool,
		ctx:         ctx,
		cancel:      cancel,
		dataChan:    make(chan *SegmentData, workerCount*2),
		errChan:     make(chan error, 1),
		bufferCond:  sync.NewCond(&sync.Mutex{}),
		workerCount: workerCount,
	}
	s.bufferSizeRemaining.Store(bufferSize)

	segmentLog.Trace("segments stream - created", "segment_count", len(segments), "buffer_size", bufferSize, "worker_count", workerCount)

	go s.startSegmentFetcher()

	return s
}

func (s *SegmentsStream) startSegmentFetcher() {
	segmentLog.Trace("segments stream - fetcher started", "segment_count", len(s.segments), "worker_count", s.workerCount)

	if len(s.segments) == 0 {
		close(s.dataChan)
		return
	}

	segmentChan := make(chan segmentWithIdx, s.workerCount)
	resultChan := make(chan segmentResult, s.workerCount*2)

	go s.startSegmentFetchDispatcher(segmentChan)

	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Go(func() {
			s.startFetcher(segmentChan, resultChan)
		})
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	s.startSegmentResultCollector(resultChan)
}

func (s *SegmentsStream) startSegmentFetchDispatcher(segmentChan chan<- segmentWithIdx) {
	defer close(segmentChan)

	for idx := range s.segments {
		segment := &s.segments[idx]

		s.bufferCond.L.Lock()
		for s.bufferSizeRemaining.Load() <= 0 && s.ctx.Err() == nil {
			segmentLog.Trace("segments stream - waiting for buffer space", "segment_num", segment.Number)
			s.bufferCond.Wait()
		}
		if s.ctx.Err() != nil {
			s.bufferCond.L.Unlock()
			return
		}
		s.bufferSizeRemaining.Add(-segment.Bytes)
		s.bufferCond.L.Unlock()

		select {
		case <-s.ctx.Done():
			return
		case segmentChan <- segmentWithIdx{Segment: segment, idx: idx}:
		}
	}
}

func (s *SegmentsStream) startFetcher(segmentChan <-chan segmentWithIdx, resultChan chan<- segmentResult) {
	for segmentWithIdx := range segmentChan {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		data, err := s.pool.fetchSegment(s.ctx, segmentWithIdx.Segment, s.groups)
		if data != nil {
			if adjustment := segmentWithIdx.Bytes - data.Size; adjustment != 0 {
				s.bufferSizeRemaining.Add(adjustment)
				s.bufferCond.Signal()
			}
		}

		select {
		case resultChan <- segmentResult{idx: segmentWithIdx.idx, data: data, err: err}:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *SegmentsStream) startSegmentResultCollector(resultCh <-chan segmentResult) {
	defer close(s.dataChan)

	pending := make(map[int]*SegmentData)
	nextIdx := 0
	totalSegments := len(s.segments)
	receivedCount := 0

	for receivedCount < totalSegments {
		select {
		case <-s.ctx.Done():
			return
		case result, ok := <-resultCh:
			if !ok {
				return
			}

			receivedCount++

			if result.err != nil {
				segmentLog.Trace("segments stream - failed result", "error", result.err, "idx", result.idx)
				select {
				case s.errChan <- result.err:
				default:
				}
				return
			}

			segmentLog.Trace("segments stream - received result", "idx", result.idx, "next_expected_idx", nextIdx, "pending_count", len(pending))

			pending[result.idx] = result.data

			for {
				data, ok := pending[nextIdx]
				if !ok {
					break
				}
				delete(pending, nextIdx)

				select {
				case s.dataChan <- data:
					segmentLog.Trace("segments stream - sent segment", "idx", nextIdx, "size", len(data.Body))
					nextIdx++
				case <-s.ctx.Done():
					return
				}
			}
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
		case err := <-s.errChan:
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

		data, ok := <-s.dataChan
		if !ok {
			segmentLog.Trace("segments stream - no more segments", "segment_count", len(s.segments))
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		}

		s.bufferSizeRemaining.Add(data.Size)
		s.bufferCond.Signal()

		segmentLog.Trace("segments stream - segment received", "size", len(data.Body))

		s.currData = data.Body
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

	s.bufferCond.Broadcast()

	for range s.dataChan {
		// drain
	}

	return nil
}
