package imdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

// cacheShort is for stale data, a gallery still loading, and 404s.
const cacheShort = "public, max-age=60"

// maxAgeUntil is the whole seconds from now until expires, rounded up and at
// least 1.
func maxAgeUntil(expires, now time.Time) int64 {
	return max(1, int64(math.Ceil(expires.Sub(now).Seconds())))
}

// titleCacheControl caches a title until its next refresh is due. A cache
// revalidating an expired title reaches StremThru, which serves the stored
// data and starts the refresh.
func titleCacheControl(expires, now time.Time) string {
	return fmt.Sprintf("public, max-age=%d, stale-while-revalidate=604800, stale-if-error=86400", maxAgeUntil(expires, now))
}

// galleryCacheControl caches a gallery page until the gallery is due for a
// resync, so every page expires at the same time. stale-while-revalidate is
// left out: it would let caches serve each page past that shared expiry.
func galleryCacheControl(expires, now time.Time) string {
	return fmt.Sprintf("public, max-age=%d, stale-if-error=86400", maxAgeUntil(expires, now))
}

// bodyETag is a weak ETag over the JSON encoding of v, so it changes whenever
// anything in the response does.
func bodyETag(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := fnv.New64a()
	h.Write(b)
	return fmt.Sprintf(`W/"%016x"`, h.Sum64()), nil
}

// sendCached sets the ETag and answers 304 when it matches If-None-Match;
// otherwise it sends data. Cache-Control must already be set.
func sendCached(w http.ResponseWriter, r *http.Request, data any) {
	etag, err := bodyETag(data)
	if err != nil {
		shared.ErrorInternalServerError(r, "").WithCause(err).Send(w, r)
		return
	}
	w.Header().Set("ETag", etag)
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	shared.SendResponse(w, r, http.StatusOK, data, nil)
}

func IsEnabled() bool {
	return config.Feature.HasIMDBMeta()
}

func sendError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidID):
		shared.ErrorBadRequest(r, "invalid id").Send(w, r)
	case errors.Is(err, ErrNotFound):
		w.Header().Set("Cache-Control", cacheShort)
		shared.ErrorNotFound(r).Send(w, r)
	case errors.Is(err, ErrUpstream):
		shared.ErrorBadGateway(r, "").WithCause(err).Send(w, r)
	default:
		shared.ErrorInternalServerError(r, "").WithCause(err).Send(w, r)
	}
}

// etagMatches reports whether an If-None-Match header matches etag, using
// weak comparison over a comma-separated list or "*".
func etagMatches(header, etag string) bool {
	want := strings.TrimPrefix(etag, "W/")
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == want {
			return true
		}
	}
	return false
}

func handleTitle(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !shared.IsMethod(r, http.MethodGet) {
			shared.ErrorMethodNotAllowed(r).Send(w, r)
			return
		}

		id := r.PathValue("id")
		res, err := svc.Fetch(r.Context(), id)
		if err != nil {
			sendError(w, r, err)
			return
		}

		if res.Stale {
			// Do not let edge caches pin data that is being refreshed.
			w.Header().Set("Cache-Control", cacheShort)
			w.Header().Set("Warning", `110 - "stale"`)
		} else {
			w.Header().Set("Cache-Control", titleCacheControl(res.ExpiresAt, time.Now()))
		}
		w.Header().Set("Last-Modified", res.LastModified.UTC().Format(http.TimeFormat))
		sendCached(w, r, res.Title)
	}
}

func handleImages(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !shared.IsMethod(r, http.MethodGet) {
			shared.ErrorMethodNotAllowed(r).Send(w, r)
			return
		}

		page := 1
		if v := r.URL.Query().Get("page"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				shared.ErrorBadRequest(r, "invalid page").Send(w, r)
				return
			}
			page = n
		}

		res, err := svc.Images(r.Context(), r.PathValue("id"), page)
		if err != nil {
			sendError(w, r, err)
			return
		}

		// A page of a gallery still loading or due for a resync must not be
		// cached for long.
		if res.Stale {
			w.Header().Set("Cache-Control", cacheShort)
		} else {
			w.Header().Set("Cache-Control", galleryCacheControl(res.ExpiresAt, time.Now()))
		}
		if !res.UpdatedAt.IsZero() {
			w.Header().Set("Last-Modified", res.UpdatedAt.UTC().Format(http.TimeFormat))
		}
		sendCached(w, r, res.Page)
	}
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.WithCtx(r.Context(), "req.id", ctx.RequestId)
		next.ServeHTTP(w, r)
	})
}

// AddEndpoints registers GET /v0/meta/imdb/titles/{id} and
// GET /v0/meta/imdb/titles/{id}/images. The caller gates on IsEnabled; the
// service is injected for testability.
func AddEndpoints(mux *http.ServeMux, svc *Service) {
	router := http.NewServeMux()

	router.HandleFunc("/titles/{id}", handleTitle(svc))
	router.HandleFunc("/titles/{id}/images", handleImages(svc))

	mux.Handle("/v0/meta/imdb/", http.StripPrefix("/v0/meta/imdb", commonMiddleware(router)))
}
