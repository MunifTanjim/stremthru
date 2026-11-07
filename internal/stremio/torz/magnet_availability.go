package stremio_torz

import (
	"errors"
	"strings"
)

// debridAvailability contains only completed provider responses. An empty
// ByHash with HasErr=false means the provider checked the hashes and none
// are cached; an empty result after a provider error is not a cache miss.
type debridAvailability struct {
	ByHash       map[string]string
	FailedStores map[string]struct{}
	Err          []error
	HasErr       bool
}

// awaitDebridAvailability is deliberately synchronous. Unlike the former
// background queue, it must finish before the stream handler labels any
// torrent as cached or uncached.
func awaitDebridAvailability(check func() debridAvailability) (debridAvailability, error) {
	result := check()
	if result.HasErr && len(result.ByHash) == 0 {
		err := errors.Join(result.Err...)
		if err == nil {
			err = errors.New("debrid availability check failed")
		}
		return result, err
	}
	return result, nil
}

// Only check hashes belonging to streams that will actually be rendered.
// Deduplicating also prevents repeated checks for multiple files per torrent.
func debridCandidateHashes(streams []WrappedStream) []string {
	hashes := make([]string, 0, len(streams))
	seen := make(map[string]struct{}, len(streams))
	for i := range streams {
		result := streams[i].R
		if result == nil || result.Hash == "" {
			continue
		}
		key := strings.ToLower(result.Hash)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		hashes = append(hashes, result.Hash)
	}
	return hashes
}
