package imdb

import (
	"slices"
	"sync"
	"time"
)

// maxHoldoffs caps how many keys a holdoff keeps.
const maxHoldoffs = 10000

// holdoff suppresses repeated attempts at a failing operation until a key's
// time passes. It holds at most max keys: when full, expired keys are pruned,
// and if it is still full the keys that expire soonest are dropped until it
// is half full, so pruning costs O(1) per set on average.
type holdoff struct {
	mu    sync.Mutex
	until map[string]time.Time
	max   int
}

func newHoldoff() *holdoff {
	return &holdoff{until: map[string]time.Time{}, max: maxHoldoffs}
}

func (h *holdoff) blocked(key string, now time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	until, ok := h.until[key]
	return ok && now.Before(until)
}

func (h *holdoff) set(key string, d time.Duration, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.until[key]; !ok && len(h.until) >= h.max {
		h.shrink(now)
	}
	h.until[key] = now.Add(d)
}

// shrink drops expired keys, then the soonest-expiring keys until the map is
// at most half full.
func (h *holdoff) shrink(now time.Time) {
	for k, until := range h.until {
		if !now.Before(until) {
			delete(h.until, k)
		}
	}
	if len(h.until) < h.max {
		return
	}
	keys := make([]string, 0, len(h.until))
	for k := range h.until {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int { return h.until[a].Compare(h.until[b]) })
	for _, k := range keys[:len(keys)-h.max/2] {
		delete(h.until, k)
	}
}

func (h *holdoff) clear(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.until, key)
}
