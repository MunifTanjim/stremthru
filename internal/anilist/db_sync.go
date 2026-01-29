package anilist

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/anizip"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

var listCache = cache.NewCache[AniListList](&cache.CacheConfig{
	Lifetime: 6 * time.Hour,
	Name:     "mdblist:list:v2",
	MaxSize:  1024,
})

var anizipClient = anizip.NewAPIClient(&anizip.APIClientConfig{})

func EnsureIdMap(medias []AniListMedia, listId string) error {
	idMapGroup := anizip.GetMappingsPool().NewGroup()

	missingIdMapAnilistIds := []int{}
	for i := range medias {
		media := &medias[i]
		if media.IdMap == nil {
			missingIdMapAnilistIds = append(missingIdMapAnilistIds, media.Id)
			continue
		}
		if media.IdMap.IsStale() {
			idMapGroup.SubmitErr(func() (*anizip.GetMappingsData, error) {
				log.Debug("fetching stale idMap for media", "id", media.Id, "title", media.Title)
				return anizipClient.GetMappings(&anizip.GetMappingsParams{
					Service: anime.IdMapColumn.AniList,
					Id:      strconv.Itoa(media.Id),
				})
			})
		}
	}

	idMapByAniListId := map[string]*anime.AnimeIdMap{}

	if len(missingIdMapAnilistIds) > 0 {
		idMaps, err := anime.GetIdMapsForAniList(missingIdMapAnilistIds)
		if err != nil {
			return err
		}
		for i := range idMaps {
			idMap := &idMaps[i]
			idMapByAniListId[idMap.AniList] = idMap
		}
		for _, anilistId := range missingIdMapAnilistIds {
			if idMap, ok := idMapByAniListId[strconv.Itoa(anilistId)]; !ok || idMap.IsStale() {
				idMapGroup.SubmitErr(func() (*anizip.GetMappingsData, error) {
					log.Debug("fetching missing idMap for media", "id", anilistId)
					return anizipClient.GetMappings(&anizip.GetMappingsParams{
						Service: anime.IdMapColumn.AniList,
						Id:      strconv.Itoa(anilistId),
					})
				})
			}
		}
	}

	results, err := idMapGroup.Wait()
	if err != nil {
		return err
	}

	if len(results) > 0 {
		idMapItems := make([]anime.AnimeIdMap, 0, len(results))
		for i := range results {
			m := results[i].Mappings
			idMap := anime.AnimeIdMap{
				Type:        m.Type,
				AniDB:       strconv.Itoa(m.AniDB),
				AniList:     strconv.Itoa(m.AniList),
				AniSearch:   strconv.Itoa(m.AniSearch),
				AnimePlanet: m.AnimePlanet,
				IMDB:        m.IMDB,
				Kitsu:       strconv.Itoa(m.Kitsu),
				LiveChart:   strconv.Itoa(m.LiveChart),
				MAL:         strconv.Itoa(m.MAL),
				NotifyMoe:   m.NotifyMoe,
				TMDB:        m.TMDB,
				TVDB:        strconv.Itoa(m.TVDB),
				UpdatedAt:   db.Timestamp{Time: time.Now()},
			}
			idMapByAniListId[strconv.Itoa(m.AniList)] = &idMap
			idMapItems = append(idMapItems, idMap)
		}
		if err := anime.BulkRecordIdMaps(idMapItems, anime.IdMapColumn.AniList); err != nil {
			log.Error("failed to record idMaps", "error", err)
		}
	}

	for i := range medias {
		media := &medias[i]
		if idMap, ok := idMapByAniListId[strconv.Itoa(media.Id)]; ok {
			media.IdMap = idMap
		}
		if len(idMapByAniListId) > 0 {
			listCache.Remove(getListCacheKey(&AniListList{Id: listId}))
		}
	}

	return nil
}

func ScheduleIdMapSync(medias []AniListMedia) {
	for i := range medias {
		media := &medias[i]
		if media.IdMap == nil || media.IdMap.IsStale() {
			worker_queue.AnimeIdMapperQueue.Queue(worker_queue.AnimeIdMapperQueueItem{
				Service: anime.IdMapColumn.AniList,
				Id:      strconv.Itoa(media.Id),
			})
		}
	}
}

func getListCacheKey(l *AniListList) string {
	return l.Id
}

var syncListMutex sync.Mutex

func syncList(l *AniListList) error {
	syncListMutex.Lock()
	defer syncListMutex.Unlock()

	var list *List
	var err error
	log.Debug("fetching list by id", "id", l.Id)
	if l.GetUserName() == "~" {
		list, err = FetchSearchList(l.GetName())
	} else {
		list, err = FetchUserList(l.GetUserName(), l.GetName())
	}
	if err != nil {
		return err
	}

	if list == nil {
		return errors.New("list not found")
	}

	l.Id = list.GetId()
	l.Medias = nil

	mediaIds := list.MediaIds

	dbMedias, err := getMedias(mediaIds, list.ScoreByMediaId)
	if err != nil {
		return err
	}
	dbMediaById := map[int]*AniListMedia{}
	for i := range dbMedias {
		media := &dbMedias[i]
		dbMediaById[media.Id] = media
	}

	missingOrStaleMediaIds := []int{}
	for _, mediaId := range list.MediaIds {
		media, ok := dbMediaById[mediaId]
		if !ok || media.IsStale() {
			missingOrStaleMediaIds = append(missingOrStaleMediaIds, mediaId)
		}
	}

	if len(missingOrStaleMediaIds) > 0 {
		log.Debug("fetching list items", "id", l.Id)
		medias, err := FetchMedias(missingOrStaleMediaIds)
		if err != nil {
			return err
		}

		for i := range medias {
			media := &medias[i]
			dbMedia := AniListMedia{
				Id:          media.Id,
				Title:       media.Title,
				Type:        media.Format,
				Description: media.Description,
				Banner:      media.BannerImage,
				Cover:       media.CoverImage,
				Duration:    media.Duration,
				IsAdult:     media.IsAdult,
				StartYear:   media.StartYear,
				UpdatedAt:   db.Timestamp{Time: time.Now()},
				Genres:      media.Genres,
			}
			if dbM, ok := dbMediaById[media.Id]; ok {
				dbMedia.IdMap = dbM.IdMap
			}
			if score, ok := list.ScoreByMediaId[media.Id]; ok {
				dbMedia.Score = score
			}
			dbMediaById[media.Id] = &dbMedia
		}
	}

	for _, mediaId := range list.MediaIds {
		if media, ok := dbMediaById[mediaId]; ok {
			l.Medias = append(l.Medias, *media)
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

func (l *AniListList) Fetch() error {
	isMissing := false

	listCacheKey := getListCacheKey(l)
	var cachedL AniListList
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
