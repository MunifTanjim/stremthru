package imdb

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
)

type fakePlugin struct {
	calls int
	// episodes answers GetEpisodes; epCalls records each batch size.
	episodes map[string]imdbplugin.TitleSummary
	epErr    error
	// epFailures fails that many GetEpisodes calls before answering.
	epFailures int
	epMu       sync.Mutex
	epCalls    []int
	epIDs      [][]string
	page       *imdbplugin.ImagePage
	// pages answers GetImages by cursor when set; imgCalls counts calls and
	// imgGate, if set, holds every page after the first until it is closed.
	pages    map[string]*imdbplugin.ImagePage
	imgCalls atomic.Int32
	imgGate  chan struct{}
	title    *imdbplugin.FullTitle
	byID     map[string]*imdbplugin.FullTitle
	err      error

	epList      map[string][]imdbplugin.Season
	epListErr   error
	epListCalls atomic.Int32
	ratings     map[string]imdbplugin.Rating
	ratingsErr  error
	ratingMu    sync.Mutex
	ratingCalls [][]string
}

func (f *fakePlugin) GetEpisodeList(ctx context.Context, id string) ([]imdbplugin.Season, error) {
	f.epListCalls.Add(1)
	if f.epListErr != nil {
		return nil, f.epListErr
	}
	return f.epList[id], nil
}

func (f *fakePlugin) GetRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error) {
	f.ratingMu.Lock()
	f.ratingCalls = append(f.ratingCalls, ids)
	f.ratingMu.Unlock()
	if f.ratingsErr != nil {
		return nil, f.ratingsErr
	}
	var out []imdbplugin.Rating
	for _, id := range ids {
		if r, ok := f.ratings[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakePlugin) GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error) {
	f.calls++
	if t, ok := f.byID[id]; ok {
		return t, f.err
	}
	return f.title, f.err
}

func (f *fakePlugin) GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error) {
	f.imgCalls.Add(1)
	if f.imgGate != nil && after != "" {
		<-f.imgGate
	}
	if f.pages != nil {
		return f.pages[after], f.err
	}
	return f.page, f.err
}

func (f *fakePlugin) GetEpisodes(ctx context.Context, ids []string) ([]imdbplugin.TitleSummary, error) {
	f.epMu.Lock()
	f.epCalls = append(f.epCalls, len(ids))
	f.epIDs = append(f.epIDs, ids)
	fail := len(f.epCalls) <= f.epFailures
	f.epMu.Unlock()
	if fail {
		return nil, errors.New("flaky")
	}
	if f.epErr != nil {
		return nil, f.epErr
	}
	var out []imdbplugin.TitleSummary
	for _, id := range ids {
		if sum, ok := f.episodes[id]; ok {
			out = append(out, sum)
		}
	}
	return out, nil
}

func openService(t *testing.T, plugin Provider) (*Service, *db.DB) {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "imdb.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	return NewService(store, plugin), store
}

func sampleTitle() *imdbplugin.FullTitle {
	rating := 77
	one := 1
	return &imdbplugin.FullTitle{
		Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: "tt0903747",
			Name:   "Breaking Bad",
			Rating: rating, Votes: 100}, Type: "tvSeries", IsSeries: true,
			YearStart: 2008,
		},
		Seasons: []imdbplugin.Season{{
			Number: 1,
			Episodes: []imdbplugin.Episode{
				{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Name: "Pilot"}, Episode: &one},
			},
		}},
	}
}

func seriesWithEpisodes(n int) *imdbplugin.FullTitle {
	ft := sampleTitle()
	ft.Seasons[0].Episodes = nil
	for i := 1; i <= n; i++ {
		num := i
		ft.Seasons[0].Episodes = append(ft.Seasons[0].Episodes, imdbplugin.Episode{
			TitleSummary: imdbplugin.TitleSummary{ID: fmt.Sprintf("tt9%06d", i)}, Episode: &num,
		})
	}
	return ft
}

func TestFetchEnrichesEpisodesInBatches(t *testing.T) {
	ft := seriesWithEpisodes(imdbplugin.MaxEpisodeBatch + 3)
	plugin := &fakePlugin{title: ft, episodes: map[string]imdbplugin.TitleSummary{}}
	for _, e := range ft.Seasons[0].Episodes {
		plugin.episodes[e.ID] = imdbplugin.TitleSummary{ID: e.ID, Name: "name " + e.ID}
	}
	svc, _ := openService(t, plugin)
	res, err := svc.Fetch(context.Background(), "tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	eps := res.Title.Seasons[0].Episodes
	if len(eps) != imdbplugin.MaxEpisodeBatch+3 || eps[0].Name != "name "+eps[0].ID {
		t.Fatalf("episodes = %+v", eps)
	}
	for _, n := range plugin.epCalls {
		if n > imdbplugin.MaxEpisodeBatch {
			t.Fatalf("batch of %d ids, max %d", n, imdbplugin.MaxEpisodeBatch)
		}
	}
	if len(plugin.epCalls) != 2 {
		t.Fatalf("batches = %v", plugin.epCalls)
	}
}

func TestFetchRetriesFailedEpisodeBatchOnce(t *testing.T) {
	ft := seriesWithEpisodes(2)
	plugin := &fakePlugin{title: ft, epFailures: 1, episodes: map[string]imdbplugin.TitleSummary{
		"tt9000001": {ID: "tt9000001", Name: "One"},
	}}
	svc, _ := openService(t, plugin)
	res, err := svc.Fetch(context.Background(), "tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if res.Title.Seasons[0].Episodes[0].Name != "One" || len(plugin.epCalls) != 2 {
		t.Fatalf("episodes = %+v, calls = %v", res.Title.Seasons[0].Episodes, plugin.epCalls)
	}

	plugin = &fakePlugin{title: ft, epFailures: 2}
	svc, _ = openService(t, plugin)
	if _, err := svc.Fetch(context.Background(), "tt0903747"); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream after two failures", err)
	}
}
