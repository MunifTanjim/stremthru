package trakt

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
)

var listCache = cache.NewCache[TraktList](&cache.CacheConfig{
	Lifetime:      6 * time.Hour,
	Name:          "trakt:list",
	LocalCapacity: 1024,
})

var listIdBySlugCache = cache.NewCache[string](&cache.CacheConfig{
	Lifetime:      12 * time.Hour,
	Name:          "trakt:list-id-by-slug",
	LocalCapacity: 2048,
})

func getListCacheKey(l *TraktList, tokenId string) string {
	if l.IsUserSpecific() {
		return tokenId + ":" + l.Id
	}
	return l.Id
}

var syncListMutex sync.Mutex

func syncList(l *TraktList, tokenId string) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	isDynamic, isStandard, isUserSpecific := l.IsDynamic(), l.IsStandard(), l.IsUserSpecific()

	client := GetAPIClient(tokenId)

	var list *List
	if isDynamic {
		log.Debug("fetching dynamic list by id", "id", l.Id)
		meta := GetDynamicListMeta(l.Id)
		if meta == nil {
			return errors.New("invalid id")
		}
		now := time.Now()
		slug := strings.TrimPrefix(l.Id, "~:")
		privacy := ListPrivacyPublic
		if isStandard {
			slug, _, _ = strings.Cut(slug, ":")
		} else if isUserSpecific {
			slug = strings.TrimPrefix(slug, "u:")
			privacy = ListPrivacyPrivate
		}
		list = &List{
			Name:      meta.Name,
			Privacy:   privacy,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if meta.HasUserId {
			list.User.Ids.Slug = meta.UserId
		}
		list.Ids.Slug = slug
	} else if l.Id != "" {
		log.Debug("fetching list by id", "id", l.Id)
		listId, _ := strconv.Atoi(l.Id)
		res, err := client.FetchList(&FetchListParams{
			ListId: listId,
		})
		if err != nil {
			return err
		}
		list = &res.Data
	} else if l.UserId != "" && l.Slug != "" {
		log.Debug("fetching list by slug", "slug", l.UserId+"/"+l.Slug)
		res, err := client.FetchPersonalList(&FetchPersonalListParams{
			UserId: l.UserId,
			ListId: l.Slug,
		})
		if err != nil {
			return err
		}
		list = &res.Data
	} else {
		return errors.New("either id, or user_id and slug must be provided")
	}

	if list == nil {
		return errors.New("list not found")
	}

	if l.Id == "" {
		l.Id = strconv.Itoa(list.Ids.Trakt)
	}
	l.UserId = list.User.Ids.Slug
	l.UserName = list.User.Name
	l.Name = list.Name
	l.Slug = list.Ids.Slug
	l.Description = list.Description
	l.Private = list.Privacy != ListPrivacyPublic
	l.Likes = list.Likes
	l.Items = nil

	log.Debug("fetching list items", "id", l.Id)
	var res APIResponse[FetchListItemsData]
	var err error
	if isDynamic {
		res, err = client.fetchDynamicListItems(&fetchDynamicListItemsParams{
			id: l.Id,
		})
	} else {
		res, err = client.FetchListItems(&FetchListItemsParams{
			ListId:   list.Ids.Trakt,
			Extended: "full,images",
		})
	}
	if err != nil {
		return err
	}
	seenMap := map[int]struct{}{}
	for i := range res.Data {
		item := &res.Data[i]

		var data listItemCommon
		switch item.Type {
		case ItemTypeMovie:
			data = item.Movie.listItemCommon
		case ItemTypeShow:
			data = item.Show.listItemCommon
		case ItemTypeEpisode, ItemTypeSeason:
			if item.Show == nil {
				continue
			}
			data = item.Show.listItemCommon
		default:
			continue
		}

		if _, seen := seenMap[data.Ids.Trakt]; seen {
			continue
		}
		seenMap[data.Ids.Trakt] = struct{}{}

		lItem := TraktItem{
			Id:        data.Ids.Trakt,
			Type:      item.Type,
			Title:     data.Title,
			Year:      data.Year,
			Overview:  data.Overview,
			Runtime:   data.Runtime,
			Trailer:   data.Trailer,
			Rating:    int(data.Rating * 10),
			MPARating: data.Certification,

			Idx:    i,
			Genres: data.Genres,
			Ids:    data.Ids,
		}

		switch lItem.Type {
		case ItemTypeEpisode, ItemTypeSeason:
			lItem.Type = ItemTypeShow
		}

		if len(data.Images.Poster) > 0 {
			lItem.Poster = data.Images.Poster[0]
		}

		if len(data.Images.Fanart) > 0 {
			lItem.Fanart = data.Images.Fanart[0]
		}

		l.Items = append(l.Items, lItem)
	}

	if err := UpsertList(l); err != nil {
		return err
	}

	if err := listCache.Add(getListCacheKey(l, tokenId), *l); err != nil {
		return err
	}

	return nil
}

func (l *TraktList) Fetch(tokenId string) error {
	isMissing := false

	if l.Id == "" {
		if l.UserId == "" || l.Slug == "" {
			return errors.New("either id, or user_id and slug must be provided")
		}
		listIdBySlugCacheKey := l.UserId + "/" + l.Slug
		if !listIdBySlugCache.Get(listIdBySlugCacheKey, &l.Id) {
			if listId, err := GetListIdBySlug(l.UserId, l.Slug); err != nil {
				return err
			} else if listId == "" {
				isMissing = true
			} else {
				l.Id = listId
				log.Debug("found list id by slug", "id", l.Id, "slug", l.UserId+"/"+l.Slug)
				listIdBySlugCache.Add(listIdBySlugCacheKey, l.Id)
			}
		}
	}

	isUserSpecific := l.IsUserSpecific()

	listCacheKey := getListCacheKey(l, tokenId)
	if !isMissing {
		var cachedL TraktList
		if !listCache.Get(listCacheKey, &cachedL) {
			if !isUserSpecific {
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
