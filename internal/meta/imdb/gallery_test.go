package imdb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
	"github.com/alitto/pond/v2"
)

func galleryPages(n, perPage int) map[string]*imdbplugin.ImagePage {
	pages := map[string]*imdbplugin.ImagePage{}
	for p := 0; p*perPage < n; p++ {
		page := &imdbplugin.ImagePage{}
		for i := p * perPage; i < min(n, (p+1)*perPage); i++ {
			page.Images = append(page.Images, imdbplugin.Image{
				ID: fmt.Sprintf("rm%d", i), URL: fmt.Sprintf("http://img/%d", i), Width: 100, Height: 200,
			})
		}
		cursor := ""
		if p > 0 {
			cursor = fmt.Sprintf("c%d", p)
		}
		if (p+1)*perPage < n {
			page.Next = fmt.Sprintf("c%d", p+1)
		}
		pages[cursor] = page
	}
	return pages
}

func storeCompleteGallery(t *testing.T, store *db.DB, id string, n int, age time.Duration) {
	t.Helper()
	var images []db.Image
	for _, page := range galleryPages(n, n) {
		images = append(images, page.Images...)
	}
	if err := store.AddGalleryImages(id, images); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteGallery(id); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`UPDATE title_gallery SET uat = ? WHERE title_id = ?`, time.Now().Add(-age).Unix(), id); err != nil {
		t.Fatal(err)
	}
}

func imageIDs(p *GalleryPage) []string {
	var ids []string
	for _, img := range p.Images {
		ids = append(ids, img.ID)
	}
	return ids
}

func TestImagesInvalidID(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{})
	if _, err := svc.Images(context.Background(), "bad", 1); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestImagesFirstRequestSyncsAllPagesThenServesFromCache(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(1200, 250)}
	svc, store := openService(t, plugin)
	ctx := context.Background()

	got, err := svc.Images(ctx, "tt0903747", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Page.Images) != galleryPageSize || got.Page.Images[0].ID != "rm0" {
		t.Fatalf("first page: %d images", len(got.Page.Images))
	}
	svc.bg.Wait()
	if n := plugin.imgCalls.Load(); n != 5 {
		t.Fatalf("plugin pages fetched = %d, want 5", n)
	}

	got, err = svc.Images(ctx, "tt0903747", 2)
	if err != nil || got.Stale || !got.Page.Complete || got.Page.Total != 1200 || got.Page.Pages != 3 || got.Page.Images[0].ID != "rm500" {
		t.Fatalf("page 2: %+v %v", got, err)
	}
	got, err = svc.Images(ctx, "tt0903747", 3)
	if err != nil || len(got.Page.Images) != 200 || got.Page.Images[199].ID != "rm1199" {
		t.Fatalf("page 3: %d images, %v", len(got.Page.Images), err)
	}
	if n := plugin.imgCalls.Load(); n != 5 {
		t.Fatalf("cached requests called the plugin: %d calls", n)
	}
	_ = store
}

func TestImagesFirstRequestWaitsForFullFirstPage(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{pages: galleryPages(1000, 250)})
	got, err := svc.Images(context.Background(), "tt0903747", 1)
	if err != nil || len(got.Page.Images) != galleryPageSize {
		t.Fatalf("first page: %d images, %v", len(got.Page.Images), err)
	}
	svc.bg.Wait()
}

func TestImagesFirstRequestDoesNotWaitForLaterPages(t *testing.T) {
	gate := make(chan struct{})
	plugin := &fakePlugin{pages: galleryPages(1000, 500), imgGate: gate}
	svc, _ := openService(t, plugin)

	got, err := svc.Images(context.Background(), "tt0903747", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Page.Complete || !got.Stale || len(got.Page.Images) != 500 {
		t.Fatalf("expected the first page only while syncing: complete=%v stale=%v n=%d", got.Page.Complete, got.Stale, len(got.Page.Images))
	}

	close(gate)
	svc.bg.Wait()
	got, err = svc.Images(context.Background(), "tt0903747", 2)
	if err != nil || !got.Page.Complete || len(got.Page.Images) != 500 {
		t.Fatalf("after sync: %+v, %v", got.Page.Complete, err)
	}
}

func TestImagesConcurrentFirstRequestsSyncOnce(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(4, 2)}
	svc, _ := openService(t, plugin)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Images(context.Background(), "tt0903747", 1); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	svc.bg.Wait()
	if n := plugin.imgCalls.Load(); n != 2 {
		t.Fatalf("plugin pages fetched = %d, want 2 (one sync)", n)
	}
}

func TestImagesErrors(t *testing.T) {
	svc, _ := openService(t, &fakePlugin{err: imdbplugin.ErrNotFound})
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	plugin := &fakePlugin{err: errors.New("boom")}
	svc, _ = openService(t, plugin)
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
	// A failed sync suppresses another attempt for a while.
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream", err)
	}
	svc.bg.Wait()
	if n := plugin.imgCalls.Load(); n != 1 {
		t.Fatalf("plugin called %d times, want 1 (retry suppressed)", n)
	}
}

func TestImagesPagesAndPastEnd(t *testing.T) {
	plugin := &fakePlugin{}
	svc, store := openService(t, plugin)
	storeCompleteGallery(t, store, "tt0903747", 1200, day)

	got, err := svc.Images(context.Background(), "tt0903747", 3)
	if err != nil || len(got.Page.Images) != 200 || got.Page.Pages != 3 || got.Page.Total != 1200 {
		t.Fatalf("page 3: %d images, pages %d, %v", len(got.Page.Images), got.Page.Pages, err)
	}
	for _, page := range []int{4, 1000000} {
		got, err := svc.Images(context.Background(), "tt0903747", page)
		if err != nil || got.Page.Images == nil || len(got.Page.Images) != 0 {
			t.Fatalf("page %d: %+v, %v", page, got, err)
		}
	}
	if plugin.imgCalls.Load() != 0 {
		t.Fatal("complete fresh gallery called the plugin")
	}
}

func TestImagesExpiryIsShared(t *testing.T) {
	svc, store := openService(t, &fakePlugin{})
	storeCompleteGallery(t, store, "tt0903747", 600, 10*day)
	st, err := store.GetGallery("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	p1, err1 := svc.Images(context.Background(), "tt0903747", 1)
	p2, err2 := svc.Images(context.Background(), "tt0903747", 2)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	want := st.UpdatedAt.Add(galleryFresh)
	if p1.Stale || p2.Stale || !p1.ExpiresAt.Equal(want) || !p2.ExpiresAt.Equal(want) || !p1.UpdatedAt.Equal(st.UpdatedAt) {
		t.Fatalf("p1 = %+v, p2 = %+v, want expiry %v", p1, p2, want)
	}
}

func TestImagesStaleGalleryServedShort(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(3, 3)}
	svc, store := openService(t, plugin)
	storeCompleteGallery(t, store, "tt0903747", 2, 31*day)

	got, err := svc.Images(context.Background(), "tt0903747", 1)
	if err != nil || !got.Stale || len(got.Page.Images) != 2 {
		t.Fatalf("stale gallery must be served as cached: %+v, %v", got, err)
	}
	svc.bg.Wait()
	got, err = svc.Images(context.Background(), "tt0903747", 1)
	if err != nil || got.Stale || got.Page.Total != 3 {
		t.Fatalf("after resync: %+v, %v", got, err)
	}
}

func TestImagesRestartsIncompleteGallery(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(4, 2)}
	svc, store := openService(t, plugin)
	// A sync that died after the first page leaves images but no completion.
	if err := store.AddGalleryImages("tt0903747", plugin.pages[""].Images); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Images(context.Background(), "tt0903747", 1); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	got, total, _ := store.ListGalleryImages("tt0903747", 10, 0)
	if total != 4 || len(got) != 4 {
		t.Fatalf("total = %d, want 4 (no duplicates from the partial run): %+v", total, got)
	}
}

func TestImagesServesStoredImagesWhenSyncFails(t *testing.T) {
	plugin := &fakePlugin{err: errors.New("boom")}
	svc, store := openService(t, plugin)
	if err := store.AddGalleryImages("tt0903747", galleryPages(2, 2)[""].Images); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Images(context.Background(), "tt0903747", 1)
	svc.bg.Wait()
	if err != nil || got.Page.Complete || !got.Stale || len(got.Page.Images) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestImagesResyncRemovesImagesNoLongerListed(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(3, 3)}
	svc, store := openService(t, plugin)
	ctx := context.Background()
	if _, err := svc.Images(ctx, "tt0903747", 1); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	if _, err := store.Exec(`UPDATE title_gallery SET uat = unixepoch() - 365*86400`); err != nil {
		t.Fatal(err)
	}
	plugin.pages = map[string]*imdbplugin.ImagePage{"": {Images: galleryPages(3, 3)[""].Images[1:]}}
	if _, err := svc.Images(ctx, "tt0903747", 1); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	got, total, _ := store.ListGalleryImages("tt0903747", 10, 0)
	if total != 2 || got[0].ID != "rm1" || got[1].ID != "rm2" {
		t.Fatalf("kept = %+v", got)
	}
}

func TestImagesTruncatedGalleryIsNotPruned(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(maxGalleryPages+1, 1)}
	svc, store := openService(t, plugin)
	if err := store.AddGalleryImages("tt0903747", []db.Image{{ID: "rm-old", URL: "http://old"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Images(context.Background(), "tt0903747", 1); err != nil {
		t.Fatal(err)
	}
	svc.bg.Wait()
	got, total, _ := store.ListGalleryImages("tt0903747", 1, 0)
	if total != maxGalleryPages+1 || got[0].ID != "rm-old" {
		t.Fatalf("total = %d, first = %+v", total, got)
	}
}

func TestImagesMissingTitleSkipsPlugin(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(2, 2)}
	svc, store := openService(t, plugin)
	if err := store.MarkMissing("tt0903747"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if n := plugin.imgCalls.Load(); n != 0 {
		t.Fatalf("plugin called %d times for a missing title", n)
	}
}

func TestImagesNotFoundDeletesStoredTitle(t *testing.T) {
	plugin := &fakePlugin{err: imdbplugin.ErrNotFound}
	svc, store := openService(t, plugin)
	if err := store.UpsertFullTitle(sampleTitle()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	svc.bg.Wait()
	if _, err := store.GetFullTitle("tt0903747"); !db.IsNotFound(err) {
		t.Fatalf("title kept: %v", err)
	}
	if missing, err := store.IsMissingFresh("tt0903747", 7*24*time.Hour); err != nil || !missing {
		t.Fatalf("missing = %v, %v", missing, err)
	}
}

// countingPlugin holds every GetImages call until gate closes and records how
// many ran at once.
type countingPlugin struct {
	fakePlugin
	gate   chan struct{}
	active atomic.Int32
	peak   atomic.Int32
	total  atomic.Int32
}

func (p *countingPlugin) GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error) {
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
	return &imdbplugin.ImagePage{Images: []imdbplugin.Image{{ID: "rm-" + id, URL: "http://img/" + id}}}, nil
}

func TestGallerySyncsAreCapped(t *testing.T) {
	plugin := &countingPlugin{gate: make(chan struct{})}
	svc, store := openService(t, plugin)

	ids := make([]string, maxGallerySyncs+3)
	for i := range ids {
		ids[i] = fmt.Sprintf("tt%d", i+1)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		svc.Images(ctx, ids[i], 1)
		cancel()
	}
	deadline := time.Now().Add(5 * time.Second)
	for plugin.active.Load() < maxGallerySyncs && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if n := plugin.peak.Load(); n != maxGallerySyncs {
		t.Fatalf("concurrent syncs = %d, want %d", n, maxGallerySyncs)
	}
	close(plugin.gate)
	svc.bg.Wait()
	for _, id := range ids {
		if _, err := store.GetGallery(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
}

func TestGalleryResyncsQueueBeyondWorkers(t *testing.T) {
	plugin := &countingPlugin{gate: make(chan struct{})}
	svc, store := openService(t, plugin)
	ids := make([]string, maxGallerySyncs+2)
	for i := range ids {
		ids[i] = fmt.Sprintf("tt%d", i+1)
		storeCompleteGallery(t, store, ids[i], 1, 31*day)
	}
	for _, id := range ids {
		got, err := svc.Images(context.Background(), id, 1)
		if err != nil || !got.Stale {
			t.Fatalf("%s: %+v %v", id, got, err)
		}
	}
	close(plugin.gate)
	svc.bg.Wait()
	if n := plugin.total.Load(); n != int32(len(ids)) {
		t.Fatalf("resyncs run = %d, want %d", n, len(ids))
	}
	if n := plugin.peak.Load(); n != maxGallerySyncs {
		t.Fatalf("concurrent resyncs = %d, want %d", n, maxGallerySyncs)
	}
}

func TestGalleryFirstSyncFailsWhenQueueIsFull(t *testing.T) {
	plugin := &countingPlugin{gate: make(chan struct{})}
	svc, _ := openService(t, plugin)
	svc.galleryPool = pond.NewPool(1, pond.WithQueueSize(1))
	for i, id := range []string{"tt1", "tt2"} {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		svc.Images(ctx, id, 1)
		cancel()
		deadline := time.Now().Add(5 * time.Second)
		for i == 0 && plugin.active.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
	}
	if _, err := svc.Images(context.Background(), "tt3", 1); !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v, want ErrUpstream with a full queue", err)
	}
	close(plugin.gate)
	svc.bg.Wait()
}

func TestGallerySyncDoesNotRecreateRemovedTitle(t *testing.T) {
	gate := make(chan struct{})
	plugin := &fakePlugin{pages: galleryPages(1000, 500), imgGate: gate}
	svc, store := openService(t, plugin)
	if _, err := svc.Images(context.Background(), "tt0903747", 1); err != nil {
		t.Fatal(err)
	}
	// A background title refresh finds the title gone while the sync runs.
	if err := store.RemoveTitle("tt0903747"); err != nil {
		t.Fatal(err)
	}
	close(gate)
	svc.bg.Wait()
	if _, err := store.GetGallery("tt0903747"); !db.IsNotFound(err) {
		t.Fatalf("gallery recreated: %v", err)
	}
	if _, total, _ := store.ListGalleryImages("tt0903747", 10, 0); total != 0 {
		t.Fatalf("gallery rows = %d", total)
	}
	if _, err := svc.Images(context.Background(), "tt0903747", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGalleryResyncSkipsWhenAlreadyFresh(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(2, 2)}
	svc, store := openService(t, plugin)
	// Another resync finished just before this one took the lock.
	storeCompleteGallery(t, store, "tt0903747", 2, time.Minute)
	svc.startGallerySync("tt0903747", true)
	svc.bg.Wait()
	if n := plugin.imgCalls.Load(); n != 0 {
		t.Fatalf("resync of a fresh gallery called the plugin %d times", n)
	}
}
