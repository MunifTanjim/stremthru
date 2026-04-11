package usenet_stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPercentile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{"empty", nil, 50, 0},
		{"single", []float64{5}, 50, 5},
		{"single_p0", []float64{5}, 0, 5},
		{"single_p100", []float64{5}, 100, 5},
		{"two_p0", []float64{10, 20}, 0, 10},
		{"two_p50", []float64{10, 20}, 50, 15},
		{"two_p100", []float64{10, 20}, 100, 20},
		{"five_p50", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"five_p25", []float64{1, 2, 3, 4, 5}, 25, 2},
		{"five_p75", []float64{1, 2, 3, 4, 5}, 75, 4},
		{"five_p95", []float64{1, 2, 3, 4, 5}, 95, 4.8},
		{"five_p99", []float64{1, 2, 3, 4, 5}, 99, 4.96},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := percentile(tc.sorted, tc.p)
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

func TestComputeThroughput(t *testing.T) {
	for _, tc := range []struct {
		name             string
		wallClockBytes   int64
		totalWallClockMs float64
		avgLatencyMs     float64
		segmentsFetched  int64
		bytesDownloaded  int64
		want             float64
	}{
		{"all_zero", 0, 0, 0, 0, 0, 0},
		{"wall_clock", 10000, 2000, 0, 0, 0, 5000},                       // 10000 bytes / 2s = 5000 B/s
		{"wall_clock_1s", 5000, 1000, 0, 0, 0, 5000},                     // 5000 bytes / 1s = 5000 B/s
		{"fallback_latency", 0, 0, 100, 10, 50000, 50000},                // 100ms * 10 segs = 1s, 50000/1 = 50000
		{"wall_clock_takes_priority", 10000, 2000, 100, 10, 50000, 5000}, // uses wall clock
		{"no_wall_clock_no_latency", 0, 0, 0, 10, 50000, 0},              // no way to compute
		{"no_wall_clock_no_segments", 0, 0, 100, 0, 50000, 0},            // can't compute without segments
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := computeThroughput(tc.wallClockBytes, tc.totalWallClockMs, tc.avgLatencyMs, tc.segmentsFetched, tc.bytesDownloaded)
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

func TestAccumulatorDrain(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		seg, bytes, missing, connErr, durations, wallClock := acc.drain()
		assert.Equal(t, int64(0), seg)
		assert.Equal(t, int64(0), bytes)
		assert.Equal(t, int64(0), missing)
		assert.Equal(t, int64(0), connErr)
		assert.Nil(t, durations)
		assert.Equal(t, 0.0, wallClock)
	})

	t.Run("accumulates_and_resets", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.SegmentsFetched = 5
		acc.BytesDownloaded = 1000
		acc.MissingSegments = map[string]struct{}{"a": {}, "b": {}}
		acc.ConnectionErrors = 1
		acc.Durations = []float64{10, 20, 30}
		acc.durationCount = 3

		seg, bytes, missing, connErr, durations, _ := acc.drain()
		assert.Equal(t, int64(5), seg)
		assert.Equal(t, int64(1000), bytes)
		assert.Equal(t, int64(2), missing)
		assert.Equal(t, int64(1), connErr)
		assert.Equal(t, []float64{10, 20, 30}, durations)

		// verify reset
		seg2, bytes2, missing2, connErr2, durations2, _ := acc.drain()
		assert.Equal(t, int64(0), seg2)
		assert.Equal(t, int64(0), bytes2)
		assert.Equal(t, int64(0), missing2)
		assert.Equal(t, int64(0), connErr2)
		assert.Nil(t, durations2)
	})

	t.Run("wall_clock_non_overlapping", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.fetchIntervals = []fetchInterval{
			{startMs: 0, endMs: 100},
			{startMs: 200, endMs: 300},
		}
		_, _, _, _, _, wallClock := acc.drain()
		assert.InDelta(t, 200.0, wallClock, 0.001) // 100 + 100
	})

	t.Run("wall_clock_overlapping", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.fetchIntervals = []fetchInterval{
			{startMs: 0, endMs: 150},
			{startMs: 100, endMs: 200},
		}
		_, _, _, _, _, wallClock := acc.drain()
		assert.InDelta(t, 200.0, wallClock, 0.001) // merged: 0-200
	})

	t.Run("wall_clock_fully_contained", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.fetchIntervals = []fetchInterval{
			{startMs: 0, endMs: 300},
			{startMs: 50, endMs: 100},
			{startMs: 150, endMs: 200},
		}
		_, _, _, _, _, wallClock := acc.drain()
		assert.InDelta(t, 300.0, wallClock, 0.001) // all contained in 0-300
	})

	t.Run("wall_clock_unsorted", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.fetchIntervals = []fetchInterval{
			{startMs: 200, endMs: 300},
			{startMs: 0, endMs: 100},
		}
		_, _, _, _, _, wallClock := acc.drain()
		assert.InDelta(t, 200.0, wallClock, 0.001) // sorted then merged: 100 + 100
	})

	t.Run("missing_segments_dedup", func(t *testing.T) {
		acc := &nzbServerAccumulator{}
		acc.recordArticleNotFound("msg-1")
		acc.recordArticleNotFound("msg-1") // duplicate
		acc.recordArticleNotFound("msg-2")
		_, _, missing, _, _, _ := acc.drain()
		assert.Equal(t, int64(2), missing) // deduplicated
	})
}
