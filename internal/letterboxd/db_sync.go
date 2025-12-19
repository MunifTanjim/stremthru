package letterboxd

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	meta_type "github.com/MunifTanjim/stremthru/internal/meta/type"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/request"
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

func getListCacheKey(l *LetterboxdList) string {
	return l.Id
}

func InvalidateListCache(list *LetterboxdList) {
	listCache.Remove(getListCacheKey(list))
}

var syncListMutex sync.Mutex

func syncList(l *LetterboxdList) error {
	if l.Id == "" {
		return errors.New("id must be provided")
	}

	isUserWatchlist := l.IsUserWatchlist()
	if isUserWatchlist {
		if l.UserId == "" {
			return errors.New("user id must be provided for watchlist")
		}
	}

	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	if !LetterboxdEnabled {
		if !LetterboxdPiggybacked {
			return errors.New("letterboxd integration is not available")
		}

		log.Debug("fetching list by id from upstream", "id", l.Id)
		var res request.APIResponse[meta_type.List]
		var err error
		if isUserWatchlist {
			res, err = Peer.FetchLetterboxdUserWatchlist(&peer.FetchLetterboxdUserWatchlistParams{
				UserId: l.UserId,
			})
		} else {
			res, err = Peer.FetchLetterboxdList(&peer.FetchLetterboxdListParams{
				ListId: l.Id,
			})
		}
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
		l.Version = list.Version
		l.UpdatedAt = db.Timestamp{Time: list.UpdatedAt}
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

		return UpsertList(l)
	}

	client := GetSystemClient()

	if isUserWatchlist {
		l.Name = "Watchlist"
		l.Slug = "watchlist"
		res, err := client.FetchMemberStatistics(&FetchMemberStatisticsParams{
			Id: l.UserId,
		})
		if err != nil {
			return err
		}
		l.ItemCount = res.Data.Counts.Watchlist
		l.Version = time.Now().Unix()
	} else {
		log.Debug("fetching list by id", "id", l.Id)
		res, err := client.FetchList(&FetchListParams{
			Id: l.Id,
		})
		if err != nil {
			return err
		}
		list := &res.Data

		l.UserId = list.Owner.Id
		l.UserName = list.Owner.Username
		l.Name = list.Name
		if slug := list.GetLetterboxdSlug(); slug != "" {
			l.Slug = slug
		}
		l.Description = list.Description
		l.Private = false // list.SharePolicy != SharePolicyAnyone
		l.ItemCount = list.FilmCount
		l.Version = list.Version
	}
	l.Items = nil
	l.UpdatedAt = db.Timestamp{Time: time.Now()}

	hasMore := true
	perPage := 100
	page := 0
	cursor := ""
	max_page := 2
	lastRank := -1
	for hasMore && page < max_page {
		page++
		log.Debug("fetching list items", "id", l.Id, "page", page)
		if isUserWatchlist {
			res, err := client.FetchMemberWatchlist(&FetchMemberWatchlistParams{
				Id:      l.UserId,
				Cursor:  cursor,
				PerPage: perPage,
			})
			if err != nil {
				log.Error("failed to fetch list items", "error", err, "id", l.Id)
				return err
			}
			now := time.Now()
			for i := range res.Data.Items {
				item := &res.Data.Items[i]
				lastRank++
				rank := lastRank
				l.Items = append(l.Items, LetterboxdItem{
					Id:          item.Id,
					Name:        item.Name,
					ReleaseYear: item.ReleaseYear,
					Runtime:     item.RunTime,
					Rating:      int(item.Rating * 2 * 10),
					Adult:       item.Adult,
					Poster:      item.GetPoster(),
					UpdatedAt:   db.Timestamp{Time: now},

					GenreIds: item.GenreIds(),
					IdMap:    item.GetIdMap(),
					Rank:     rank,
				})
			}

			if page == 1 && len(res.Data.Items) > 0 {
				firstItem := &res.Data.Items[0]
				for i := range firstItem.Relationships {
					if member := &firstItem.Relationships[i].Member; member.Id == l.UserId {
						l.UserName = member.Username
						break
					}
				}
			}

			cursor = res.Data.Next
			hasMore = cursor != "" && len(res.Data.Items) == perPage
		} else {
			res, err := client.FetchListEntries(&FetchListEntriesParams{
				Id:      l.Id,
				Cursor:  cursor,
				PerPage: perPage,
			})
			if err != nil {
				log.Error("failed to fetch list items", "error", err, "id", l.Id)
				return err
			}
			now := time.Now()
			for i := range res.Data.Items {
				item := &res.Data.Items[i]
				rank := item.Rank
				if rank == 0 {
					lastRank++
					rank = lastRank
				} else {
					lastRank = rank
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
		}
		time.Sleep(200 * time.Millisecond)
	}

	return UpsertList(l)
}

func (l *LetterboxdList) Fetch() error {
	if l.Id == "" {
		return errors.New("id must be provided")
	}

	if l.IsUserWatchlist() {
		if l.UserId == "" {
			l.UserId = strings.TrimPrefix(l.Id, ID_PREFIX_USER_WATCHLIST)
		}
	}

	isMissing := false

	listCacheKey := getListCacheKey(l)
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

	if isMissing {
		if err := syncList(l); err != nil {
			return err
		}
	}

	if err := listCache.Add(getListCacheKey(l), *l); err != nil {
		return err
	}

	is_stale := l.IsStale()
	if is_stale || l.HasUnfetchedItems() {
		log.Info("queueing list for sync", "id", l.Id, "item_count", l.ItemCount, "fetched_item_count", len(l.Items), "is_stale", is_stale)
		worker_queue.LetterboxdListSyncerQueue.Queue(worker_queue.LetterboxdListSyncerQueueItem{
			ListId: l.Id,
		})
	}

	return nil
}
