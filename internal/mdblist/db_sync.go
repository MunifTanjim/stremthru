package mdblist

import (
	"errors"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/config"
)

var mdblistClient = NewAPIClient(&APIClientConfig{})

func (l *MDBListList) Fetch(apiKey string) error {
	isMissing := false
	if l.Id == 0 {
		if l.UserName == "" || l.Slug == "" {
			return errors.New("either id, or username and slug must be provided")
		}
		listIdByNameCacheKey := l.UserName + "/" + l.Slug
		if !listIdByNameCache.Get(listIdByNameCacheKey, &l.Id) {
			if listId, err := GetListIdByName(l.UserName, l.Slug); err != nil {
				return err
			} else if listId == 0 {
				isMissing = true
			} else {
				l.Id = listId
				log.Debug("found list id by name", "id", l.Id, "name", l.UserName+"/"+l.Slug)
				listIdByNameCache.Add(listIdByNameCacheKey, l.Id)
			}
		}
	}

	if !isMissing {
		listCacheKey := strconv.Itoa(l.Id)
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

	if !isMissing && !l.IsStale() {
		return nil
	}

	var list *List
	if l.Id != 0 {
		log.Debug("fetching list by id", "id", l.Id)
		params := &FetchListByIdParams{
			ListId: l.Id,
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

	l.Id = list.Id
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
	for hasMore {
		params := &FetchListItemsParams{
			ListId: l.Id,
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

				Rank:   item.Rank,
				TmdbId: strconv.Itoa(item.Id),
				TvdbId: strconv.Itoa(item.TvdbId),
			})
		}
		hasMore = len(res.Data) == limit
		offset += limit
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	listCacheKey := strconv.Itoa(l.Id)
	if err := listCache.Add(listCacheKey, *l); err != nil {
		return err
	}

	return nil
}
