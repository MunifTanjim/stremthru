package tvdb

import (
	"errors"
	"maps"
	"strconv"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/alitto/pond/v2"
	"golang.org/x/sync/singleflight"
)

var listCache = cache.NewCache[TVDBList](&cache.CacheConfig{
	Lifetime:      6 * time.Hour,
	Name:          "tvdb:list",
	LocalCapacity: 1024,
})

var listIdBySlugCache = cache.NewCache[string](&cache.CacheConfig{
	Lifetime:      12 * time.Hour,
	Name:          "tvdb:list-id-by-slug",
	LocalCapacity: 2048,
})

func getListCacheKey(l *TVDBList) string {
	return l.Id
}

var syncItemPool = pond.NewPool(10)
var syncItemGroup singleflight.Group

func syncItem(client *APIClient, li *TVDBItem, saveToDB bool) error {
	if li.Id == 0 {
		return errors.New("id must be provided")
	}

	key := string(li.Type) + strconv.Itoa(li.Id)

	data, err, _ := syncItemGroup.Do(key, func() (any, error) {
		switch li.Type {
		case TVDBItemTypeMovie:
			log.Debug("fetching movie by id", "id", li.Id)
			res, err := client.FetchMovie(&FetchMovieParams{
				Id: li.Id,
			})
			if err != nil {
				return nil, err
			}
			li.Name = res.Data.Name
			li.Overview = res.Data.Translations.GetOverview()
			li.Year = util.SafeParseInt(res.Data.Year, 0)
			li.Runtime = res.Data.Runtime
			li.Poster = res.Data.GetPoster()
			li.Background = res.Data.GetBackground()
			li.Trailer = res.Data.GetTrailer()
			for i := range res.Data.Genres {
				li.Genres = append(li.Genres, res.Data.Genres[i].Id)
			}
			li.IdMap = res.Data.GetIdMap()
		case TVDBItemTypeSeries:
			log.Debug("fetching series by id", "id", li.Id)
			res, err := client.FetchSeries(&FetchSeriesParams{
				Id: li.Id,
			})
			if err != nil {
				return nil, err
			}
			li.Name = res.Data.Name
			li.Overview = res.Data.Translations.GetOverview()
			li.Year = util.SafeParseInt(res.Data.Year, 0)
			li.Runtime = res.Data.AverageRuntime
			li.Poster = res.Data.GetPoster()
			li.Background = res.Data.GetBackground()
			li.Trailer = res.Data.GetTrailer()
			for i := range res.Data.Genres {
				li.Genres = append(li.Genres, res.Data.Genres[i].Id)
			}
			li.IdMap = res.Data.GetIdMap()
		}

		li.UpdatedAt = db.Timestamp{Time: time.Now()}

		item := *li
		if saveToDB {
			if err := UpsertItems(db.GetDB(), []TVDBItem{item}); err != nil {
				return nil, err
			}
		}
		return item, nil
	})
	if err != nil {
		return err
	}

	*li = data.(TVDBItem)
	return nil
}

func (li *TVDBItem) Fetch(client *APIClient) error {
	if li.Id == 0 {
		return errors.New("id must be provided")
	}

	if li.Type == "" {
		return errors.New("type must be provided")
	}

	isMissing := false

	if item, err := GetItemById(li.Type, li.Id); err != nil {
		return err
	} else if item == nil {
		isMissing = true
	} else {
		*li = *item
		log.Debug("found item by id", "type", li.Type, "id", li.Id, "is_stale", li.IsStale())
	}

	if !isMissing {
		if li.IsStale() {
			staleItem := *li
			go func() {
				if err := syncItem(client, &staleItem, true); err != nil {
					log.Error("failed to sync stale item", "error", err, "type", li.Type, "id", li.Id)
				}
			}()
		}
		return nil
	}

	if err := syncItem(client, li, true); err != nil {
		return err
	}

	return nil
}

var syncListMutex sync.Mutex

func syncList(l *TVDBList) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	client := GetAPIClient()

	if l.Id == "" {
		if l.Slug == "" {
			return errors.New("id or slug must be provided")
		}
		res, err := client.FetchList(&FetchListParams{
			Slug: l.Slug,
		})
		if err != nil {
			return err
		}
		l.Id = strconv.Itoa(res.Data.Id)
	}

	log.Debug("fetching list by id", "id", l.Id)
	listId := util.SafeParseInt(l.Id, -1)
	res, err := client.FetchExtendedList(&FetchExtendedListParams{
		Id: listId,
	})
	if err != nil {
		return err
	}
	list := &res.Data

	l.Name = list.Name
	l.Slug = list.URL
	l.Overview = list.Overview
	l.IsOfficial = list.IsOfficial

	movieById := map[int]*TVDBItem{}
	seriesById := map[int]*TVDBItem{}
	for i := range l.Items {
		item := &l.Items[i]
		if !item.HasBasicMeta() {
			continue
		}
		switch item.Type {
		case TVDBItemTypeMovie:
			movieById[item.Id] = item
		case TVDBItemTypeSeries:
			seriesById[item.Id] = item
		}
	}

	l.Items = nil

	log.Debug("fetching list items", "id", l.Id)

	items := make([]TVDBItem, 0, len(list.Entities))

	newMovieIds := []int{}
	newSeriesIds := []int{}
	for i := range list.Entities {
		e := &list.Entities[i]

		lItem := TVDBItem{
			Order: e.Order,
		}

		if e.MovieId != 0 {
			lItem.Id = e.MovieId
			lItem.Type = TVDBItemTypeMovie
			if _, ok := movieById[e.MovieId]; !ok {
				newMovieIds = append(newMovieIds, e.MovieId)
			}
		} else if e.SeriesId != 0 {
			lItem.Id = e.SeriesId
			lItem.Type = TVDBItemTypeSeries
			if _, ok := seriesById[e.SeriesId]; !ok {
				newSeriesIds = append(newSeriesIds, e.SeriesId)
			}
		} else {
			continue
		}

		items = append(items, lItem)
	}

	if byId, err := GetItemsById(TVDBItemTypeMovie, newMovieIds...); err != nil {
		return err
	} else {
		maps.Copy(movieById, byId)
	}
	if byId, err := GetItemsById(TVDBItemTypeSeries, newSeriesIds...); err != nil {
		return err
	} else {
		maps.Copy(seriesById, byId)
	}

	wg := syncItemPool.NewGroup()
	for i := range items {
		item := &items[i]
		order := item.Order
		switch item.Type {
		case TVDBItemTypeMovie:
			if movie, ok := movieById[item.Id]; ok {
				*item = *movie
			}
		case TVDBItemTypeSeries:
			if series, ok := seriesById[item.Id]; ok {
				*item = *series
			}
		}
		item.Order = order
		if item.IsStale() || !item.HasBasicMeta() {
			wg.SubmitErr(func() error {
				return syncItem(client, item, false)
			})
		}
	}
	if err := wg.Wait(); err != nil {
		return err
	}

	l.Items = items

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l), *l); err != nil {
		return err
	}

	return nil
}

func (l *TVDBList) Fetch() error {
	isMissing := false

	if l.Id == "" {
		if l.Slug == "" {
			return errors.New("either id, or slug must be provided")
		}
		listIdBySlugCacheKey := l.Slug
		if !listIdBySlugCache.Get(listIdBySlugCacheKey, &l.Id) {
			if listId, err := GetListIdBySlug(l.Slug); err != nil {
				return err
			} else if listId == "" {
				isMissing = true
			} else {
				l.Id = listId
				log.Debug("found list id by slug", "id", l.Id, "slug", l.Slug)
				listIdBySlugCache.Add(listIdBySlugCacheKey, l.Id)
			}
		}
	}

	listCacheKey := getListCacheKey(l)
	if !isMissing {
		var cachedL TVDBList
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

	if !isMissing {
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

	if err := syncList(l); err != nil {
		return err
	}

	return nil
}
