package imdb

import (
	"fmt"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
)

func TestEvaluate(t *testing.T) {
	now := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	ago := func(d time.Duration) time.Time { return now.Add(-d) }

	cases := []struct {
		name string
		st   db.RefreshState
		want freshness
	}{
		{
			name: "fresh running series",
			st:   db.RefreshState{Type: "tvSeries", IsSeries: true, Running: true, UpdatedAt: ago(day), FastUpdatedAt: ago(day)},
			want: freshness{FastID: "tt1", ExpiresAt: now.Add(2 * day)},
		},
		{
			name: "running series with stale fast part",
			st:   db.RefreshState{Type: "tvSeries", IsSeries: true, Running: true, UpdatedAt: ago(4 * day), FastUpdatedAt: ago(4 * day)},
			want: freshness{FastStale: true, FastID: "tt1", ExpiresAt: ago(day)},
		},
		{
			name: "ended series",
			st:   db.RefreshState{Type: "tvSeries", IsSeries: true, UpdatedAt: ago(10 * day), FastUpdatedAt: ago(10 * day)},
			want: freshness{ExpiresAt: now.Add(20 * day)},
		},
		{
			name: "recently released movie",
			st:   db.RefreshState{Type: "movie", ReleaseDate: "2026-09-01", UpdatedAt: ago(4 * day), FastUpdatedAt: ago(4 * day)},
			want: freshness{FastStale: true, FastID: "tt1", ExpiresAt: ago(day)},
		},
		{
			name: "old movie",
			st:   db.RefreshState{Type: "movie", ReleaseDate: "2001-05-05", YearStart: 2001, UpdatedAt: ago(31 * day), FastUpdatedAt: ago(31 * day)},
			want: freshness{SlowStale: true, ExpiresAt: ago(day)},
		},
		{
			name: "partial episode",
			st:   db.RefreshState{Type: "tvEpisode", Partial: true, UpdatedAt: ago(day), FastUpdatedAt: ago(day), SeriesID: "tt0903747", SeriesRunning: true, SeriesFastUpdatedAt: ago(day)},
			want: freshness{SlowStale: true, FastID: "tt0903747", ExpiresAt: now.Add(2 * day)},
		},
		{
			name: "episode of a running series",
			st:   db.RefreshState{Type: "tvEpisode", UpdatedAt: ago(day), FastUpdatedAt: ago(4 * day), SeriesID: "tt0903747", SeriesRunning: true, SeriesFastUpdatedAt: ago(4 * day)},
			want: freshness{FastStale: true, FastID: "tt0903747", ExpiresAt: ago(day)},
		},
		{
			name: "episode follows its series' fast refresh time",
			st:   db.RefreshState{Type: "tvEpisode", UpdatedAt: ago(day), FastUpdatedAt: ago(day), SeriesID: "tt0903747", SeriesRunning: true, SeriesFastUpdatedAt: ago(4 * day)},
			want: freshness{FastStale: true, FastID: "tt0903747", ExpiresAt: ago(day)},
		},
		{
			name: "orphan episode has only the slow part",
			st:   db.RefreshState{Type: "tvEpisode", UpdatedAt: ago(day), FastUpdatedAt: ago(10 * day)},
			want: freshness{ExpiresAt: now.Add(29 * day)},
		},
		{
			name: "slow part due in 500ms",
			st:   db.RefreshState{Type: "movie", ReleaseDate: "2001-05-05", YearStart: 2001, UpdatedAt: ago(30*day - 500*time.Millisecond)},
			want: freshness{ExpiresAt: now.Add(500 * time.Millisecond)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := c.st
			if got := evaluate(&st, "tt1", now); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestRecentlyReleased(t *testing.T) {
	now := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		date string
		year int
		want bool
	}{
		{"2026-07-20", 2026, true},
		{"2026-07-18", 2026, false},
		{"2027-01-01", 2027, true},
		{"2026", 2026, true},
		{"", 2025, false},
		{"", 2026, true},
	}
	for _, c := range cases {
		if got := recentlyReleased(c.date, c.year, now); got != c.want {
			t.Errorf("recentlyReleased(%q, %d) = %v, want %v", c.date, c.year, got, c.want)
		}
	}
}

func TestRecentlyAired(t *testing.T) {
	now := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	for date, want := range map[string]bool{"2026-10-10": true, "2026-10-02": false, "2026-11-01": true, "2026": true, "2008": false, "": false} {
		if got := recentlyAired(date, now); got != want {
			t.Errorf("recentlyAired(%q) = %v, want %v", date, got, want)
		}
	}
}

func TestHoldoff(t *testing.T) {
	h := newHoldoff()
	now := time.Now()
	h.set("a", time.Minute, now)
	if !h.blocked("a", now.Add(30*time.Second)) || h.blocked("a", now.Add(time.Minute)) {
		t.Fatal("blocked window wrong")
	}
	h.clear("a")
	if h.blocked("a", now) {
		t.Fatal("clear did not unblock")
	}
}

func TestHoldoffIsBounded(t *testing.T) {
	h := &holdoff{until: map[string]time.Time{}, max: 4}
	now := time.Now()
	h.set("expired", time.Second, now.Add(-time.Hour))
	for i := 0; i < 10; i++ {
		h.set(fmt.Sprintf("k%d", i), time.Duration(i+1)*time.Minute, now)
		if n := len(h.until); n > 4 {
			t.Fatalf("size %d over cap after %d sets", n, i+1)
		}
	}
	if _, ok := h.until["expired"]; ok {
		t.Fatal("expired key kept")
	}
	// The newest keys, which are held longest, stay.
	if !h.blocked("k9", now) {
		t.Fatal("newest key evicted")
	}
}
