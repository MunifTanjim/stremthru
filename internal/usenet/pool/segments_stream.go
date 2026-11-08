package usenet_pool

import (
	"context"
	"errors"
	"fmt"
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

// errSegmentOffset reports a segment whose yEnc range does not start where
// the bytes before it ended, or does not match its decoded length.
var errSegmentOffset = errors.New("usenet: segment does not continue the previous one")

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

	mu           sync.Mutex
	currData     []byte // Current segment's remaining data
	currPos      int    // Position within currentData
	readErr      error  // Error or EOF that ended the stream, returned once the data before it is read
	checkOffsets bool   // Fail with errSegmentOffset when a segment does not continue the bytes before it
	nextOffset   int64  // File offset the next segment should start at
	closed       bool

	workerCount int
}

func NewSegmentsStream(
	ctx context.Context,
	pool *Pool,
	segments []nzb.Segment,
	groups []string,
	bufferSize int64,
) *SegmentsStream {
	ctx, cancel := context.WithCancel(ctx)

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

	go func() {
		<-s.ctx.Done()
		// Under the lock, so the wakeup cannot fall between the dispatcher's
		// ctx check and its Wait.
		s.bufferCond.L.Lock()
		s.bufferCond.Broadcast()
		s.bufferCond.L.Unlock()
	}()

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
				s.bufferCond.L.Lock()
				s.bufferSizeRemaining.Add(adjustment)
				s.bufferCond.L.Unlock()
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
	failed := make(map[int]error)
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
				segmentLog.Trace("segments stream - failed result", "error", result.err, "idx", result.idx, "next_expected_idx", nextIdx)
				failed[result.idx] = result.err
			} else {
				segmentLog.Trace("segments stream - received result", "idx", result.idx, "next_expected_idx", nextIdx, "pending_count", len(pending))
				pending[result.idx] = result.data
			}

			// Deliver in order. A failed segment ends the stream only once
			// every segment before it has been delivered, so the reader gets
			// all bytes up to a missing article before the error, however
			// far ahead the read-ahead found it.
			for {
				if err, ok := failed[nextIdx]; ok {
					select {
					case s.errChan <- err:
					default:
					}
					return
				}

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

	for n < len(p) && s.readErr == nil {
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
			// The collector sends a fetch error before it closes dataChan,
			// and only after every segment before the failed one. The error
			// stays, so a later call cannot mistake it for EOF.
			select {
			case err := <-s.errChan:
				s.readErr = err
			default:
				s.readErr = io.EOF
			}
			break
		}

		s.bufferCond.L.Lock()
		s.bufferSizeRemaining.Add(data.Size)
		s.bufferCond.L.Unlock()
		s.bufferCond.Signal()

		if s.checkOffsets {
			if data.ByteRange.Start != s.nextOffset || data.ByteRange.Count() != int64(len(data.Body)) {
				s.readErr = fmt.Errorf("%w: it holds %d bytes for [%d, %d), expected a start at %d", errSegmentOffset, len(data.Body), data.ByteRange.Start, data.ByteRange.End, s.nextOffset)
				break
			}
			s.nextOffset = data.ByteRange.End
		}

		segmentLog.Trace("segments stream - segment received", "size", len(data.Body))

		s.currData = data.Body
		s.currPos = 0
	}

	if n > 0 {
		return n, nil
	}
	return 0, s.readErr
}

// checkSegmentOffsets makes Read fail with errSegmentOffset when a segment's
// yEnc range does not start where the bytes before it ended, or does not
// match its decoded length. start is the file offset of the stream's first
// segment. Call it before the first Read.
func (s *SegmentsStream) checkSegmentOffsets(start int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkOffsets = true
	s.nextOffset = start
}

// growBuffer raises the stream's read-ahead budget by n bytes.
func (s *SegmentsStream) growBuffer(n int64) {
	if n <= 0 {
		return
	}
	s.bufferCond.L.Lock()
	s.bufferSizeRemaining.Add(n)
	s.bufferCond.L.Unlock()
	s.bufferCond.Signal()
}

func (s *SegmentsStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.cancel()

	s.bufferCond.L.Lock()
	s.bufferCond.Broadcast()
	s.bufferCond.L.Unlock()

	for range s.dataChan {
		// drain
	}

	return nil
}
