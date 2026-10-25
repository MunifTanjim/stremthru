package imdb

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	"github.com/MunifTanjim/stremthru/plugin/imdb"
	"github.com/alitto/pond/v2"
	"golang.org/x/sync/singleflight"
)

var (
	ErrInvalidID = errors.New("meta/imdb: invalid id")
	ErrNotFound  = imdbplugin.ErrNotFound
	ErrUpstream  = errors.New("meta/imdb: upstream unavailable")
)

var idPattern = regexp.MustCompile(`^tt\d+$`)

const (
	// missingTombstone is how long a plugin 404 suppresses refetch.
	missingTombstone = 7 * 24 * time.Hour
	// failedRefreshHold is how long a failed background refresh suppresses
	// another attempt; the stored data keeps being served meanwhile.
	failedRefreshHold = 10 * time.Minute
	// failedFetchHold is how long a failed first fetch (nothing stored)
	// suppresses another attempt.
	failedFetchHold = 60 * time.Second
	// maxBackgroundRefreshes caps concurrent background refreshes; others
	// wait in a queue of up to refreshQueueSize. When the queue is full a
	// refresh is dropped, and the next request for that title queues it
	// again, so a burst cannot build an unbounded backlog of plugin calls.
	maxBackgroundRefreshes = 4
	refreshQueueSize       = 1000
	// titleTimeout bounds the plugin's GetTitle call.
	titleTimeout = 30 * time.Second
	// Episodes are enriched in parallel batches. Each attempt at a batch has
	// its own deadline, so a refresh takes longer for a longer series instead
	// of failing once a fixed budget runs out.
	episodeBatchSize   = imdbplugin.MaxEpisodeBatch
	episodeConcurrency = 4
	// The plugin client caps each call at 10s, so a longer attempt timeout
	// would never apply.
	episodeBatchTimeout  = 10 * time.Second
	episodeBatchAttempts = 2

	// galleryFresh is how long a complete gallery is served before a
	// background resync starts (the cached copy keeps being served).
	galleryFresh = 30 * 24 * time.Hour
	// galleryTimeout bounds a whole gallery sync, across all pages.
	galleryTimeout = 2 * time.Minute
	// galleryRetry is how long a failed sync suppresses another attempt.
	galleryRetry    = 60 * time.Second
	maxGalleryPages = 50
	// maxGallerySyncs caps concurrent gallery syncs; others wait in a queue
	// of up to galleryQueueSize. When the queue is full a resync is dropped
	// and a first sync fails with ErrUpstream.
	maxGallerySyncs  = 4
	galleryQueueSize = 1000
)

type Store interface {
	GetFullTitle(id string) (*db.FullTitle, error)
	UpsertFullTitle(ft *db.FullTitle) error
	MarkMissing(id string) error
	IsMissingFresh(id string, maxAge time.Duration) (bool, error)
	GetRefreshState(id string) (*db.RefreshState, error)
	ApplyFastRefresh(fr *db.FastRefresh) error
	ListEpisodeDates(seriesID string) (map[string]string, error)
	GetTitleAndState(id string) (*db.FullTitle, *db.RefreshState, error)
	RemoveTitle(id string) error

	GetGallery(id string) (*db.GalleryState, error)
	ListGalleryImages(id string, limit, offset int) ([]db.Image, int, error)
	AddGalleryImages(id string, images []db.Image) error
	PruneGallery(id string, keep []string) error
	CompleteGallery(id string) error
}

type Provider interface {
	GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error)
	GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error)
	GetEpisodes(ctx context.Context, ids []string) ([]imdbplugin.TitleSummary, error)
	GetEpisodeList(ctx context.Context, id string) ([]imdbplugin.Season, error)
	GetRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error)
}

type Service struct {
	store  Store
	plugin Provider
	group  singleflight.Group

	// bg tracks background refreshes and gallery syncs.
	bg          sync.WaitGroup
	holds       *holdoff
	refreshPool pond.Pool
	runningMu   sync.Mutex
	running     map[string]bool

	galleryMu   sync.Mutex
	galleries   map[string]*gallerySync
	galleryPool pond.Pool
}

func NewService(store Store, plugin Provider) *Service {
	return &Service{
		store:       store,
		plugin:      plugin,
		galleries:   map[string]*gallerySync{},
		galleryPool: pond.NewPool(maxGallerySyncs, pond.WithQueueSize(galleryQueueSize)),
		holds:       newHoldoff(),
		refreshPool: pond.NewPool(maxBackgroundRefreshes, pond.WithQueueSize(refreshQueueSize)),
		running:     map[string]bool{},
	}
}

// enrichEpisodes fills the bare episodes of a series from the plugin in
// parallel batches. A failed batch is retried once; a batch that fails again
// fails the refresh, so a series is never stored with nameless episodes. Ids
// the plugin omits stay bare.
func (s *Service) enrichEpisodes(ctx context.Context, ft *imdbplugin.FullTitle) error {
	var ids []string
	for _, season := range ft.Seasons {
		for _, ep := range season.Episodes {
			if ep.ID != "" {
				ids = append(ids, ep.ID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	summaries, err := s.fetchSummaries(ctx, ids)
	if err != nil {
		return err
	}

	for si := range ft.Seasons {
		for ei := range ft.Seasons[si].Episodes {
			ep := &ft.Seasons[si].Episodes[ei]
			sum, ok := summaries[ep.ID]
			if !ok {
				continue
			}
			if sum.Description == "" {
				sum.Description = ep.Description
			}
			ep.TitleSummary = sum
		}
	}
	return nil
}
