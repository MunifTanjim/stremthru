package imdb

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
)

const (
	// fastFresh is how long ratings and a running series' episode list stay
	// fresh.
	fastFresh = 3 * 24 * time.Hour
	// slowFresh is how long everything else stays fresh.
	slowFresh = 30 * 24 * time.Hour
	// recentWindow is how long after release a title counts as recently
	// released, so its rating is refreshed with the fast part.
	recentWindow = 90 * 24 * time.Hour
	// airedWindow is how long after airing an episode's details are still
	// refreshed with its series' fast part.
	airedWindow = 14 * 24 * time.Hour
)

// freshness says which parts of a title are stale and when the stored data
// is next due for refresh.
type freshness struct {
	SlowStale bool
	FastStale bool
	FastID    string    // id the fast refresh targets ("" if no fast part)
	ExpiresAt time.Time // the earlier due time of the parts that exist
}

func evaluate(st *db.RefreshState, id string, now time.Time) freshness {
	var f freshness
	slowDue := st.UpdatedAt.Add(slowFresh)
	f.SlowStale = st.Partial || !now.Before(slowDue)
	f.ExpiresAt = slowDue

	switch {
	case st.IsSeries:
		if st.Running {
			f.FastID = id
		}
	case st.Type == "tvEpisode" || st.SeriesID != "":
		if st.SeriesID != "" && st.SeriesRunning {
			f.FastID = st.SeriesID
		}
	case recentlyReleased(st.ReleaseDate, st.YearStart, now):
		f.FastID = id
	}
	if f.FastID != "" {
		fastUpdatedAt := st.FastUpdatedAt
		if f.FastID != id {
			// An episode's fast part is its series'.
			fastUpdatedAt = st.SeriesFastUpdatedAt
		}
		fastDue := fastUpdatedAt.Add(fastFresh)
		f.FastStale = !now.Before(fastDue)
		if fastDue.Before(f.ExpiresAt) {
			f.ExpiresAt = fastDue
		}
	}
	return f
}

// parseDay parses a "YYYY-MM-DD" release date.
func parseDay(date string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, date)
	return t, err == nil
}

// recentlyReleased reports whether a title was released within recentWindow
// or is not released yet. A title with only a year counts as recent when the
// year is this year or later.
func recentlyReleased(releaseDate string, yearStart int, now time.Time) bool {
	if d, ok := parseDay(releaseDate); ok {
		return now.Sub(d) <= recentWindow
	}
	year := yearStart
	if len(releaseDate) == 4 {
		if t, err := time.Parse("2006", releaseDate); err == nil {
			year = t.Year()
		}
	}
	return year != 0 && year >= now.Year()
}

// recentlyAired reports whether an episode aired within airedWindow or has
// not aired yet. Undated episodes are not recent; the caller decides about
// them.
func recentlyAired(releaseDate string, now time.Time) bool {
	if d, ok := parseDay(releaseDate); ok {
		return now.Sub(d) <= airedWindow
	}
	if t, err := time.Parse("2006", releaseDate); err == nil {
		return t.Year() >= now.Year()
	}
	return false
}
