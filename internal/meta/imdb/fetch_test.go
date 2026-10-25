package imdb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
	"github.com/alitto/pond/v2"
)

const day = 24 * time.Hour

// ageTitle moves the title's slow (uat) and fast (fast_uat) refresh times
// back by the given amounts.
func ageTitle(t *testing.T, store *db.DB, id string, slow, fast time.Duration) {
	t.Helper()
	now := time.Now()
	if _, err := store.Exec(`UPDATE title SET uat = ?, fast_uat = ? WHERE id = ?`,
		now.Add(-slow).Unix(), now.Add(-fast).Unix(), id); err != nil {
		t.Fatal(err)
	}
}

func intp(n int) *int { return &n }

func storeEpisodeTitle(t *testing.T, store *db.DB, id string) {
	t.Helper()
	if err := store.UpsertFullTitle(&db.FullTitle{Title: db.Title{
		TitleSummary: db.TitleSummary{ID: id, Name: "Pilot"}, Type: "tvEpisode",
	}}); err != nil {
		t.Fatal(err)
	}
}

func TestFetchMissingWaitsForFullFetch(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, _ := openService(t, plugin)
	ctx := context.Background()

	res, err := svc.Fetch(ctx, "tt0903747")
	if err != nil || res.Stale || res.Title.Title.Name != "Breaking Bad" {
		t.Fatalf("miss: %+v %v", res, err)
	}
	if plugin.calls != 1 {
		t.Fatalf("expected 1 plugin call, got %d", plugin.calls)
	}

	plugin.err = errors.New("boom")
	res, err = svc.Fetch(ctx, "tt0903747")
	svc.bg.Wait()
	if err != nil || res.Stale || res.Title.Title.Name != "Breaking Bad" {
		t.Fatalf("hit: %+v %v", res, err)
	}
	if plugin.calls != 1 {
		t.Fatalf("expected no new plugin call on fresh hit, got %d", plugin.calls)
	}
	if left := time.Until(res.ExpiresAt); left <= 2*day || left > fastFresh {
		t.Fatalf("expires in %v, want about %v", left, fastFresh)
	}
}

func TestFetchInvalidID(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{})
	if _, err := svc.Fetch(context.Background(), "bad-id"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

func TestFetchNotFoundCached(t *testing.T) {
	plugin := &fakePlugin{err: ErrNotFound}
	svc, _ := openService(t, plugin)
	for i := 0; i < 2; i++ {
		if _, err := svc.Fetch(context.Background(), "tt0000000"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	}
	if plugin.calls != 1 {
		t.Fatalf("expected 1 plugin call (tombstone), got %d", plugin.calls)
	}
}

func TestFetchStaleSlowServesThenRefreshes(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 31*day, 31*day)
	renamed := sampleTitle()
	renamed.Title.Name = "Breaking Bad (renamed)"
	plugin.title = renamed

	res, err := svc.Fetch(ctx, "tt0903747")
	if err != nil || !res.Stale || res.Title.Title.Name != "Breaking Bad" {
		t.Fatalf("stale: %+v %v", res, err)
	}
	svc.bg.Wait()
	res, err = svc.Fetch(ctx, "tt0903747")
	if err != nil || res.Stale || res.Title.Title.Name != "Breaking Bad (renamed)" {
		t.Fatalf("after refresh: %+v %v", res, err)
	}
}

func TestFetchUpstreamErrorWithoutCache(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{err: errors.New("upstream down")})
	if _, err := svc.Fetch(context.Background(), "tt0903747"); !errors.Is(err, ErrUpstream) {
		t.Fatalf("expected ErrUpstream, got %v", err)
	}
}

func TestFetchNilTitleIsUpstreamError(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{title: nil})
	if _, err := svc.Fetch(context.Background(), "tt0903747"); !errors.Is(err, ErrUpstream) {
		t.Fatalf("expected ErrUpstream, got %v", err)
	}
}

func TestFetchForcesRequestedID(t *testing.T) {
	ft := sampleTitle()
	ft.Title.ID = "tt9999999"
	svc, store := openService(t, &fakePlugin{title: ft})
	res, err := svc.Fetch(context.Background(), "tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if res.Title.Title.ID != "tt0903747" {
		t.Fatalf("id = %q, want tt0903747", res.Title.Title.ID)
	}
	if _, err := store.GetFullTitle("tt9999999"); !db.IsNotFound(err) {
		t.Fatalf("orphan row stored under plugin id: %v", err)
	}
}

func TestFetchBackgroundNotFoundDeletesTitle(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 31*day, 31*day)
	plugin.title = nil
	plugin.err = ErrNotFound

	res, err := svc.Fetch(ctx, "tt0903747")
	if err != nil || !res.Stale || res.Title.Title.Name != "Breaking Bad" {
		t.Fatalf("stale: %+v %v", res, err)
	}
	svc.bg.Wait()
	if _, err := svc.Fetch(ctx, "tt0903747"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if plugin.calls != 2 {
		t.Fatalf("plugin calls = %d, want 2", plugin.calls)
	}
	if _, err := store.GetFullTitle("tt0903747"); !db.IsNotFound(err) {
		t.Fatalf("title kept: %v", err)
	}
}

type blockingPlugin struct {
	fakePlugin
	title *imdbplugin.FullTitle
	calls atomic.Int32
}

func (f *blockingPlugin) GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error) {
	f.calls.Add(1)
	time.Sleep(200 * time.Millisecond)
	return f.title, nil
}

func TestFetchConcurrentMissesCallPluginOnce(t *testing.T) {
	plugin := &blockingPlugin{title: sampleTitle()}
	svc, _ := openService(t, plugin)

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.Fetch(context.Background(), "tt0903747")
		}(i)
	}
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
	}
	if got := plugin.calls.Load(); got != 1 {
		t.Fatalf("expected 1 plugin call, got %d", got)
	}
}

func TestFetchPartialEpisodeRefetchesFromPlugin(t *testing.T) {
	plugin := &fakePlugin{byID: map[string]*imdbplugin.FullTitle{
		"tt0903747": sampleTitle(),
		"tt0959621": {Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Name: "Pilot"}, Type: "tvEpisode", YearStart: 2008}},
	}}
	svc, _ := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}

	// The partial row is served at once; the full fetch runs in the background.
	res, err := svc.Fetch(ctx, "tt0959621")
	if err != nil || !res.Stale || !res.Title.Title.Partial {
		t.Fatalf("partial episode: %+v err=%v", res, err)
	}
	svc.bg.Wait()
	if plugin.calls != 2 {
		t.Fatalf("a partial episode must be fetched from the plugin, calls=%d", plugin.calls)
	}

	res, err = svc.Fetch(ctx, "tt0959621")
	svc.bg.Wait()
	if err != nil || res.Stale || res.Title.Title.Partial || res.Title.Title.YearStart != 2008 {
		t.Fatalf("full episode: %+v err=%v", res, err)
	}
	if plugin.calls != 2 {
		t.Fatalf("a full episode is fresh and must be served from cache, calls=%d", plugin.calls)
	}
}

func TestFetchSeriesRefreshDoesNotStaleFullEpisode(t *testing.T) {
	plugin := &fakePlugin{byID: map[string]*imdbplugin.FullTitle{
		"tt0903747": sampleTitle(),
		"tt0959621": {Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Name: "Pilot"}, Type: "tvEpisode"}},
	}}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0959621"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertFullTitle(sampleTitle()); err != nil {
		t.Fatal(err)
	}
	before := plugin.calls
	res, err := svc.Fetch(ctx, "tt0959621")
	svc.bg.Wait()
	if err != nil || res.Stale || res.Title.Title.Partial {
		t.Fatalf("episode: %+v err=%v", res, err)
	}
	if plugin.calls != before {
		t.Fatalf("series upsert must not make a full episode partial or stale, calls %d -> %d", before, plugin.calls)
	}
}

func TestFetchPartialEpisodeServedWhenPluginFails(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, _ := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	plugin.err = errors.New("boom")
	res, err := svc.Fetch(ctx, "tt0959621")
	svc.bg.Wait()
	if err != nil || !res.Stale || !res.Title.Title.Partial || res.Title.Title.Name != "Pilot" {
		t.Fatalf("expected partial row served stale: %+v err=%v", res, err)
	}
}

// gatedPlugin blocks GetTitle until release closes, and reports whether the
// ctx it received was cancelled.
type gatedPlugin struct {
	fakePlugin
	release   chan struct{}
	ctxErr    atomic.Value
	startedCh chan struct{}
}

func (b *gatedPlugin) GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error) {
	close(b.startedCh)
	<-b.release
	b.ctxErr.Store(struct{ err error }{ctx.Err()})
	return b.title, nil
}

func TestRefreshSurvivesFirstCallerCancel(t *testing.T) {
	plugin := &gatedPlugin{
		fakePlugin: fakePlugin{title: sampleTitle()},
		release:    make(chan struct{}),
		startedCh:  make(chan struct{}),
	}
	svc, _ := openService(t, plugin)

	ctx1, cancel1 := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		_, err := svc.Fetch(ctx1, "tt0903747")
		first <- err
	}()
	<-plugin.startedCh

	second := make(chan error, 1)
	go func() {
		_, err := svc.Fetch(context.Background(), "tt0903747")
		second <- err
	}()
	time.Sleep(50 * time.Millisecond)

	cancel1()
	if err := <-first; !errors.Is(err, ErrUpstream) {
		t.Fatalf("first caller err = %v, want ErrUpstream", err)
	}
	close(plugin.release)
	if err := <-second; err != nil {
		t.Fatalf("second caller err = %v, want nil", err)
	}
	if v := plugin.ctxErr.Load().(struct{ err error }); v.err != nil {
		t.Fatalf("plugin ctx cancelled: %v", v.err)
	}
}

func TestFetchStaleFastRunsFastRefreshOnly(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 5*day, 4*day)
	plugin.epList = map[string][]imdbplugin.Season{"tt0903747": {{Number: 1, Episodes: []imdbplugin.Episode{
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621"}, Episode: intp(1)},
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt1054724"}, Episode: intp(2)},
	}}}}
	plugin.episodes = map[string]imdbplugin.TitleSummary{"tt1054724": {ID: "tt1054724", Name: "Cat's in the Bag..."}}
	plugin.ratings = map[string]imdbplugin.Rating{
		"tt0903747": {ID: "tt0903747", Rating: 95, Votes: 2300000},
		"tt0959621": {ID: "tt0959621", Rating: 90, Votes: 41000},
		"tt1054724": {ID: "tt1054724", Rating: 87, Votes: 30000},
	}

	res, err := svc.Fetch(ctx, "tt0903747")
	if err != nil || !res.Stale {
		t.Fatalf("fast-stale fetch: %+v %v", res, err)
	}
	svc.bg.Wait()
	if plugin.calls != 1 {
		t.Fatalf("fast refresh called GetTitle: %d calls", plugin.calls)
	}
	ft, err := store.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	eps := ft.Seasons[0].Episodes
	if ft.Title.Rating != 95 || len(eps) != 2 || eps[1].Name != "Cat's in the Bag..." || eps[0].Rating != 90 || eps[1].Votes != 30000 {
		t.Fatalf("after fast refresh: %+v %+v", ft.Title, eps)
	}
	for _, batch := range plugin.ratingCalls {
		if len(batch) > imdbplugin.MaxRatingBatch {
			t.Fatalf("rating batch of %d", len(batch))
		}
	}
	res, err = svc.Fetch(ctx, "tt0903747")
	if err != nil || res.Stale {
		t.Fatalf("after fast refresh the title is fresh: %+v %v", res, err)
	}
}

func TestFetchFastRefreshDetailsOnlyNewAndRecent(t *testing.T) {
	future := time.Now().Add(30 * day).Format(time.DateOnly)
	ft := sampleTitle()
	ft.Seasons[0].Episodes = []imdbplugin.Episode{
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Name: "Pilot", ReleaseDate: "2008-01-20"}, Episode: intp(1)},
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt1054724", Name: "TBA", ReleaseDate: future}, Episode: intp(2)},
	}
	plugin := &fakePlugin{}
	svc, store := openService(t, plugin)
	if err := store.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 5*day, 4*day)
	plugin.epList = map[string][]imdbplugin.Season{"tt0903747": {{Number: 1, Episodes: []imdbplugin.Episode{
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621"}, Episode: intp(1)},
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt1054724"}, Episode: intp(2)},
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt1054725"}, Episode: intp(3)},
	}}}}

	if _, err := svc.Fetch(context.Background(), "tt0903747"); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	var got []string
	for _, batch := range plugin.epIDs {
		got = append(got, batch...)
	}
	sort.Strings(got)
	if fmt.Sprint(got) != "[tt1054724 tt1054725]" {
		t.Fatalf("GetEpisodes ids = %v", got)
	}
}

func TestFetchEpisodeTriggersSeriesFastRefresh(t *testing.T) {
	plugin := &fakePlugin{}
	svc, store := openService(t, plugin)
	storeEpisodeTitle(t, store, "tt0959621")
	if err := store.UpsertFullTitle(sampleTitle()); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 5*day, 4*day)
	plugin.epList = map[string][]imdbplugin.Season{"tt0903747": {{Number: 1, Episodes: []imdbplugin.Episode{
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621"}, Episode: intp(1)},
	}}}}

	res, err := svc.Fetch(context.Background(), "tt0959621")
	if err != nil || !res.Stale {
		t.Fatalf("episode: %+v %v", res, err)
	}
	svc.bg.Wait()
	if n := plugin.epListCalls.Load(); n != 1 {
		t.Fatalf("series episode list calls = %d", n)
	}
	if plugin.calls != 0 {
		t.Fatalf("GetTitle calls = %d", plugin.calls)
	}
}

func TestFetchOrphanEpisodeHasOnlySlowPart(t *testing.T) {
	plugin := &fakePlugin{}
	svc, store := openService(t, plugin)
	storeEpisodeTitle(t, store, "tt0959621")
	ageTitle(t, store, "tt0959621", day, 10*day)
	res, err := svc.Fetch(context.Background(), "tt0959621")
	svc.bg.Wait()
	if err != nil || res.Stale {
		t.Fatalf("orphan episode: %+v %v", res, err)
	}
	if plugin.calls != 0 || plugin.epListCalls.Load() != 0 || len(plugin.ratingCalls) != 0 {
		t.Fatal("orphan episode called the plugin")
	}
}

func TestFetchFailedBackgroundRefreshIsHeldOff(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 31*day, 31*day)
	plugin.err = errors.New("boom")

	for i := 0; i < 2; i++ {
		res, err := svc.Fetch(ctx, "tt0903747")
		svc.bg.Wait()
		if err != nil || !res.Stale || res.Title.Title.Name != "Breaking Bad" {
			t.Fatalf("call %d: %+v %v", i, res, err)
		}
	}
	if plugin.calls != 2 {
		t.Fatalf("plugin calls = %d, want 2 (initial + one held-off refresh)", plugin.calls)
	}
}

func TestFetchFailedFirstFetchIsRemembered(t *testing.T) {
	plugin := &fakePlugin{err: errors.New("boom")}
	svc, _ := openService(t, plugin)
	for i := 0; i < 2; i++ {
		if _, err := svc.Fetch(context.Background(), "tt0903747"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if plugin.calls != 1 {
		t.Fatalf("plugin calls = %d, want 1", plugin.calls)
	}
}

// peakPlugin holds every GetTitle until gate closes and records how many ran
// at once.
type peakPlugin struct {
	fakePlugin
	gate   chan struct{}
	active atomic.Int32
	peak   atomic.Int32
	total  atomic.Int32
}

func (p *peakPlugin) GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error) {
	p.total.Add(1)
	n := p.active.Add(1)
	defer p.active.Add(-1)
	for {
		peak := p.peak.Load()
		if n <= peak || p.peak.CompareAndSwap(peak, n) {
			break
		}
	}
	<-p.gate
	return &imdbplugin.FullTitle{Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: id, Name: "Movie"}, Type: "movie", YearStart: 2001}}, nil
}

func TestFetchStaleBurstNeverBlocks(t *testing.T) {
	plugin := &peakPlugin{gate: make(chan struct{})}
	svc, store := openService(t, plugin)
	ids := make([]string, 10)
	for i := range ids {
		ids[i] = fmt.Sprintf("tt%07d", i+1)
		if err := store.UpsertFullTitle(&db.FullTitle{Title: db.Title{
			TitleSummary: db.TitleSummary{ID: ids[i], Name: "Movie"}, Type: "movie", YearStart: 2001,
		}}); err != nil {
			t.Fatal(err)
		}
		ageTitle(t, store, ids[i], 31*day, 31*day)
	}

	// The plugin is held for the whole loop, so a Fetch that waited on it
	// would never return.
	for _, id := range ids {
		res, err := svc.Fetch(context.Background(), id)
		if err != nil || !res.Stale {
			t.Fatalf("%s: %+v %v", id, res, err)
		}
	}
	close(plugin.gate)
	svc.bg.Wait()
	// Refreshes beyond the workers wait in the queue and all run.
	if n := plugin.total.Load(); n != int32(len(ids)) {
		t.Fatalf("refreshes run = %d, want %d", n, len(ids))
	}
	if n := plugin.peak.Load(); n != maxBackgroundRefreshes {
		t.Fatalf("concurrent refreshes = %d, want %d", n, maxBackgroundRefreshes)
	}
}

func TestFetchRefreshDroppedWhenQueueIsFull(t *testing.T) {
	plugin := &peakPlugin{gate: make(chan struct{})}
	svc, store := openService(t, plugin)
	svc.refreshPool = pond.NewPool(1, pond.WithQueueSize(1))
	ids := []string{"tt0000001", "tt0000002", "tt0000003"}
	for _, id := range ids {
		if err := store.UpsertFullTitle(&db.FullTitle{Title: db.Title{
			TitleSummary: db.TitleSummary{ID: id, Name: "Movie"}, Type: "movie", YearStart: 2001,
		}}); err != nil {
			t.Fatal(err)
		}
		ageTitle(t, store, id, 31*day, 31*day)
	}
	for _, id := range ids {
		if _, err := svc.Fetch(context.Background(), id); err != nil {
			t.Fatal(err)
		}
		// Let the worker pick up the first refresh before queueing the next.
		deadline := time.Now().Add(5 * time.Second)
		for id == ids[0] && plugin.active.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
	}
	close(plugin.gate)
	svc.bg.Wait()
	if n := plugin.total.Load(); n != 2 {
		t.Fatalf("refreshes run = %d, want 2 (one running, one queued, one dropped)", n)
	}
	// The dropped title is queued again by its next request.
	plugin.gate = make(chan struct{})
	close(plugin.gate)
	if _, err := svc.Fetch(context.Background(), ids[2]); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	if n := plugin.total.Load(); n != 3 {
		t.Fatalf("refreshes run = %d, want 3", n)
	}
}

func TestFastRefreshEmptyEpisodeListKeepsStoredEpisodes(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 5*day, 4*day)
	plugin.epList = map[string][]imdbplugin.Season{"tt0903747": {}}

	for i := 0; i < 2; i++ {
		if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
			t.Fatal(err)
		}
		svc.bg.Wait()
	}
	ft, err := store.GetFullTitle("tt0903747")
	if err != nil || len(ft.Seasons) != 1 || len(ft.Seasons[0].Episodes) != 1 {
		t.Fatalf("stored episodes lost: %+v %v", ft, err)
	}
	if n := plugin.epListCalls.Load(); n != 1 {
		t.Fatalf("episode list calls = %d, want 1 (failure held off)", n)
	}
}

func TestFullRefreshWithoutEpisodesKeepsStoredEpisodes(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 31*day, 31*day)
	empty := sampleTitle()
	empty.Seasons = []imdbplugin.Season{}
	plugin.title = empty

	if _, err := svc.Fetch(ctx, "tt0903747"); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	ft, err := store.GetFullTitle("tt0903747")
	if err != nil || len(ft.Seasons) != 1 {
		t.Fatalf("stored episodes lost: %+v %v", ft, err)
	}
}
