package letterboxd

import (
	"errors"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
)

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

var syncListMutex sync.Mutex

var client = NewAPIClient(&APIClientConfig{
	apiKey: config.Integration.Letterboxd.APIKey,
	secret: config.Integration.Letterboxd.Secret,
})

func syncList(l *LetterboxdList) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

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
			return err
		}
		l.Id = listId
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
	l.Private = list.SharePolicy != SharePolicyAnyone
	l.Items = nil

	log.Debug("fetching list items", "id", l.Id)
	hasMore := true
	perPage := 100
	cursor := ""
	for hasMore {
		res, err := client.FetchListEntries(&FetchListEntriesParams{
			Id:      l.Id,
			Cursor:  cursor,
			PerPage: perPage,
		})
		if err != nil {
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
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l), *l); err != nil {
		return err
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

	if l.IsStale() {
		staleList := *l
		go func() {
			if err := syncList(&staleList); err != nil {
				log.Error("failed to sync stale list", "id", l.Id, "error", err)
			}
		}()
	}

	return nil
}
