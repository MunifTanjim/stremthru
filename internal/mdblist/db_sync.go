package mdblist

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

var mdblistClient = NewAPIClient(&APIClientConfig{})

func getListCacheKey(l *MDBListList) string {
	return l.Id
}

var syncListMutex sync.Mutex

func syncList(l *MDBListList, apiKey string) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	isWatchlist := l.IsWatchlist()

	var list *List
	if isWatchlist {
		userName := strings.TrimPrefix(l.Id, ID_PREFIX_USER_WATCHLIST)
		list = &List{
			UserName: userName,
			Name:     "Watchlist",
			Slug:     "watchlist/" + userName,
			Updated:  time.Now(),
		}
	} else if l.Id != "" {
		log.Debug("fetching list by id", "id", l.Id)
		id, err := strconv.Atoi(l.Id)
		if err != nil {
			return err
		}
		params := &FetchListByIdParams{
			ListId: id,
		}
		params.APIKey = apiKey
		res, err := mdblistClient.FetchListById(params)
		if err != nil {
			return err
		}
		list = &res.Data
	} else if l.UserName != "" && l.Slug != "" {
		log.Debug("fetching list by name", "name", l.UserName+"/"+l.Slug)
		params := &FetchListByNameParams{
			UserName: l.UserName,
			Slug:     l.Slug,
		}
		params.APIKey = apiKey
		res, err := mdblistClient.FetchListByName(params)
		if err != nil {
			return err
		}
		list = &res.Data
	}

	if list == nil {
		return errors.New("list not found")
	}

	if list.Private && config.IsPublicInstance {
		return errors.New("private list not supported on public instance")
	}

	if l.Id == "" {
		l.Id = strconv.Itoa(list.Id)
	}
	l.UserId = list.UserId
	l.UserName = list.UserName
	l.Name = list.Name
	l.Slug = list.Slug
	l.Description = list.Description
	l.Mediatype = list.Mediatype
	l.Dynamic = list.Dynamic
	l.Private = list.Private
	l.Likes = list.Likes
	l.Items = nil

	log.Debug("fetching list items", "id", l.Id)
	hasMore := true
	limit := 500
	offset := 0
	if isWatchlist {
		for hasMore {
			params := &FetchWatchlistItemsParams{
				Limit:  limit,
				Offset: offset,
			}
			params.APIKey = apiKey
			res, err := mdblistClient.FetchWatchlistItems(params)
			if err != nil {
				return err
			}
			for i := range res.Data {
				item := &res.Data[i]
				l.Items = append(l.Items, MDBListItem{
					IMDBId:         item.ImdbId,
					Adult:          item.Adult == 1,
					Title:          item.Title,
					Poster:         item.Poster,
					Language:       item.Language,
					Mediatype:      item.Mediatype,
					ReleaseYear:    item.ReleaseYear,
					SpokenLanguage: item.SpokenLanguage,
					Genre:          item.Genre,

					Rank:   i,
					TmdbId: strconv.Itoa(item.Id),
					TvdbId: strconv.Itoa(item.TvdbId),
				})
			}
			hasMore = len(res.Data) == limit
			offset += limit
		}
	} else {
		listId := list.Id
		for hasMore {
			params := &FetchListItemsParams{
				ListId: listId,
				Limit:  limit,
				Offset: offset,
			}
			params.APIKey = apiKey
			res, err := mdblistClient.FetchListItems(params)
			if err != nil {
				return err
			}
			for i := range res.Data {
				item := &res.Data[i]
				rank := item.Rank
				if rank == 0 {
					rank = i
				}
				l.Items = append(l.Items, MDBListItem{
					IMDBId:         item.ImdbId,
					Adult:          item.Adult == 1,
					Title:          item.Title,
					Poster:         item.Poster,
					Language:       item.Language,
					Mediatype:      item.Mediatype,
					ReleaseYear:    item.ReleaseYear,
					SpokenLanguage: item.SpokenLanguage,
					Genre:          item.Genre,

					Rank:   rank,
					TmdbId: strconv.Itoa(item.Id),
					TvdbId: strconv.Itoa(item.TvdbId),
				})
			}
			hasMore = len(res.Data) == limit
			offset += limit
		}
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l), *l); err != nil {
		return err
	}

	return nil
}

func (l *MDBListList) Fetch(apiKey string) error {
	isMissing := false
	if l.Id == "" {
		if l.UserName == "" || l.Slug == "" {
			return errors.New("either id, or username and slug must be provided")
		}
		listIdByNameCacheKey := l.UserName + "/" + l.Slug
		if !listIdByNameCache.Get(listIdByNameCacheKey, &l.Id) {
			if listId, err := GetListIdByName(l.UserName, l.Slug); err != nil {
				return err
			} else if listId == "" {
				isMissing = true
			} else {
				l.Id = listId
				log.Debug("found list id by name", "id", l.Id, "name", l.UserName+"/"+l.Slug)
				listIdByNameCache.Add(listIdByNameCacheKey, l.Id)
			}
		}
	}

	if !isMissing {
		listCacheKey := getListCacheKey(l)
		var cachedL MDBListList
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
				if err := syncList(&staleList, apiKey); err != nil {
					log.Error("failed to sync stale list", "id", l.Id, "error", err)
				}
			}()
		}
		return nil
	}

	if err := syncList(l, apiKey); err != nil {
		return err
	}

	return nil
}
