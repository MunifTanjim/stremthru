package imdb

import (
	"context"
	"errors"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
)

// galleryPageSize is the number of images per gallery page.
const galleryPageSize = 500

// maxGalleryPage caps the requested page so the offset cannot overflow; any
// page this far is past the end of every gallery.
const maxGalleryPage = 1 << 20

// GalleryPage is one page of a title's cached image gallery. Total is the
// number of images stored so far; it is the full count only when Complete.
type GalleryPage struct {
	Images   []db.Image `json:"images"`
	Total    int        `json:"total"`
	Pages    int        `json:"pages"`
	Complete bool       `json:"complete"`
}

// GalleryResult is a gallery page with what the caller needs for caching.
type GalleryResult struct {
	Page      *GalleryPage
	Stale     bool      // stale or still loading: short cache
	ExpiresAt time.Time // complete and fresh: uat + galleryFresh
	UpdatedAt time.Time // gallery uat, zero while loading
}

// gallerySync is a running gallery sync. first closes once the first page
// (galleryPageSize images) is stored, or the sync ended (with err set on
// failure), so a request can be answered while the remaining pages load in
// the background.
type gallerySync struct {
	first chan struct{}
	err   error
}

// Images returns page (from 1) of the title's gallery. The first request for
// a title starts a background sync that fetches every plugin page into the
// database and waits only for the first gallery page; later requests are
// served from the database. A complete gallery is fresh for galleryFresh, and
// every page of it expires at the same time. Errors are ErrInvalidID,
// ErrNotFound, ErrUpstream.
func (s *Service) Images(ctx context.Context, id string, page int) (*GalleryResult, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrInvalidID
	}
	page = min(max(page, 1), maxGalleryPage)

	state, err := s.store.GetGallery(id)
	if err != nil && !db.IsNotFound(err) {
		return nil, err
	}
	res := &GalleryResult{}
	var syncErr error
	if state == nil {
		if missing, err := s.store.IsMissingFresh(id, missingTombstone); err != nil {
			return nil, err
		} else if missing {
			return nil, ErrNotFound
		}
		gs := s.startGallerySync(id, false)
		select {
		case <-gs.first:
		case <-ctx.Done():
			return nil, ErrUpstream
		}
		syncErr = gs.err
		state, err = s.store.GetGallery(id)
		if err != nil && !db.IsNotFound(err) {
			return nil, err
		}
	} else if time.Since(state.UpdatedAt) >= galleryFresh {
		s.startGallerySync(id, true)
		res.Stale = true
	}
	if state == nil {
		res.Stale = true
	} else {
		res.UpdatedAt = state.UpdatedAt
		res.ExpiresAt = state.UpdatedAt.Add(galleryFresh)
	}

	images, total, err := s.store.ListGalleryImages(id, galleryPageSize, (page-1)*galleryPageSize)
	if err != nil {
		return nil, err
	}
	// A failed sync still serves the images stored so far.
	if syncErr != nil && total == 0 {
		return nil, syncErr
	}
	res.Page = &GalleryPage{
		Images:   images,
		Total:    total,
		Pages:    (total + galleryPageSize - 1) / galleryPageSize,
		Complete: state != nil,
	}
	return res, nil
}

// startGallerySync starts a background sync for id unless one is running or
// one failed recently, and returns the sync to wait on. A resync of a
// complete gallery never blocks callers.
func (s *Service) startGallerySync(id string, resync bool) *gallerySync {
	s.galleryMu.Lock()
	defer s.galleryMu.Unlock()
	if gs, ok := s.galleries[id]; ok {
		return gs
	}
	done := func(err error) *gallerySync {
		gs := &gallerySync{first: make(chan struct{}), err: err}
		close(gs.first)
		return gs
	}
	// A sync can finish between the caller's GetGallery and this lock; it
	// completes the gallery before leaving the map, so check again: a first
	// sync is not needed once the gallery is complete, a resync not once it
	// is fresh.
	if state, err := s.store.GetGallery(id); err == nil && (!resync || time.Since(state.UpdatedAt) < galleryFresh) {
		return done(nil)
	}
	if s.holds.blocked("gallery:"+id, time.Now()) {
		return done(ErrUpstream)
	}
	gs := &gallerySync{first: make(chan struct{})}
	s.galleries[id] = gs
	s.bg.Add(1)
	if _, ok := s.galleryPool.TrySubmit(func() {
		defer s.bg.Done()
		s.runGallerySync(id, gs, resync)
	}); !ok {
		// The queue is full: drop a resync (the stored gallery keeps being
		// served), fail a first sync.
		delete(s.galleries, id)
		s.bg.Done()
		if resync {
			return done(nil)
		}
		return done(ErrUpstream)
	}
	return gs
}

// runGallerySync stores every plugin page, then removes the images the
// plugin no longer lists and marks the gallery complete. Images are upserted,
// so a sync that died part way, or a resync of a complete gallery, can simply
// run again.
func (s *Service) runGallerySync(id string, gs *gallerySync, resync bool) {
	ctx, cancel := context.WithTimeout(context.Background(), galleryTimeout)
	defer cancel()

	firstClosed := false
	if resync {
		close(gs.first)
		firstClosed = true
	}
	finish := func(err error) {
		s.galleryMu.Lock()
		delete(s.galleries, id)
		s.galleryMu.Unlock()
		switch {
		case err == nil:
			s.holds.clear("gallery:" + id)
		case !errors.Is(err, ErrNotFound):
			s.holds.set("gallery:"+id, galleryRetry, time.Now())
		}
		if !firstClosed {
			gs.err = err
			close(gs.first)
		}
	}

	var seen []string
	cursor := ""
	read := false
	for page := 0; page < maxGalleryPages; page++ {
		p, err := s.plugin.GetImages(ctx, id, cursor)
		if err != nil {
			if errors.Is(err, imdbplugin.ErrNotFound) {
				err := s.removeTitle(id)
				finish(err)
			} else {
				finish(ErrUpstream)
			}
			return
		}
		if p == nil {
			finish(ErrUpstream)
			return
		}
		if err := s.store.AddGalleryImages(id, p.Images); err != nil {
			finish(err)
			return
		}
		for _, img := range p.Images {
			seen = append(seen, img.ID)
		}
		if !firstClosed && len(seen) >= galleryPageSize {
			firstClosed = true
			close(gs.first)
		}
		if p.Next == "" || p.Next == cursor {
			read = true
			break
		}
		cursor = p.Next
	}
	if read {
		if err := s.store.PruneGallery(id, seen); err != nil {
			finish(err)
			return
		}
	} else {
		// Pruning needs every page; keep what is stored and serve it as is.
		log.Warn("gallery truncated at page limit", "id", id, "pages", maxGalleryPages)
	}
	finish(s.store.CompleteGallery(id))
}
