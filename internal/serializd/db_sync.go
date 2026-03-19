package serializd

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
)

var client = NewAPIClient(&APIClientConfig{})

var listCache = cache.NewCache[SerializdList](&cache.CacheConfig{
	Lifetime: 1 * time.Hour,
	Name:     "serializd:list",
	MaxSize:  16,
})

const max_items = 500

var syncListMutex sync.Mutex

func syncList(l *SerializdList) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	name, ok := listNames[l.Id]
	if !ok {
		return errors.New("invalid list id: " + l.Id)
	}
	l.Name = name

	log.Debug("fetching list", "id", l.Id)

	var items []SerializdItem
	idx := 0

	for page := 1; ; page++ {
		log.Debug("fetching list items", "id", l.Id, "page", page)
		var totalPages int
		switch strings.TrimPrefix(l.Id, ID_PREFIX_DYNAMIC) {
		case "popular":
			res, err := client.ListPopularShows(&ListPopularShowsParams{Page: page})
			if err != nil {
				return err
			}
			totalPages = res.Data.TotalPages
			for i := range res.Data.Results {
				s := &res.Data.Results[i]
				items = append(items, SerializdItem{
					ID:          s.ID,
					Name:        s.Name,
					BannerImage: s.BannerImage,
					Idx:         idx,
				})
				idx++
			}

		case "trending":
			res, err := client.ListTrendingShows(&ListTrendingShowsParams{Page: page})
			if err != nil {
				return err
			}
			totalPages = res.Data.TotalPages
			for i := range res.Data.Results {
				s := &res.Data.Results[i]
				items = append(items, SerializdItem{
					ID:          s.ID,
					Name:        s.Name,
					BannerImage: s.BannerImage,
					Idx:         idx,
				})
				idx++
			}

		case "featured":
			res, err := client.ListFeaturedShows(&ListFeaturedShowsParams{Page: page})
			if err != nil {
				return err
			}
			totalPages = res.Data.TotalPages
			for i := range res.Data.Results {
				s := &res.Data.Results[i].ShowDetails
				items = append(items, SerializdItem{
					ID:          s.ID,
					Name:        s.Name,
					BannerImage: s.BannerImage,
					Idx:         idx,
				})
				idx++
			}

		case "anticipated":
			res, err := client.ListAnticipatedShows(&ListAnticipatedShowsParams{Page: page})
			if err != nil {
				return err
			}
			totalPages = res.Data.TotalPages
			for i := range res.Data.Results {
				s := &res.Data.Results[i].ShowDetails
				items = append(items, SerializdItem{
					ID:          s.ID,
					Name:        s.Name,
					BannerImage: s.BannerImage,
					Idx:         idx,
				})
				idx++
			}

		case "top-shows":
			res, err := client.ListTopShows(&ListTopShowsParams{Page: page})
			if err != nil {
				return err
			}
			totalPages = res.Data.TotalPages
			for i := range res.Data.Results {
				s := &res.Data.Results[i].ShowDetails
				items = append(items, SerializdItem{
					ID:          s.ID,
					Name:        s.Name,
					BannerImage: s.BannerImage,
					Idx:         idx,
				})
				idx++
			}
		}

		if page >= totalPages || len(items) >= max_items {
			break
		}
	}

	l.Items = items

	if err := UpsertList(l); err != nil {
		return err
	}

	listCache.Add(l.Id, *l)
	return nil
}

func (l *SerializdList) Fetch() error {
	if l.Id == "" {
		return errors.New("id must be provided")
	}

	if !IsValidListId(l.Id) {
		return errors.New("invalid list id: " + l.Id)
	}

	isMissing := false

	var cachedL SerializdList
	if !listCache.Get(l.Id, &cachedL) {
		if list, err := GetListById(l.Id); err != nil {
			return err
		} else if list == nil {
			isMissing = true
		} else {
			*l = *list
			log.Debug("found list by id", "id", l.Id, "is_stale", l.IsStale())
			listCache.Add(l.Id, *l)
		}
	} else {
		*l = cachedL
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

	return syncList(l)
}
