package imdb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
	"golang.org/x/sync/errgroup"
)

// TitleResult is a title served from the database.
type TitleResult struct {
	Title        *db.FullTitle
	Stale        bool      // served while a refresh runs or after it failed
	ExpiresAt    time.Time // meaningful when !Stale
	LastModified time.Time // later of uat and fast_uat
}

// Fetch returns the stored title for id. Stored data is always served at
// once: a stale part starts a background refresh and the result is marked
// Stale. Only a title with nothing stored waits for the plugin. Errors are
// ErrInvalidID, ErrNotFound, ErrUpstream.
func (s *Service) Fetch(ctx context.Context, id string) (*TitleResult, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrInvalidID
	}
	res, err := s.load(id)
	if db.IsNotFound(err) {
		return s.fetchMissing(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// load reads the stored title, judges its freshness and starts the refresh a
// stale part needs.
func (s *Service) load(id string) (*TitleResult, error) {
	ft, st, err := s.store.GetTitleAndState(id)
	if err != nil {
		return nil, err
	}
	f := evaluate(st, id, time.Now())
	res := &TitleResult{Title: ft, ExpiresAt: f.ExpiresAt, LastModified: st.UpdatedAt}
	if st.FastUpdatedAt.After(res.LastModified) {
		res.LastModified = st.FastUpdatedAt
	}
	switch {
	case f.SlowStale:
		s.startRefresh("full:", id)
		res.Stale = true
	case f.FastStale:
		s.startRefresh("fast:", f.FastID)
		res.Stale = true
	}
	return res, nil
}

func (s *Service) fetchMissing(ctx context.Context, id string) (*TitleResult, error) {
	if missing, err := s.store.IsMissingFresh(id, missingTombstone); err != nil {
		return nil, err
	} else if missing {
		return nil, ErrNotFound
	}
	if s.holds.blocked("fetch:"+id, time.Now()) {
		return nil, ErrUpstream
	}
	// The shared call runs detached from any single caller: one client
	// disconnecting must not fail every waiter. Each caller still stops
	// waiting when its own ctx ends. Every plugin call in it has a deadline.
	ch := s.group.DoChan("full:"+id, func() (any, error) {
		return nil, s.fullRefresh(context.WithoutCancel(ctx), id)
	})
	select {
	case <-ctx.Done():
		return nil, ErrUpstream
	case r := <-ch:
		if r.Err != nil {
			if errors.Is(r.Err, ErrUpstream) {
				s.holds.set("fetch:"+id, failedFetchHold, time.Now())
			}
			return nil, r.Err
		}
	}
	res, err := s.load(id)
	if db.IsNotFound(err) {
		return nil, ErrUpstream
	}
	return res, err
}

// startRefresh queues a background refresh of id unless one is already
// queued or running, or a recent one failed. When the queue is full the
// refresh is dropped; the next request for the stale title queues it again.
// kind is "full:" or "fast:".
func (s *Service) startRefresh(kind, id string) {
	key := kind + id
	if s.holds.blocked(key, time.Now()) {
		return
	}
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	if s.running[key] {
		return
	}
	s.running[key] = true
	s.bg.Add(1)
	if _, ok := s.refreshPool.TrySubmit(func() { s.runRefresh(kind, id) }); !ok {
		delete(s.running, key)
		s.bg.Done()
	}
}

func (s *Service) runRefresh(kind, id string) {
	key := kind + id
	defer s.bg.Done()
	defer func() {
		s.runningMu.Lock()
		delete(s.running, key)
		s.runningMu.Unlock()
	}()
	_, err, _ := s.group.Do(key, func() (any, error) {
		ctx := context.Background()
		if kind == "full:" {
			return nil, s.fullRefresh(ctx, id)
		}
		return nil, s.fastRefresh(ctx, id)
	})
	switch {
	case err == nil:
		s.holds.clear(key)
	case errors.Is(err, ErrNotFound):
		// fullRefresh and fastRefresh already removed the title.
	default:
		s.holds.set(key, failedRefreshHold, time.Now())
		log.Warn("background refresh failed", "id", id, "kind", kind, "error", err)
	}
}

// removeTitle deletes a title IMDb no longer has and remembers the 404.
func (s *Service) removeTitle(id string) error {
	if err := s.store.RemoveTitle(id); err != nil {
		return err
	}
	return ErrNotFound
}

// fullRefresh fetches the whole title and stores it.
func (s *Service) fullRefresh(ctx context.Context, id string) error {
	titleCtx, cancel := context.WithTimeout(ctx, titleTimeout)
	title, err := s.plugin.GetTitle(titleCtx, id)
	cancel()
	if err != nil {
		if errors.Is(err, imdbplugin.ErrNotFound) {
			return s.removeTitle(id)
		}
		return ErrUpstream
	}
	if title == nil {
		return ErrUpstream
	}
	// Store under the requested id; a mismatched plugin id would
	// otherwise orphan a row the read-back misses.
	title.Title.ID = id
	if err := s.checkEpisodesKept(id, title.Title.IsSeries && !hasEpisodes(title.Seasons)); err != nil {
		return err
	}
	if err := s.enrichEpisodes(ctx, title); err != nil {
		return err
	}
	return s.store.UpsertFullTitle(title)
}

// fastRefresh refreshes the fast part: for a series, its episode list (with
// details for new, undated and recently aired episodes) and the ratings of
// the series and its episodes; for any other title, its rating. Any failed
// call fails the refresh and nothing is written.
func (s *Service) fastRefresh(ctx context.Context, id string) error {
	st, err := s.store.GetRefreshState(id)
	if db.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	fr := &db.FastRefresh{ID: id, Series: st.IsSeries}
	ratingIDs := []string{id}

	if st.IsSeries {
		seasons, err := withAttempts(ctx, func(ctx context.Context) ([]imdbplugin.Season, error) {
			return s.plugin.GetEpisodeList(ctx, id)
		})
		if err != nil {
			if errors.Is(err, imdbplugin.ErrNotFound) {
				return s.removeTitle(id)
			}
			return ErrUpstream
		}
		dates, err := s.store.ListEpisodeDates(id)
		if err != nil {
			return err
		}
		if !hasEpisodes(seasons) && len(dates) > 0 {
			return errEmptyEpisodeList
		}
		now := time.Now()
		var detailIDs []string
		for _, season := range seasons {
			for _, e := range season.Episodes {
				if e.ID == "" {
					continue
				}
				fr.Episodes = append(fr.Episodes, db.EpisodeRef{ID: e.ID, Season: season.Number, Episode: e.Episode, Plot: e.Description})
				ratingIDs = append(ratingIDs, e.ID)
				date, stored := dates[e.ID]
				if !stored || date == "" || recentlyAired(date, now) {
					detailIDs = append(detailIDs, e.ID)
				}
			}
		}
		if len(detailIDs) > 0 {
			summaries, err := s.fetchSummaries(ctx, detailIDs)
			if err != nil {
				return err
			}
			for _, sum := range summaries {
				fr.Details = append(fr.Details, sum)
			}
		}
	}

	fr.Ratings, err = s.fetchRatings(ctx, ratingIDs)
	if err != nil {
		return err
	}
	return s.store.ApplyFastRefresh(fr)
}

// errEmptyEpisodeList fails a refresh whose episode list came back empty for
// a series that has stored episodes: one bad IMDb response must not delete
// them. It is an upstream failure, so the refresh is held off.
var errEmptyEpisodeList = fmt.Errorf("%w: empty episode list for a series with stored episodes", ErrUpstream)

func hasEpisodes(seasons []imdbplugin.Season) bool {
	for _, season := range seasons {
		if len(season.Episodes) > 0 {
			return true
		}
	}
	return false
}

// checkEpisodesKept returns errEmptyEpisodeList when empty is set and the
// series has stored episodes.
func (s *Service) checkEpisodesKept(id string, empty bool) error {
	if !empty {
		return nil
	}
	dates, err := s.store.ListEpisodeDates(id)
	if err != nil {
		return err
	}
	if len(dates) > 0 {
		return errEmptyEpisodeList
	}
	return nil
}

// withAttempts calls fn up to episodeBatchAttempts times, each attempt with
// its own episodeBatchTimeout. A not-found result is not retried.
func withAttempts[T any](ctx context.Context, fn func(context.Context) (T, error)) (v T, err error) {
	for range episodeBatchAttempts {
		attemptCtx, cancel := context.WithTimeout(ctx, episodeBatchTimeout)
		v, err = fn(attemptCtx)
		cancel()
		if err == nil || errors.Is(err, imdbplugin.ErrNotFound) || ctx.Err() != nil {
			return v, err
		}
	}
	return v, err
}

// inBatches calls fn for consecutive batches of at most size ids, at most
// episodeConcurrency at a time. Any failed batch fails the whole call.
func inBatches(ctx context.Context, ids []string, size int, fn func(ctx context.Context, batch []string) error) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(episodeConcurrency)
	for start := 0; start < len(ids); start += size {
		batch := ids[start:min(start+size, len(ids))]
		g.Go(func() error { return fn(gctx, batch) })
	}
	return g.Wait()
}

// fetchSummaries fetches episode summaries in batches, by episode id.
func (s *Service) fetchSummaries(ctx context.Context, ids []string) (map[string]imdbplugin.TitleSummary, error) {
	out := make(map[string]imdbplugin.TitleSummary, len(ids))
	var mu sync.Mutex
	err := inBatches(ctx, ids, episodeBatchSize, func(ctx context.Context, batch []string) error {
		eps, err := withAttempts(ctx, func(ctx context.Context) ([]imdbplugin.TitleSummary, error) {
			return s.plugin.GetEpisodes(ctx, batch)
		})
		if err != nil {
			return ErrUpstream
		}
		mu.Lock()
		for _, e := range eps {
			out[e.ID] = e
		}
		mu.Unlock()
		return nil
	})
	return out, err
}

// fetchRatings fetches ratings in batches of imdbplugin.MaxRatingBatch.
func (s *Service) fetchRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error) {
	var out []imdbplugin.Rating
	var mu sync.Mutex
	err := inBatches(ctx, ids, imdbplugin.MaxRatingBatch, func(ctx context.Context, batch []string) error {
		ratings, err := withAttempts(ctx, func(ctx context.Context) ([]imdbplugin.Rating, error) {
			return s.plugin.GetRatings(ctx, batch)
		})
		if err != nil {
			return ErrUpstream
		}
		mu.Lock()
		out = append(out, ratings...)
		mu.Unlock()
		return nil
	})
	return out, err
}
