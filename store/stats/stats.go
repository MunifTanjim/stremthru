package stats

import (
	"maps"
	"math"
	"math/rand"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/store"
)

const (
	bucket_count             = 60
	bucket_duration          = time.Minute
	max_durations_per_bucket = 100_000
)

type bucket struct {
	timestamp       int64
	count           int64
	errorCount      int64
	totalDurationMs float64
	minDurationMs   float64
	maxDurationMs   float64
	durations       []float64
}

type MethodStats struct {
	mu      sync.Mutex
	buckets [bucket_count]bucket
}

func (ms *MethodStats) getBucket(now time.Time) *bucket {
	ts := now.Truncate(bucket_duration).Unix()
	idx := int(ts/int64(bucket_duration.Seconds())) % bucket_count
	b := &ms.buckets[idx]
	if b.timestamp != ts {
		b.timestamp = ts
		b.count = 0
		b.errorCount = 0
		b.totalDurationMs = 0
		b.minDurationMs = 0
		b.maxDurationMs = 0
		b.durations = b.durations[:0]
	}
	return b
}

func (ms *MethodStats) Record(duration time.Duration, isError bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	b := ms.getBucket(now)
	durationMs := float64(duration.Microseconds()) / 1000.0
	b.count++
	if isError {
		b.errorCount++
	}
	b.totalDurationMs += durationMs
	if len(b.durations) < max_durations_per_bucket {
		b.durations = append(b.durations, durationMs)
	} else {
		if j := rand.Intn(int(b.count)); j < max_durations_per_bucket {
			b.durations[j] = durationMs
		}
	}
	if b.count == 1 {
		b.minDurationMs = durationMs
		b.maxDurationMs = durationMs
	} else {
		if durationMs < b.minDurationMs {
			b.minDurationMs = durationMs
		}
		if durationMs > b.maxDurationMs {
			b.maxDurationMs = durationMs
		}
	}
}

type MethodStatsSnapshot struct {
	TotalCount        int64   `json:"total_count"`
	ErrorCount        int64   `json:"error_count"`
	ErrorRate         float64 `json:"error_rate"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
	MinDurationMs     float64 `json:"min_duration_ms"`
	MaxDurationMs     float64 `json:"max_duration_ms"`
	P50DurationMs     float64 `json:"p50_duration_ms"`
	P95DurationMs     float64 `json:"p95_duration_ms"`
	P99DurationMs     float64 `json:"p99_duration_ms"`
	RequestsPerMinute float64 `json:"requests_per_minute"`
}

func (ms *MethodStats) Snapshot(window time.Duration) MethodStatsSnapshot {
	now := time.Now()
	cutoff := now.Add(-window).Truncate(bucket_duration).Unix()

	var snap MethodStatsSnapshot
	var totalDurationMs float64
	var allDurations []float64
	snap.MinDurationMs = math.MaxFloat64
	hasData := false
	var earliestTs int64

	ms.mu.Lock()
	for i := range ms.buckets {
		b := &ms.buckets[i]
		if b.timestamp == 0 || b.timestamp < cutoff {
			continue
		}
		hasData = true
		if earliestTs == 0 || b.timestamp < earliestTs {
			earliestTs = b.timestamp
		}
		snap.TotalCount += b.count
		snap.ErrorCount += b.errorCount
		totalDurationMs += b.totalDurationMs
		if b.minDurationMs < snap.MinDurationMs {
			snap.MinDurationMs = b.minDurationMs
		}
		if b.maxDurationMs > snap.MaxDurationMs {
			snap.MaxDurationMs = b.maxDurationMs
		}
		allDurations = append(allDurations, b.durations...)
	}
	ms.mu.Unlock()

	if !hasData {
		snap.MinDurationMs = 0
		return snap
	}

	if snap.TotalCount > 0 {
		snap.AvgDurationMs = math.Round(totalDurationMs/float64(snap.TotalCount)*100) / 100
		snap.ErrorRate = math.Round(float64(snap.ErrorCount)/float64(snap.TotalCount)*10000) / 100
		nowTruncated := now.Truncate(bucket_duration)
		earliestTime := time.Unix(earliestTs, 0)
		actualWindow := min(nowTruncated.Sub(earliestTime)+bucket_duration, window)
		actualMinutes := actualWindow.Minutes()
		if actualMinutes > 0 {
			snap.RequestsPerMinute = math.Round(float64(snap.TotalCount)/actualMinutes*100) / 100
		}
	}
	snap.MinDurationMs = math.Round(snap.MinDurationMs*100) / 100
	snap.MaxDurationMs = math.Round(snap.MaxDurationMs*100) / 100

	if len(allDurations) > 0 {
		slices.Sort(allDurations)
		snap.P50DurationMs = percentile(allDurations, 50)
		snap.P95DurationMs = percentile(allDurations, 95)
		snap.P99DurationMs = percentile(allDurations, 99)
	}

	return snap
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	rank := max(int(math.Ceil(p/100*float64(n)))-1, 0)
	if rank >= n {
		rank = n - 1
	}
	return math.Round(sorted[rank]*100) / 100
}

type StoreStats struct {
	mu      sync.Mutex
	methods map[string]*MethodStats
}

func (ss *StoreStats) Record(method string, duration time.Duration, isError bool) {
	ss.mu.Lock()
	ms, ok := ss.methods[method]
	if !ok {
		ms = &MethodStats{}
		ss.methods[method] = ms
	}
	ss.mu.Unlock()
	ms.Record(duration, isError)
}

func (ss *StoreStats) Snapshot(window time.Duration) map[string]MethodStatsSnapshot {
	ss.mu.Lock()
	methods := make(map[string]*MethodStats, len(ss.methods))
	maps.Copy(methods, ss.methods)
	ss.mu.Unlock()

	result := make(map[string]MethodStatsSnapshot, len(methods))
	for name, ms := range methods {
		snap := ms.Snapshot(window)
		if snap.TotalCount > 0 {
			result[name] = snap
		}
	}
	return result
}

var (
	globalMu    sync.Mutex
	globalStats = make(map[store.StoreName]*StoreStats)
)

func Record(storeName store.StoreName, method string, duration time.Duration, isError bool) {
	globalMu.Lock()
	ss, ok := globalStats[storeName]
	if !ok {
		ss = &StoreStats{methods: make(map[string]*MethodStats)}
		globalStats[storeName] = ss
	}
	globalMu.Unlock()
	ss.Record(method, duration, isError)
}

type StoreStatsSnapshot struct {
	Name    store.StoreName            `json:"name"`
	Methods []MethodStatsSnapshotEntry `json:"methods"`
}

type MethodStatsSnapshotEntry struct {
	Name string `json:"name"`
	MethodStatsSnapshot
}

func GetSnapshot(window time.Duration) []StoreStatsSnapshot {
	globalMu.Lock()
	stores := make(map[store.StoreName]*StoreStats, len(globalStats))
	maps.Copy(stores, globalStats)
	globalMu.Unlock()

	var result []StoreStatsSnapshot
	for name, ss := range stores {
		methods := ss.Snapshot(window)
		if len(methods) == 0 {
			continue
		}
		entry := StoreStatsSnapshot{
			Name:    name,
			Methods: make([]MethodStatsSnapshotEntry, 0, len(methods)),
		}
		for methodName, snap := range methods {
			entry.Methods = append(entry.Methods, MethodStatsSnapshotEntry{
				Name:                methodName,
				MethodStatsSnapshot: snap,
			})
		}
		sort.Slice(entry.Methods, func(i, j int) bool {
			return entry.Methods[i].Name < entry.Methods[j].Name
		})
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
