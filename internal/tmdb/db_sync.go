package tmdb

import (
	"errors"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
)

var listCache = cache.NewCache[TMDBList](&cache.CacheConfig{
	Lifetime:      6 * time.Hour,
	Name:          "tmdb:list",
	LocalCapacity: 1024,
})

func getListCacheKey(l *TMDBList, tokenId string) string {
	if l.IsUserSpecific() {
		return tokenId + ":" + l.Id
	}
	return l.Id
}

func syncList(l *TMDBList, tokenId string) error {
	client := GetAPIClient(tokenId)

	var list *List
	if l.IsDynamic() {
		log.Debug("fetching dynamic list by id", "id", l.Id)
		meta := GetDynamicListMeta(l.Id)
		if meta == nil {
			return errors.New("invalid id")
		}
		var err error
		list, err = meta.Fetch(client)
		if err != nil {
			return err
		}
	} else if l.Id != "" {
		log.Debug("fetching list by id", "id", l.Id)
		listId, _ := strconv.Atoi(l.Id)
		page := 0
		for {
			page++
			log.Debug("fetching list page", "id", l.Id, "page", page)
			res, err := client.FetchList(&FetchListParams{
				ListId: listId,
				Page:   page,
			})
			if err != nil {
				return err
			}
			if page == 1 {
				list = &res.Data
			} else {
				list.Page = res.Data.Page
				list.Results = append(list.Results, res.Data.Results...)
			}
			if list.Page == list.TotalPages {
				break
			}
		}
	} else {
		return errors.New("id must be provided")
	}

	if list == nil {
		return errors.New("list not found")
	}

	l.Name = list.Name
	l.Description = list.Description
	l.Private = !list.Public
	if list.CreatedBy.Id != "" {
		l.AccountId = list.CreatedBy.Id
	}
	if list.CreatedBy.Username != "" {
		l.Username = list.CreatedBy.Username
	}
	l.Items = make([]TMDBItem, 0, len(list.Results))

	now := time.Now()
	for i := range list.Results {
		r := &list.Results[i]
		if r.MediaType == "" || r.data == nil {
			continue
		}

		d := TMDBItem{
			Type:      r.MediaType,
			IsPartial: true,
			UpdatedAt: db.Timestamp{Time: now},
			Idx:       i,
		}
		switch r.MediaType {
		case MediaTypeMovie:
			data := r.Movie()
			d.Id = data.Id
			d.Title = data.Title
			d.OriginalTitle = data.OriginalTitle
			d.Overview = data.Overview
			d.ReleaseDate = db.DateOnly{Time: data.GetReleaseDate()}
			d.IsAdult = data.Adult
			d.Backdrop = data.BackdropPath
			d.Poster = data.PosterPath
			d.Popularity = data.Popularity
			d.VoteAverage = data.VoteAverage
			d.VoteCount = data.VoteCount
			d.Genres = data.GenreIds
		case MediaTypeTVShow:
			data := r.Show()
			d.Id = data.Id
			d.Title = data.Name
			d.OriginalTitle = data.OriginalName
			d.Overview = data.Overview
			d.ReleaseDate = db.DateOnly{Time: data.GetFirstAirDate()}
			d.IsAdult = data.Adult
			d.Backdrop = data.BackdropPath
			d.Poster = data.PosterPath
			d.Popularity = data.Popularity
			d.VoteAverage = data.VoteAverage
			d.VoteCount = data.VoteCount
			d.Genres = data.GenreIds
		}
		l.Items = append(l.Items, d)
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l, tokenId), *l); err != nil {
		return err
	}

	return nil
}

func (l *TMDBList) Fetch(tokenId string) error {
	if l.Id == "" {
		return errors.New("id must be provided")
	}

	isMissing := false

	listCacheKey := getListCacheKey(l, tokenId)
	var cachedL TMDBList
	if !listCache.Get(listCacheKey, &cachedL) {
		if !l.IsUserSpecific() {
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
			isMissing = true
		}
	} else {
		*l = cachedL
	}

	if !isMissing {
		if l.IsStale() {
			staleList := *l
			go func() {
				if err := syncList(&staleList, tokenId); err != nil {
					log.Error("failed to sync stale list", "id", l.Id, "error", err)
				}
			}()
		}
		return nil
	}

	if err := syncList(l, tokenId); err != nil {
		return err
	}

	return nil
}
