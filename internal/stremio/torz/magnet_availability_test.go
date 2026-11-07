package stremio_torz

import (
	"errors"
	"reflect"
	"testing"
	"time"

	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
)

func TestDebridCandidateHashesUsesVisibleUniqueHashes(t *testing.T) {
	streams := []WrappedStream{
		{},
		{R: &stremio_transformer.StreamExtractorResult{Hash: "abcdef"}},
		{R: &stremio_transformer.StreamExtractorResult{Hash: "ABCDEF"}},
		{R: &stremio_transformer.StreamExtractorResult{Hash: ""}},
		{R: &stremio_transformer.StreamExtractorResult{Hash: "123456"}},
	}
	got := debridCandidateHashes(streams)
	want := []string{"abcdef", "123456"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checked hashes = %v; want %v", got, want)
	}
}

func TestAwaitDebridAvailabilityBlocksUntilProviderReturns(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	var result debridAvailability
	var err error
	go func() {
		result, err = awaitDebridAvailability(func() debridAvailability {
			close(started)
			<-release
			return debridAvailability{ByHash: map[string]string{"abc": "RD"}}
		})
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("debrid check never started")
	}
	select {
	case <-done:
		t.Fatal("returned availability before provider completed")
	default:
	}
	close(release)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("availability did not return after provider completed")
	}
	if err != nil || result.ByHash["abc"] != "RD" {
		t.Fatalf("availability = %#v, error = %v", result, err)
	}
}

func TestAwaitDebridAvailabilityDistinguishesProviderErrorFromUncached(t *testing.T) {
	providerErr := errors.New("debrid temporarily unavailable")
	_, err := awaitDebridAvailability(func() debridAvailability {
		return debridAvailability{HasErr: true, Err: []error{providerErr}}
	})
	if !errors.Is(err, providerErr) {
		t.Fatalf("provider failure should not be treated as uncached: %v", err)
	}
	_, err = awaitDebridAvailability(func() debridAvailability {
		return debridAvailability{HasErr: true}
	})
	if err == nil {
		t.Fatal("provider failure without a detailed error must still return an error")
	}

	uncached, err := awaitDebridAvailability(func() debridAvailability {
		return debridAvailability{ByHash: map[string]string{}}
	})
	if err != nil || len(uncached.ByHash) != 0 {
		t.Fatalf("completed uncached lookup = %#v, error = %v", uncached, err)
	}

	partial, err := awaitDebridAvailability(func() debridAvailability {
		return debridAvailability{
			ByHash:       map[string]string{"abc": "RD"},
			FailedStores: map[string]struct{}{"TB": {}},
			HasErr:       true,
			Err:          []error{providerErr},
		}
	})
	if err != nil || partial.ByHash["abc"] != "RD" {
		t.Fatalf("partial successful lookup = %#v, error = %v", partial, err)
	}
	if _, found := partial.FailedStores["TB"]; !found {
		t.Fatal("failed provider must remain excluded from uncached streams")
	}
}
