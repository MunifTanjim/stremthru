package mdblist

import (
	"errors"
	"strconv"
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
				listIdByNameCache.Add(listIdByNameCacheKey, l.Id)
			}
		}
	}

	if !isMissing {
		listCacheKey := strconv.Itoa(l.Id)
		if !listCache.Get(listCacheKey, l) {
			if list, err := GetListById(l.Id); err != nil {
				return err
			} else if list == nil {
				isMissing = true
			} else {
				*l = *list
				listCache.Add(listCacheKey, *l)
			}
		}
	}

	if !isMissing {
		return nil
	}

	var list *List
	if l.Id != 0 {
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

	if list.Private {
		return errors.New("list is private")
	}

	l.Id = list.Id
	l.UserId = list.UserId
	l.UserName = list.UserName
	l.Name = list.Name
	l.Slug = list.Slug
	l.Description = list.Description
	l.Mediatype = list.Mediatype
	l.Dynamic = list.Dynamic
	l.Likes = list.Likes

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
				Id:             item.Id,
				Rank:           item.Rank,
				Adult:          item.Adult == 1,
				Title:          item.Title,
				Poster:         item.Poster,
				ImdbId:         item.ImdbId,
				TvdbId:         item.TvdbId,
				Language:       item.Language,
				Mediatype:      item.Mediatype,
				ReleaseYear:    item.ReleaseYear,
				SpokenLanguage: item.SpokenLanguage,
				Genre:          item.Genre,
			})
		}
		hasMore = len(res.Data) == limit
		offset += limit
	}

	if err := UpsertList(l); err != nil {
		return nil
	}
	if err := UpsertItems(l.Items); err != nil {
		return nil
	}
	itemIds := make([]int, len(l.Items))
	for i := range l.Items {
		itemIds[i] = l.Items[i].Id
	}
	if err := SetListItems(l.Id, itemIds); err != nil {
		return err
	}
	listCacheKey := strconv.Itoa(l.Id)
	if err := listCache.Add(listCacheKey, *l); err != nil {
		return err
	}

	return nil
}
