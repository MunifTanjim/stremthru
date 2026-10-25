package imdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/server"
)

func serve(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	req = server.SetReqCtx(req, &server.ReqCtx{
		StartTime: time.Now(),
		RequestId: "test",
		ReqMethod: req.Method,
		ReqPath:   req.URL.Path,
		ReqQuery:  req.URL.Query(),
		Log:       logger.Scoped("test"),
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func openRouterService(t *testing.T, plugin *fakePlugin) (*Service, *db.DB) {
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

func openRouter(t *testing.T, plugin *fakePlugin) *http.ServeMux {
	t.Helper()
	svc, _ := openRouterService(t, plugin)
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)
	return mux
}

func TestHandler200HeadersAndETag(t *testing.T) {
	mux := openRouter(t, &fakePlugin{title: sampleTitle()})

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	cc := rec.Header().Get("Cache-Control")
	if !strings.HasPrefix(cc, "public, max-age=") || !strings.Contains(cc, "stale-while-revalidate") {
		t.Fatalf("Cache-Control = %q", cc)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" || rec.Header().Get("Last-Modified") == "" {
		t.Fatalf("missing ETag/Last-Modified: %v", rec.Header())
	}
	if !strings.Contains(rec.Body.String(), "Breaking Bad") {
		t.Fatalf("body missing title: %s", rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil)
	req.Header.Set("If-None-Match", etag)
	rec2 := serve(mux, req)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Fatalf("304 must have empty body, got %q", rec2.Body.String())
	}
}

func TestHandler400And404(t *testing.T) {
	mux := openRouter(t, &fakePlugin{err: ErrNotFound})

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/bad", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	rec = serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0000000", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=60") {
		t.Fatalf("404 Cache-Control = %q", cc)
	}

	rec = serve(mux, httptest.NewRequest(http.MethodPost, "/v0/meta/imdb/titles/tt0903747", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandler502(t *testing.T) {
	mux := openRouter(t, &fakePlugin{err: errors.New("plugin down")})

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	rec = serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("images status = %d, want 502", rec.Code)
	}
}

func TestHandlerIfNoneMatchForms(t *testing.T) {
	mux := openRouter(t, &fakePlugin{title: sampleTitle()})
	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	etag := rec.Header().Get("ETag")

	for _, inm := range []string{"*", `"x", ` + etag, strings.TrimPrefix(etag, "W/")} {
		req := httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil)
		req.Header.Set("If-None-Match", inm)
		if rec := serve(mux, req); rec.Code != http.StatusNotModified {
			t.Fatalf("If-None-Match %q: status = %d, want 304", inm, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil)
	req.Header.Set("If-None-Match", `W/"1"`)
	if rec := serve(mux, req); rec.Code != http.StatusOK {
		t.Fatalf("non-matching etag: status = %d, want 200", rec.Code)
	}
}

func TestHandlerStaleNotCachedLong(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openRouterService(t, plugin)
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)

	if _, err := svc.Fetch(context.Background(), "tt0903747"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Exec(`UPDATE title SET uat = unixepoch() - 365*86400`); err != nil {
		t.Fatal(err)
	}
	plugin.err = errors.New("down")

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); strings.Contains(cc, "max-age=86400") {
		t.Fatalf("stale response Cache-Control = %q", cc)
	}
	if rec.Header().Get("Warning") == "" {
		t.Fatal("missing Warning header")
	}
}

func TestHandlerImages(t *testing.T) {
	plugin := &fakePlugin{pages: galleryPages(700, 700)}
	svc, _ := openRouterService(t, plugin)
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"rm0"`) || strings.Contains(rec.Body.String(), `"rm500"`) {
		t.Fatalf("page 1 body has the wrong images")
	}
	svc.bg.Wait()

	rec = serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images?page=2&limit=10", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"rm500"`) || strings.Contains(rec.Body.String(), `"rm499"`) ||
		!strings.Contains(rec.Body.String(), `"total":700`) || !strings.Contains(rec.Body.String(), `"pages":2`) {
		t.Fatalf("page 2: %d %s", rec.Code, rec.Body.String()[:min(300, rec.Body.Len())])
	}
}

func TestHandlerImagesBadParams(t *testing.T) {
	mux := openRouter(t, &fakePlugin{pages: galleryPages(1, 1)})
	for _, q := range []string{"?page=0", "?page=-1", "?page=x"} {
		rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images"+q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", q, rec.Code)
		}
	}
	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/bad/images", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func maxAge(t *testing.T, cc string) int {
	t.Helper()
	var n int
	if _, err := fmt.Sscanf(cc, "public, max-age=%d", &n); err != nil {
		t.Fatalf("Cache-Control = %q", cc)
	}
	return n
}

func TestTitleMaxAgeFollowsExpiry(t *testing.T) {
	svc, store := openRouterService(t, &fakePlugin{title: sampleTitle()})
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)
	if _, err := svc.Fetch(context.Background(), "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", day, day)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	if n := maxAge(t, rec.Header().Get("Cache-Control")); n < 2*86400-5 || n > 2*86400 {
		t.Fatalf("max-age = %d, want about %d", n, 2*86400)
	}
}

func TestCacheControlRoundsUp(t *testing.T) {
	now := time.Now()
	if cc := titleCacheControl(now.Add(500*time.Millisecond), now); !strings.HasPrefix(cc, "public, max-age=1,") {
		t.Fatalf("Cache-Control = %q", cc)
	}
	if cc := galleryCacheControl(now.Add(-time.Second), now); !strings.HasPrefix(cc, "public, max-age=1,") {
		t.Fatalf("past expiry Cache-Control = %q", cc)
	}
}

func TestStale304CarriesShortCache(t *testing.T) {
	plugin := &fakePlugin{title: sampleTitle()}
	svc, store := openRouterService(t, plugin)
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)
	if _, err := svc.Fetch(context.Background(), "tt0903747"); err != nil {
		t.Fatal(err)
	}
	ageTitle(t, store, "tt0903747", 31*day, 31*day)
	plugin.err = errors.New("down")

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
	req := httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil)
	req.Header.Set("If-None-Match", rec.Header().Get("ETag"))
	rec2 := serve(mux, req)
	svc.bg.Wait()
	if rec2.Code != http.StatusNotModified || rec2.Header().Get("Cache-Control") != cacheShort {
		t.Fatalf("status = %d, Cache-Control = %q", rec2.Code, rec2.Header().Get("Cache-Control"))
	}
}

func TestETagChangesWithBody(t *testing.T) {
	svc, store := openRouterService(t, &fakePlugin{title: sampleTitle()})
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)
	get := func() string {
		rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747", nil))
		return rec.Header().Get("ETag")
	}
	first := get()
	if first == "" || get() != first {
		t.Fatalf("ETag not stable: %q", first)
	}
	if _, err := store.Exec(`UPDATE title SET rating = 95 WHERE id = 'tt0959621'`); err != nil {
		t.Fatal(err)
	}
	if get() == first {
		t.Fatal("ETag did not change with an episode rating")
	}
}

func TestImagesCacheControlNoSWR(t *testing.T) {
	svc, store := openRouterService(t, &fakePlugin{})
	mux := http.NewServeMux()
	AddEndpoints(mux, svc)
	storeCompleteGallery(t, store, "tt0903747", 3, day)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images", nil))
	cc := rec.Header().Get("Cache-Control")
	if n := maxAge(t, cc); n < 29*86400-5 || n > 29*86400 {
		t.Fatalf("max-age = %d", n)
	}
	if !strings.Contains(cc, "stale-if-error") || strings.Contains(cc, "stale-while-revalidate") {
		t.Fatalf("Cache-Control = %q", cc)
	}
	etag := rec.Header().Get("ETag")
	req := httptest.NewRequest(http.MethodGet, "/v0/meta/imdb/titles/tt0903747/images", nil)
	req.Header.Set("If-None-Match", etag)
	if rec := serve(mux, req); etag == "" || rec.Code != http.StatusNotModified || rec.Header().Get("Last-Modified") == "" {
		t.Fatalf("etag %q: status = %d", etag, rec.Code)
	}
}
