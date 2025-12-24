package nntp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/mnightingale/rapidyenc"
)

const (
	defaultParallelDownloads = 4
	defaultBufferAhead       = 2
	maxRetries               = 5
)

// StreamFileConfig configures file streaming from Usenet
type StreamFileConfig struct {
	Segments          []nzb.Segment // Ordered segments from NZB file
	Groups            []string      // Newsgroups where segments are posted
	ParallelDownloads int           // Number of concurrent connections (default: 4)
	BufferAhead       int           // Number of segments to prefetch ahead (default: 2)
}

// StreamFileResult wraps the ReadSeekCloser with metadata
type StreamFileResult struct {
	io.ReadSeekCloser
	Size int64 // Total file size (sum of segment sizes)
}

// UsenetFileReader implements io.ReadSeekCloser for streaming Usenet files
type UsenetFileReader struct {
	pool   *UsenetPool
	ctx    context.Context
	cancel context.CancelFunc

	// Segment metadata
	segments       []nzb.Segment
	segmentOffsets []int64 // Cumulative byte offsets [0, seg0.Bytes, seg0+seg1.Bytes, ...]
	totalSize      int64
	groups         []string

	// State
	mu       sync.Mutex
	position int64
	closed   bool

	// Caching & prefetch
	cache       *segmentCache
	prefetchCh  chan int
	prefetchWg  sync.WaitGroup
	parallelism int

	log *logger.Logger
}

// cachedSegment represents a cached segment with its data or error
type cachedSegment struct {
	data     []byte
	err      error
	ready    chan struct{} // Closed when data/error is ready
	fetching bool
}

// segmentCache manages caching of decoded segments with LRU eviction
type segmentCache struct {
	mu       sync.Mutex
	segments map[int]*cachedSegment
	maxSize  int
	lru      []int // LRU order for eviction
}

// newSegmentCache creates a new segment cache with the specified max size
func newSegmentCache(maxSegments int) *segmentCache {
	return &segmentCache{
		segments: make(map[int]*cachedSegment),
		maxSize:  maxSegments,
		lru:      make([]int, 0, maxSegments),
	}
}

// get retrieves a cached segment if it exists
func (sc *segmentCache) get(idx int) (*cachedSegment, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	seg, ok := sc.segments[idx]
	if ok {
		sc.touchLRU(idx)
	}
	return seg, ok
}

// hasOrClaiming checks if a segment exists or claims it for fetching
func (sc *segmentCache) hasOrClaiming(idx int) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if seg, ok := sc.segments[idx]; ok {
		return seg.fetching || seg.data != nil || seg.err != nil
	}

	// Claim this segment for fetching
	sc.segments[idx] = &cachedSegment{
		ready:    make(chan struct{}),
		fetching: true,
	}
	return false
}

// store stores segment data or error in the cache
func (sc *segmentCache) store(idx int, data []byte, err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	seg := sc.segments[idx]
	needsClose := false
	if seg == nil {
		seg = &cachedSegment{ready: make(chan struct{})}
		sc.segments[idx] = seg
		needsClose = true
	} else if seg.fetching {
		// Segment was claimed in hasOrClaiming, ready channel exists
		needsClose = true
	}

	seg.data = data
	seg.err = err
	seg.fetching = false

	if needsClose {
		close(seg.ready)
	}

	sc.lru = append(sc.lru, idx)
	sc.evictIfNeeded()
}

// touchLRU moves a segment to the end of the LRU list
func (sc *segmentCache) touchLRU(idx int) {
	// Remove from current position
	for i, v := range sc.lru {
		if v == idx {
			sc.lru = append(sc.lru[:i], sc.lru[i+1:]...)
			break
		}
	}
	// Add to end
	sc.lru = append(sc.lru, idx)
}

// evictIfNeeded removes oldest segments if cache exceeds max size
func (sc *segmentCache) evictIfNeeded() {
	for len(sc.lru) > sc.maxSize {
		oldest := sc.lru[0]
		sc.lru = sc.lru[1:]
		delete(sc.segments, oldest)
	}
}

// clear removes all cached segments
func (sc *segmentCache) clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.segments = make(map[int]*cachedSegment)
	sc.lru = sc.lru[:0]
}

// calculateOffsets computes cumulative byte offsets for all segments
func (ufr *UsenetFileReader) calculateOffsets() {
	ufr.segmentOffsets = make([]int64, len(ufr.segments)+1)
	var offset int64 = 0
	for i, seg := range ufr.segments {
		ufr.segmentOffsets[i] = offset
		offset += seg.Bytes
	}
	ufr.segmentOffsets[len(ufr.segments)] = offset // End offset
	ufr.totalSize = offset
}

// findSegmentIndex finds which segment contains the given byte position
func (ufr *UsenetFileReader) findSegmentIndex(pos int64) int {
	// Binary search for segment containing position
	return sort.Search(len(ufr.segments), func(i int) bool {
		return ufr.segmentOffsets[i+1] > pos
	})
}

// contextReader wraps an io.Reader with context cancellation
type contextReader struct {
	ctx context.Context
	r   io.Reader
	log *logger.Logger
	seg int
}

func (cr *contextReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled before reading
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
	}

	// Use a channel to make the read cancellable
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)

	go func() {
		n, err := cr.r.Read(p)
		ch <- result{n, err}
	}()

	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	case res := <-ch:
		return res.n, res.err
	}
}

// decodeYEnc decodes yEnc-encoded data from a reader
// expectedSize is used to pre-allocate the buffer for better performance
func decodeYEnc(ctx context.Context, r io.Reader, expectedSize int64, log *logger.Logger, segmentNum int) ([]byte, error) {
	log.Trace("decoding yEnc segment", "segment_number", segmentNum, "expected_bytes", expectedSize)

	// Add 30 seconds timeout for the entire decode operation
	decodeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Wrap reader with context cancellation support
	cr := &contextReader{
		ctx: decodeCtx,
		r:   r,
		log: log,
		seg: segmentNum,
	}

	dec := rapidyenc.NewDecoder(cr)

	// Pre-allocate buffer based on expected size to reduce allocations
	// The decoder is already streaming - it decodes as data arrives from network
	var buf bytes.Buffer
	if expectedSize > 0 {
		buf.Grow(int(expectedSize))
	}

	written, err := io.Copy(&buf, dec)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("yenc decode timeout after 30s: %w", err)
		}
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("yenc decode cancelled: %w", err)
		}
		log.Error("yEnc decode failed", "segment_number", segmentNum, "bytes_decoded", written, "error", err)
		return nil, fmt.Errorf("yenc decode failed: %w", err)
	}

	log.Trace("decoded yEnc segment", "segment_number", segmentNum, "decoded_bytes", written)
	return buf.Bytes(), nil
}

// isNoSuchArticleError checks if an error indicates a missing article
func isNoSuchArticleError(err error) bool {
	var nntpErr *Error
	if errors.As(err, &nntpErr) {
		return nntpErr.Code == ErrorCodeNoSuchArticle
	}
	return false
}

// fetchSegment fetches and decodes a single segment with provider failover
func (ufr *UsenetFileReader) fetchSegment(ctx context.Context, seg nzb.Segment) ([]byte, error) {
	ufr.log.Trace("fetching segment", "segment_number", seg.Number, "bytes", seg.Bytes)

	var excludeProviders []string
	includeBackup := false

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Get connection, potentially excluding failed providers
		conn, err := ufr.pool.GetConnection(ctx, excludeProviders, includeBackup)
		if err != nil {
			if !includeBackup {
				includeBackup = true
				continue
			}
			return nil, fmt.Errorf("no available providers: %w", err)
		}

		providerId := conn.ProviderId()

		// Select group if needed
		if len(ufr.groups) > 0 && conn.CurrentGroup() != ufr.groups[0] {
			if _, err := conn.Group(ufr.groups[0]); err != nil {
				conn.Release()
				ufr.log.Warn("failed to select newsgroup", "group", ufr.groups[0], "provider", providerId, "error", err)
				excludeProviders = append(excludeProviders, providerId)
				continue
			}
		}

		// Prepare message-id with angle brackets if not already present
		messageId := strings.TrimSpace(seg.MessageId)
		if !strings.HasPrefix(messageId, "<") {
			messageId = "<" + messageId + ">"
		}

		// Fetch article body
		article, err := conn.Body(messageId)
		if err != nil {
			conn.Release()

			if isNoSuchArticleError(err) {
				// Article missing from this provider - try another
				ufr.log.Trace("article not found, trying another provider", "segment_number", seg.Number, "provider", providerId)
				excludeProviders = append(excludeProviders, providerId)
				if !includeBackup {
					includeBackup = true
				}
				continue
			}
			ufr.log.Error("failed to fetch article", "message_id", messageId, "provider", providerId, "error", err)
			return nil, fmt.Errorf("failed to fetch article: %w", err)
		}
		defer article.Body.Close()

		// Decode yEnc
		data, err := decodeYEnc(ctx, article.Body, seg.Bytes, ufr.log, seg.Number)
		conn.Release()

		if err != nil {
			ufr.log.Error("yEnc decode failed", "segment_number", seg.Number, "error", err)
			return nil, err
		}

		ufr.log.Trace("fetched segment", "segment_number", seg.Number, "provider", providerId, "decoded_bytes", len(data))
		return data, nil
	}

	ufr.log.Error("segment failed on all providers", "segment_number", seg.Number, "message_id", seg.MessageId, "max_retries", maxRetries)
	return nil, fmt.Errorf("segment %s failed on all providers after %d attempts", seg.MessageId, maxRetries)
}

// getSegmentData retrieves segment data from cache or fetches it
func (ufr *UsenetFileReader) getSegmentData(idx int) ([]byte, error) {
	// Check cache first
	if seg, ok := ufr.cache.get(idx); ok {
		ufr.log.Trace("segment cache hit", "segment_index", idx)
		// Wait for segment to be ready
		<-seg.ready
		if seg.err != nil {
			return nil, seg.err
		}
		return seg.data, nil
	}

	// Fetch synchronously if not in cache
	ufr.log.Trace("segment cache miss, fetching", "segment_index", idx)
	data, err := ufr.fetchSegment(ufr.ctx, ufr.segments[idx])
	if err != nil {
		ufr.cache.store(idx, nil, err)
		return nil, err
	}
	ufr.cache.store(idx, data, nil)
	return data, nil
}

// startWorkers starts prefetch worker goroutines
func (ufr *UsenetFileReader) startWorkers() {
	ufr.prefetchCh = make(chan int, ufr.parallelism*2)

	for i := 0; i < ufr.parallelism; i++ {
		ufr.prefetchWg.Add(1)
		go ufr.prefetchWorker(i)
	}
}

// prefetchWorker fetches segments in the background
func (ufr *UsenetFileReader) prefetchWorker(workerID int) {
	defer ufr.prefetchWg.Done()

	for {
		select {
		case <-ufr.ctx.Done():
			return
		case segIdx, ok := <-ufr.prefetchCh:
			if !ok {
				return
			}
			if segIdx < 0 || segIdx >= len(ufr.segments) {
				continue
			}

			// Check if already cached or being fetched
			if ufr.cache.hasOrClaiming(segIdx) {
				continue
			}

			// Fetch and cache
			data, err := ufr.fetchSegment(ufr.ctx, ufr.segments[segIdx])
			ufr.cache.store(segIdx, data, err)
		}
	}
}

// triggerPrefetch triggers prefetch for segments starting at the given index
func (ufr *UsenetFileReader) triggerPrefetch(startIdx int) {
	// Non-blocking sends for prefetch requests
	for i := 0; i < ufr.parallelism && startIdx+i < len(ufr.segments); i++ {
		select {
		case ufr.prefetchCh <- startIdx + i:
		default:
			// Channel full, skip
		}
	}
}

// Read reads data from the file stream
func (ufr *UsenetFileReader) Read(p []byte) (n int, err error) {
	ufr.mu.Lock()
	defer ufr.mu.Unlock()

	if ufr.closed {
		return 0, io.ErrClosedPipe
	}

	if ufr.position >= ufr.totalSize {
		return 0, io.EOF
	}

	ufr.log.Trace("read", "position", ufr.position, "requested_bytes", len(p))

	totalRead := 0
	for totalRead < len(p) && ufr.position < ufr.totalSize {
		// Find which segment contains current position
		segIdx := ufr.findSegmentIndex(ufr.position)
		segStart := ufr.segmentOffsets[segIdx]
		segEnd := ufr.segmentOffsets[segIdx+1]

		// Get segment data (from cache or fetch)
		segData, err := ufr.getSegmentData(segIdx)
		if err != nil {
			ufr.log.Error("failed to get segment data", "segment_index", segIdx, "error", err)
			if totalRead > 0 {
				return totalRead, nil // Return what we have
			}
			return 0, err
		}

		// Calculate offset within segment
		offsetInSeg := ufr.position - segStart

		// Calculate how many bytes to copy
		bytesAvailable := int(segEnd - ufr.position)
		bytesToCopy := min(len(p)-totalRead, bytesAvailable)

		// Copy data
		copy(p[totalRead:], segData[offsetInSeg:offsetInSeg+int64(bytesToCopy)])

		totalRead += bytesToCopy
		ufr.position += int64(bytesToCopy)

		// Trigger prefetch for next segments
		ufr.triggerPrefetch(segIdx + 1)
	}

	ufr.log.Trace("read complete", "bytes_read", totalRead, "new_position", ufr.position)
	return totalRead, nil
}

// Seek seeks to a position in the file stream
func (ufr *UsenetFileReader) Seek(offset int64, whence int) (int64, error) {
	ufr.mu.Lock()
	defer ufr.mu.Unlock()

	if ufr.closed {
		return 0, io.ErrClosedPipe
	}

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = ufr.position + offset
	case io.SeekEnd:
		newPos = ufr.totalSize + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newPos < 0 {
		return 0, errors.New("negative position")
	}

	ufr.log.Trace("seek", "old_position", ufr.position, "new_position", newPos)
	ufr.position = newPos

	// Prefetch segments around new position
	if newPos < ufr.totalSize {
		segIdx := ufr.findSegmentIndex(newPos)
		ufr.triggerPrefetch(segIdx)
	}

	return newPos, nil
}

// Close closes the file stream and stops all workers
func (ufr *UsenetFileReader) Close() error {
	ufr.mu.Lock()
	if ufr.closed {
		ufr.mu.Unlock()
		return nil
	}
	ufr.closed = true
	ufr.mu.Unlock()

	// Cancel context to stop workers
	ufr.cancel()

	// Close prefetch channel and wait for workers
	close(ufr.prefetchCh)
	ufr.prefetchWg.Wait()

	// Clear cache
	ufr.cache.clear()

	return nil
}
