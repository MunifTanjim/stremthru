package usenet_stats

import (
	"cmp"
	"math/rand/v2"
	"slices"
	"sync"
	"time"
)

type fetchInterval struct {
	startMs float64
	endMs   float64
}

type ServerInfo struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	IsBackup bool   `json:"is_backup"`
}

var (
	globalMu      sync.Mutex
	globalServers = make(map[string]ServerInfo)
)

func RegisterServer(providerId string, info ServerInfo) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalServers[providerId] = info
}

func UnregisterServer(providerId string) {
	globalMu.Lock()
	defer globalMu.Unlock()
	delete(globalServers, providerId)
}

func ClearServers() {
	globalMu.Lock()
	defer globalMu.Unlock()
	clear(globalServers)
}

type nzbServerKey struct {
	NZBHash    string
	ProviderId string
}

const max_duration_count = 10_000

type nzbServerAccumulator struct {
	mu sync.Mutex

	SegmentsFetched  int64
	BytesDownloaded  int64
	MissingSegments  map[string]struct{}
	ConnectionErrors int64
	Durations        []float64
	durationCount    int64 // total recorded, may exceed len(Durations)
	fetchIntervals   []fetchInterval
}

func (a *nzbServerAccumulator) recordSegmentFetch(bytes int64, durationMs float64) {
	now := time.Now()
	endMs := float64(now.UnixMicro()) / 1000
	startMs := endMs - durationMs

	a.mu.Lock()
	defer a.mu.Unlock()
	a.SegmentsFetched++
	a.BytesDownloaded += bytes
	a.durationCount++
	if len(a.Durations) < max_duration_count {
		a.Durations = append(a.Durations, durationMs)
	} else {
		// reservoir sampling: replace a random element with probability maxDurationCount/durationCount
		if j := rand.Int64N(a.durationCount); j < max_duration_count {
			a.Durations[j] = durationMs
		}
	}
	a.fetchIntervals = append(a.fetchIntervals, fetchInterval{startMs, endMs})
}

func (a *nzbServerAccumulator) recordArticleNotFound(messageId string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.MissingSegments == nil {
		a.MissingSegments = make(map[string]struct{})
	}
	a.MissingSegments[messageId] = struct{}{}
}

func (a *nzbServerAccumulator) recordConnectionError() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ConnectionErrors++
}

func (a *nzbServerAccumulator) drain() (segFetched, bytesDown, missingSegments, connErrors int64, durations []float64, wallClockMs float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	segFetched = a.SegmentsFetched
	bytesDown = a.BytesDownloaded
	missingSegments = int64(len(a.MissingSegments))
	connErrors = a.ConnectionErrors
	durations = a.Durations

	// merge overlapping fetch intervals to get actual active download time
	if len(a.fetchIntervals) > 0 {
		slices.SortFunc(a.fetchIntervals, func(a, b fetchInterval) int {
			return cmp.Compare(a.startMs, b.startMs)
		})
		merged := a.fetchIntervals[0]
		for _, iv := range a.fetchIntervals[1:] {
			if iv.startMs <= merged.endMs {
				merged.endMs = max(merged.endMs, iv.endMs)
			} else {
				wallClockMs += merged.endMs - merged.startMs
				merged = iv
			}
		}
		wallClockMs += merged.endMs - merged.startMs
	}

	a.SegmentsFetched = 0
	a.BytesDownloaded = 0
	a.MissingSegments = nil
	a.ConnectionErrors = 0
	a.Durations = nil
	a.durationCount = 0
	a.fetchIntervals = nil
	return
}

var (
	nzbMu           sync.Mutex
	nzbAccumulators = make(map[nzbServerKey]*nzbServerAccumulator)
)

type EventName string

const (
	EventNameSegmentFetched  EventName = "segment_fetched"
	EventNameArticleNotFound EventName = "article_not_found"
	EventNameConnectionError EventName = "connection_error"
)

func Record(event EventName, nzbHash string, providerId string, messageId string, duration time.Duration, bytes int64) {
	key := nzbServerKey{NZBHash: nzbHash, ProviderId: providerId}
	nzbMu.Lock()
	acc, ok := nzbAccumulators[key]
	if !ok {
		acc = &nzbServerAccumulator{}
		nzbAccumulators[key] = acc
	}
	nzbMu.Unlock()

	switch event {
	case EventNameSegmentFetched:
		durationMs := float64(duration.Microseconds()) / 1000.0
		acc.recordSegmentFetch(bytes, durationMs)
	case EventNameArticleNotFound:
		acc.recordArticleNotFound(messageId)
	case EventNameConnectionError:
		acc.recordConnectionError()
	}
}

func drainAllAccumulators() map[nzbServerKey]*nzbServerAccumulator {
	nzbMu.Lock()
	defer nzbMu.Unlock()
	snapshot := make(map[nzbServerKey]*nzbServerAccumulator, len(nzbAccumulators))
	for k, v := range nzbAccumulators {
		snapshot[k] = v
	}
	return snapshot
}
