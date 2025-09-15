package letterboxd

import (
	"errors"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

const MAX_LIST_ITEM_COUNT = 5000

var LetterboxdEnabled = config.Integration.Letterboxd.IsEnabled()
var LetterboxdPiggybacked = config.Integration.Letterboxd.IsPiggybacked()

var Peer = peer.NewAPIClient(&peer.APIClientConfig{
	BaseURL: config.PeerURL,
})

var listCache = cache.NewCache[LetterboxdList](&cache.CacheConfig{
	Lifetime:      6 * time.Hour,
	Name:          "letterboxd:list",
	LocalCapacity: 1024,
})

var listIdBySlugCache = cache.NewCache[string](&cache.CacheConfig{
	Lifetime:      12 * time.Hour,
	Name:          "letterboxd:list-id-by-slug",
	LocalCapacity: 2048,
})

func getListCacheKey(l *LetterboxdList) string {
	return l.Id
}

func InvalidateListCache(list *LetterboxdList) {
	listCache.Remove(getListCacheKey(list))
}

var syncListMutex sync.Mutex

func syncList(l *LetterboxdList) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	client := GetSystemClient()

	var list *List

	if l.Id == "" {
		if l.UserName == "" || l.Slug == "" {
			return errors.New("either id, or user_id and slug must be provided")
		}

		log.Debug("fetching list id by slug", "slug", l.UserName+"/"+l.Slug)
		listId, err := client.FetchListID(&FetchListIDParams{
			ListURL: SITE_BASE_URL + "/" + l.UserName + "/list/" + l.Slug + "/",
		})
		if err != nil {
			log.Error("failed to fetch list id by slug", "error", err, "slug", l.UserName+"/"+l.Slug)
			return err
		}
		l.Id = listId
	}

	if !LetterboxdEnabled {
		if !LetterboxdPiggybacked {
			return errors.New("letterboxd integration is not available")
		}

		log.Debug("fetching list by id from upstream", "id", l.Id)
		res, err := Peer.FetchLetterboxdList(&peer.FetchLetterboxdListParams{
			ListId: l.Id,
		})
		if err != nil {
			return err
		}

		list := &res.Data

		l.UserId = list.UserId
		l.UserName = list.UserSlug
		l.Name = list.Title
		l.Slug = list.Slug
		l.Description = list.Description
		l.Private = list.IsPrivate
		l.ItemCount = list.ItemCount
		l.Items = nil
		for i := range list.Items {
			item := &list.Items[i]
			l.Items = append(l.Items, LetterboxdItem{
				Id:          item.Id,
				Name:        item.Title,
				ReleaseYear: item.Year,
				Runtime:     item.Runtime,
				Rating:      item.Rating,
				Adult:       item.IsAdult,
				Poster:      item.Poster,
				UpdatedAt:   db.Timestamp{Time: item.UpdatedAt},

				GenreIds: item.GenreIds,
				IdMap:    &item.IdMap,
				Rank:     item.Index,
			})
		}

		if err := UpsertList(l); err != nil {
			return err
		}

		if l.HasUnfetchedItems() {
			if err := listCache.AddWithLifetime(getListCacheKey(l), *l, l.StaleIn()); err != nil {
				return err
			}
		} else {
			if err := listCache.Add(getListCacheKey(l), *l); err != nil {
				return err
			}
		}

		return nil
	}

	log.Debug("fetching list by id", "id", l.Id)
	res, err := client.FetchList(&FetchListParams{
		Id: l.Id,
	})
	if err != nil {
		return err
	}
	list = &res.Data

	l.UserId = list.Owner.Id
	l.UserName = list.Owner.Username
	l.Name = list.Name
	if slug := list.getLetterboxdSlug(); slug != "" {
		l.Slug = slug
	}
	l.Description = list.Description
	l.Private = false // list.SharePolicy != SharePolicyAnyone
	l.ItemCount = list.FilmCount
	l.Items = nil

	hasMore := true
	perPage := 100
	page := 0
	cursor := ""
	max_page := 2
	for hasMore && page < max_page {
		page++
		log.Debug("fetching list items", "id", l.Id, "page", page)
		res, err := client.FetchListEntries(&FetchListEntriesParams{
			Id:      l.Id,
			Cursor:  cursor,
			PerPage: perPage,
		})
		if err != nil {
			log.Error("failed to fetch list items", "id", l.Id, "error", err)
			return err
		}
		now := time.Now()
		for i := range res.Data.Items {
			item := &res.Data.Items[i]
			rank := item.Rank
			if rank == 0 {
				rank = i
			}
			l.Items = append(l.Items, LetterboxdItem{
				Id:          item.Film.Id,
				Name:        item.Film.Name,
				ReleaseYear: item.Film.ReleaseYear,
				Runtime:     item.Film.RunTime,
				Rating:      int(item.Film.Rating * 2 * 10),
				Adult:       item.Film.Adult,
				Poster:      item.Film.GetPoster(),
				UpdatedAt:   db.Timestamp{Time: now},

				GenreIds: item.Film.GenreIds(),
				IdMap:    item.Film.GetIdMap(),
				Rank:     rank,
			})
		}
		cursor = res.Data.Next
		hasMore = cursor != "" && len(res.Data.Items) == perPage
		time.Sleep(2 * time.Second)
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l), *l); err != nil {
		return err
	}

	if l.HasUnfetchedItems() {
		log.Info("list is not fully synced, queuing for sync", "id", l.Id, "item_count", l.ItemCount, "fetched_item_count", len(l.Items))
		worker_queue.LetterboxdListSyncerQueue.Queue(worker_queue.LetterboxdListSyncerQueueItem{
			ListId: l.Id,
		})
	}

	return nil
}

func (l *LetterboxdList) Fetch() error {
	isMissing := false

	if l.Id == "" {
		if l.UserName == "" || l.Slug == "" {
			return errors.New("either id, or user_name and slug must be provided")
		}
		listIdBySlugCacheKey := l.UserName + "/" + l.Slug
		if !listIdBySlugCache.Get(listIdBySlugCacheKey, &l.Id) {
			if listId, err := GetListIdBySlug(l.UserName, l.Slug); err != nil {
				return err
			} else if listId == "" {
				isMissing = true
			} else {
				l.Id = listId
				log.Debug("found list id by slug", "id", l.Id, "slug", l.UserName+"/"+l.Slug)
				listIdBySlugCache.Add(listIdBySlugCacheKey, l.Id)
			}
		}
	}

	listCacheKey := getListCacheKey(l)
	if !isMissing {
		var cachedL LetterboxdList
		if !listCache.Get(listCacheKey, &cachedL) {
			if list, err := GetListById(l.Id); err != nil {
				return err
			} else if list == nil {
				isMissing = true
			} else {
				*l = *list
				log.Debug("found list by id", "id", l.Id, "is_stale", l.IsStale())
				listCache.Add(listCacheKey, *l)
			}
		} else {
			*l = cachedL
		}
	}

	if isMissing {
		return syncList(l)
	}

	if l.IsStale() || l.HasUnfetchedItems() {
		log.Info("queueing list for sync", "id", l.Id, "item_count", l.ItemCount, "fetched_item_count", len(l.Items))
		worker_queue.LetterboxdListSyncerQueue.Queue(worker_queue.LetterboxdListSyncerQueueItem{
			ListId: l.Id,
		})
	}

	return nil
}
