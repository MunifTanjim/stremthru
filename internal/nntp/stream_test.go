package nntp

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateOffsets(t *testing.T) {
	reader := &UsenetFileReader{
		segments: []nzb.Segment{
			{Bytes: 100, Number: 1, MessageId: "msg1"},
			{Bytes: 200, Number: 2, MessageId: "msg2"},
			{Bytes: 150, Number: 3, MessageId: "msg3"},
		},
		log: slog.Default(),
	}

	reader.calculateOffsets()

	expected := []int64{0, 100, 300, 450}
	assert.Equal(t, expected, reader.segmentOffsets)
	assert.Equal(t, int64(450), reader.totalSize)
}

func TestFindSegmentIndex(t *testing.T) {
	reader := &UsenetFileReader{
		segments: []nzb.Segment{
			{Bytes: 100, Number: 1, MessageId: "msg1"},
			{Bytes: 200, Number: 2, MessageId: "msg2"},
			{Bytes: 150, Number: 3, MessageId: "msg3"},
		},
		log: slog.Default(),
	}
	reader.calculateOffsets()

	tests := []struct {
		position     int64
		expectedIdx  int
		expectedDesc string
	}{
		{0, 0, "start of first segment"},
		{50, 0, "middle of first segment"},
		{99, 0, "end of first segment"},
		{100, 1, "start of second segment"},
		{200, 1, "middle of second segment"},
		{299, 1, "end of second segment"},
		{300, 2, "start of third segment"},
		{400, 2, "middle of third segment"},
		{449, 2, "end of third segment"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedDesc, func(t *testing.T) {
			idx := reader.findSegmentIndex(tt.position)
			assert.Equal(t, tt.expectedIdx, idx, "position=%d", tt.position)
		})
	}
}

func TestSegmentCache(t *testing.T) {
	t.Run("store and get", func(t *testing.T) {
		cache := newSegmentCache(3)
		data := []byte("test data")
		cache.store(0, data, nil)

		seg, ok := cache.get(0)
		require.True(t, ok)
		<-seg.ready
		assert.Equal(t, data, seg.data)
		assert.Nil(t, seg.err)
	})

	t.Run("LRU eviction", func(t *testing.T) {
		cache := newSegmentCache(3)
		cache.store(0, []byte("data0"), nil)
		cache.store(1, []byte("data1"), nil)
		cache.store(2, []byte("data2"), nil)

		// All three should be present
		_, ok0 := cache.get(0)
		_, ok1 := cache.get(1)
		_, ok2 := cache.get(2)
		assert.True(t, ok0)
		assert.True(t, ok1)
		assert.True(t, ok2)

		// Add a fourth item, should evict the oldest (0)
		cache.store(3, []byte("data3"), nil)

		_, ok0 = cache.get(0)
		_, ok3 := cache.get(3)
		assert.False(t, ok0, "segment 0 should be evicted")
		assert.True(t, ok3)
	})

	t.Run("touch updates LRU", func(t *testing.T) {
		cache := newSegmentCache(3)
		cache.store(0, []byte("data0"), nil)
		cache.store(1, []byte("data1"), nil)
		cache.store(2, []byte("data2"), nil)

		// Touch segment 0 (make it most recently used)
		cache.get(0)

		// Add a fourth item, should evict segment 1 (oldest now)
		cache.store(3, []byte("data3"), nil)

		_, ok0 := cache.get(0)
		_, ok1 := cache.get(1)
		_, ok3 := cache.get(3)
		assert.True(t, ok0, "segment 0 should still be present")
		assert.False(t, ok1, "segment 1 should be evicted")
		assert.True(t, ok3)
	})

	t.Run("hasOrClaiming", func(t *testing.T) {
		cache := newSegmentCache(3)

		// First call should claim and return false
		claimed := cache.hasOrClaiming(5)
		assert.False(t, claimed)

		// Second call should return true (already claimed)
		claimed = cache.hasOrClaiming(5)
		assert.True(t, claimed)

		// Store data for segment 5
		cache.store(5, []byte("data5"), nil)

		// Now it should return true (data present)
		claimed = cache.hasOrClaiming(5)
		assert.True(t, claimed)
	})
}

func TestSeek(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &UsenetFileReader{
		ctx:    ctx,
		cancel: cancel,
		segments: []nzb.Segment{
			{Bytes: 100, Number: 1, MessageId: "msg1"},
			{Bytes: 200, Number: 2, MessageId: "msg2"},
			{Bytes: 150, Number: 3, MessageId: "msg3"},
		},
		log: slog.Default(),
	}
	reader.calculateOffsets()

	tests := []struct {
		name           string
		offset         int64
		whence         int
		expectedPos    int64
		expectedError  bool
		initialPos     int64
	}{
		{"seek to start", 0, io.SeekStart, 0, false, 0},
		{"seek from start", 100, io.SeekStart, 100, false, 0},
		{"seek from current forward", 50, io.SeekCurrent, 150, false, 100},
		{"seek from current backward", -25, io.SeekCurrent, 75, false, 100},
		{"seek from end", -50, io.SeekEnd, 400, false, 0},
		{"seek from end to start", -450, io.SeekEnd, 0, false, 0},
		{"negative position error", -10, io.SeekStart, 0, true, 0},
		{"seek past end (allowed)", 500, io.SeekStart, 500, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader.position = tt.initialPos
			pos, err := reader.Seek(tt.offset, tt.whence)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPos, pos)
				assert.Equal(t, tt.expectedPos, reader.position)
			}
		})
	}
}

func TestSeekOnClosedReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &UsenetFileReader{
		ctx:      ctx,
		cancel:   cancel,
		segments: []nzb.Segment{{Bytes: 100, Number: 1, MessageId: "msg1"}},
		closed:   true,
		log:      slog.Default(),
	}
	reader.calculateOffsets()

	_, err := reader.Seek(0, io.SeekStart)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestReadOnClosedReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &UsenetFileReader{
		ctx:      ctx,
		cancel:   cancel,
		segments: []nzb.Segment{{Bytes: 100, Number: 1, MessageId: "msg1"}},
		closed:   true,
		log:      slog.Default(),
	}
	reader.calculateOffsets()

	buf := make([]byte, 10)
	_, err := reader.Read(buf)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestReadAtEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &UsenetFileReader{
		ctx:    ctx,
		cancel: cancel,
		segments: []nzb.Segment{
			{Bytes: 100, Number: 1, MessageId: "msg1"},
		},
		position: 100, // At end
		log:      slog.Default(),
	}
	reader.calculateOffsets()

	buf := make([]byte, 10)
	n, err := reader.Read(buf)
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)
}

func TestIsNoSuchArticleError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "no such article error",
			err:      &Error{Code: ErrorCodeNoSuchArticle},
			expected: true,
		},
		{
			name:     "connection error",
			err:      &Error{Code: ErrorCodeConnection},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNoSuchArticleError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStreamFileConfig_Validation(t *testing.T) {
	ctx := context.Background()
	pool := &UsenetPool{
		Log: slog.Default(),
	}

	t.Run("empty segments", func(t *testing.T) {
		config := StreamFileConfig{
			Segments: []nzb.Segment{},
		}

		_, err := pool.StreamFile(ctx, config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no segments")
	})

	t.Run("defaults applied", func(t *testing.T) {
		config := StreamFileConfig{
			Segments: []nzb.Segment{
				{Bytes: 100, Number: 1, MessageId: "msg1"},
			},
			// ParallelDownloads and BufferAhead not set
		}

		result, err := pool.StreamFile(ctx, config)
		require.NoError(t, err)
		defer result.Close()

		reader := result.ReadSeekCloser.(*UsenetFileReader)
		assert.Equal(t, defaultParallelDownloads, reader.parallelism)
		assert.Equal(t, defaultParallelDownloads+defaultBufferAhead, reader.cache.maxSize)
	})

	t.Run("custom values respected", func(t *testing.T) {
		config := StreamFileConfig{
			Segments: []nzb.Segment{
				{Bytes: 100, Number: 1, MessageId: "msg1"},
			},
			ParallelDownloads: 8,
			BufferAhead:       3,
		}

		result, err := pool.StreamFile(ctx, config)
		require.NoError(t, err)
		defer result.Close()

		reader := result.ReadSeekCloser.(*UsenetFileReader)
		assert.Equal(t, 8, reader.parallelism)
		assert.Equal(t, 11, reader.cache.maxSize) // 8 + 3
	})
}

func TestClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &UsenetFileReader{
		ctx:         ctx,
		cancel:      cancel,
		segments:    []nzb.Segment{{Bytes: 100, Number: 1, MessageId: "msg1"}},
		cache:       newSegmentCache(5),
		parallelism: 2,
		log:         slog.Default(),
	}
	reader.calculateOffsets()
	reader.startWorkers()

	// Store some data in cache
	reader.cache.store(0, []byte("test"), nil)

	err := reader.Close()
	assert.NoError(t, err)
	assert.True(t, reader.closed)

	// Second close should be idempotent
	err = reader.Close()
	assert.NoError(t, err)

	// Cache should be cleared
	assert.Equal(t, 0, len(reader.cache.segments))
}
