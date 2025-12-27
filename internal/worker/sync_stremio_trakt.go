package worker

import (
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/meta"
	stremio_account "github.com/MunifTanjim/stremthru/internal/stremio/account"
	stremio_api "github.com/MunifTanjim/stremthru/internal/stremio/api"
	"github.com/MunifTanjim/stremthru/internal/stremio/cinemeta"
	"github.com/MunifTanjim/stremthru/internal/sync/stremio_trakt"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	trakt_account "github.com/MunifTanjim/stremthru/internal/trakt/account"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/stremio"
	stremio_watched_bitfield "github.com/MunifTanjim/stremthru/stremio/watched_bitfield"
)

func InitSyncStremioTraktWorker(conf *WorkerConfig) *Worker {
	type Ctx struct {
		now        time.Time
		log        *logger.Logger
		link       *sync_stremio_trakt.SyncStremioTraktLink
		isFullSync bool

		stremioAccount *stremio_account.StremioAccount
		stremioClient  *stremio_api.Client
		stremioToken   string
		stremioMovies  []stremio_api.LibraryItem
		stremioSeries  []stremio_api.LibraryItem

		traktAccount  *trakt_account.TraktAccount
		traktClient   *trakt.APIClient
		traktMovies   []trakt.HistoryItem
		traktEpisodes []trakt.HistoryItem
	}

	createLibraryItem := func(ctx *Ctx, meta stremio.Meta, state stremio_api.LibraryItemState) stremio_api.LibraryItem {
		return stremio_api.LibraryItem{
			Id:          meta.Id,
			Type:        string(meta.Type),
			Name:        meta.Name,
			Poster:      meta.Poster,
			PosterShape: meta.PosterShape,
			Background:  meta.Background,
			Logo:        meta.Logo,
			Year:        meta.ReleaseInfo,
			State:       state,
			Removed:     false,
			Temp:        false,
			CTime:       stremio_api.JSONTime{Time: ctx.now},
			MTime:       stremio_api.JSONTime{Time: ctx.now},
		}
	}

	getAllTraktHistory := func(client *trakt.APIClient, params trakt.GetHistoryParams) ([]trakt.HistoryItem, error) {
		page := 1
		limit := 100
		var allItems []trakt.HistoryItem
		for {
			p := params
			p.Page = page
			p.Limit = limit
			res, err := client.GetHistory(&p)
			if err != nil {
				return nil, err
			}
			allItems = append(allItems, res.Data...)
			if len(res.Data) < limit {
				break
			}
			page++
		}
		return allItems, nil
	}

	syncMovieFromStremioToTrakt := func(ctx *Ctx) error {
		traktWatchedImdbIds := util.NewSet[string]()
		for _, item := range ctx.traktMovies {
			if item.Movie == nil || item.Movie.Ids.IMDB == "" {
				continue
			}
			traktWatchedImdbIds.Add(item.Movie.Ids.IMDB)
		}

		var moviesToAdd []trakt.SyncHistoryParamsItem
		if ctx.isFullSync {
			for _, item := range ctx.stremioMovies {
				if item.State.TimesWatched == 0 || traktWatchedImdbIds.Has(item.Id) {
					continue
				}

				moviesToAdd = append(moviesToAdd, trakt.SyncHistoryParamsItem{
					Ids:       trakt.ListItemIds{IMDB: item.Id},
					WatchedAt: &item.State.LastWatched,
				})
			}
		} else {
			var imdbIds []string
			for _, item := range ctx.stremioMovies {
				if item.State.TimesWatched == 0 || traktWatchedImdbIds.Has(item.Id) {
					continue
				}
				imdbIds = append(imdbIds, item.Id)
			}
			idMaps, err := imdb_title.GetIdMapsByIMDBId(imdbIds)
			if err != nil {
				return err
			}
			traktIdByImdbId := make(map[string]int)
			for imdbId, idMap := range idMaps {
				if traktId := util.SafeParseInt(idMap.TraktId, 0); traktId > 0 {
					traktIdByImdbId[imdbId] = traktId
				}
			}

			newIdMaps := []meta.IdMap{}

			for _, item := range ctx.stremioMovies {
				if item.State.TimesWatched == 0 || traktWatchedImdbIds.Has(item.Id) {
					continue
				}

				traktId, hasTraktId := traktIdByImdbId[item.Id]
				if !hasTraktId {
					res, err := ctx.traktClient.LookupId(&trakt.LookupIdParams{
						IdType: trakt.IdTypeIMDB,
						Id:     item.Id,
						Type:   trakt.ItemTypeMovie,
					})
					if err != nil || len(res.Data) == 0 || res.Data[0].Movie == nil {
						continue
					}
					traktId = res.Data[0].Movie.Ids.Trakt
					newIdMaps = append(newIdMaps, res.Data[0].Movie.Ids.ToIdMap(trakt.ItemTypeMovie))
				}
				res, err := ctx.traktClient.GetHistory(&trakt.GetHistoryParams{
					Type: trakt.HistoryItemTypeMovies,
					Id:   traktId,
				})
				if err != nil {
					return err
				}
				if len(res.Data) > 0 {
					continue
				}
				moviesToAdd = append(moviesToAdd, trakt.SyncHistoryParamsItem{
					Ids:       trakt.ListItemIds{IMDB: item.Id},
					WatchedAt: &item.State.LastWatched,
				})
			}

			if len(newIdMaps) > 0 {
				util.LogError(ctx.log, meta.SetIdMaps(newIdMaps, meta.IdProviderIMDB), "failed to set id maps")
			}
		}

		if len(moviesToAdd) == 0 {
			return nil
		}

		_, err := ctx.traktClient.AddToHistory(&trakt.AddToHistoryParams{
			Movies: moviesToAdd,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced movies from stremio to trakt", "count", len(moviesToAdd))
		return nil
	}

	syncSeriesFromStremioToTrakt := func(ctx *Ctx) error {
		traktWatchedByImdbId := map[string]*util.Set[string]{}
		for _, item := range ctx.traktEpisodes {
			if item.Show == nil || item.Episode == nil || item.Show.Ids.IMDB == "" {
				continue
			}
			imdbId := item.Show.Ids.IMDB
			if traktWatchedByImdbId[imdbId] == nil {
				traktWatchedByImdbId[imdbId] = util.NewSet[string]()
			}
			traktWatchedByImdbId[imdbId].Add(
				fmt.Sprintf("%d:%d", item.Episode.Season, item.Episode.Number),
			)
		}

		traktIdByImdbId := make(map[string]int)
		if !ctx.isFullSync {
			var imdbIds []string
			for _, item := range ctx.stremioSeries {
				if item.State.Watched == "" {
					continue
				}
				imdbIds = append(imdbIds, item.Id)
			}
			idMaps, err := imdb_title.GetIdMapsByIMDBId(imdbIds)
			if err != nil {
				return err
			}
			for imdbId, idMap := range idMaps {
				if traktId := util.SafeParseInt(idMap.TraktId, 0); traktId > 0 {
					traktIdByImdbId[imdbId] = traktId
				}
			}
		}

		newIdMaps := []meta.IdMap{}

		var showsToAdd []trakt.SyncHistoryShow
		for _, item := range ctx.stremioSeries {
			if item.State.Watched == "" {
				continue
			}

			meta, err := cinemeta.FetchMeta("series", item.Id)
			if err != nil {
				return err
			}
			var videoIds []string
			for _, video := range meta.Videos {
				videoIds = append(videoIds, video.Id)
			}
			wbf, err := stremio_watched_bitfield.NewWatchedBitFieldFromString(item.State.Watched, videoIds)
			if err != nil {
				continue
			}

			if !ctx.isFullSync {
				imdbId := item.Id
				traktId, hasTraktId := traktIdByImdbId[imdbId]
				if !hasTraktId {
					res, err := ctx.traktClient.LookupId(&trakt.LookupIdParams{
						IdType: trakt.IdTypeIMDB,
						Id:     item.Id,
						Type:   trakt.ItemTypeShow,
					})
					if err != nil || len(res.Data) == 0 || res.Data[0].Show == nil {
						continue
					}
					traktId = res.Data[0].Show.Ids.Trakt
					newIdMaps = append(newIdMaps, res.Data[0].Show.Ids.ToIdMap(trakt.ItemTypeShow))
				}

				episodeHistoryItems, err := getAllTraktHistory(ctx.traktClient, trakt.GetHistoryParams{
					Type: trakt.HistoryItemTypeShows,
					Id:   traktId,
				})
				if err != nil {
					return err
				}
				for _, item := range episodeHistoryItems {
					if item.Episode == nil {
						continue
					}
					if traktWatchedByImdbId[imdbId] == nil {
						traktWatchedByImdbId[imdbId] = util.NewSet[string]()
					}
					traktWatchedByImdbId[imdbId].Add(
						fmt.Sprintf("%d:%d", item.Episode.Season, item.Episode.Number),
					)
				}
			}

			episodesBySeason := map[int][]trakt.SyncHistoryShowSeasonEpisode{}
			for _, videoId := range videoIds {
				if wbf.GetVideo(videoId) {
					parts := strings.Split(videoId, ":")
					if len(parts) < 3 {
						continue
					}

					season, episode := util.SafeParseInt(parts[1], 0), util.SafeParseInt(parts[2], 0)
					if season < 1 || episode < 1 {
						continue
					}

					if set, ok := traktWatchedByImdbId[item.Id]; ok && set.Has(
						fmt.Sprintf("%d:%d", season, episode),
					) {
						continue
					}

					episodesBySeason[season] = append(episodesBySeason[season], trakt.SyncHistoryShowSeasonEpisode{
						Number: episode,
					})
				}
			}

			if len(episodesBySeason) == 0 {
				continue
			}

			seasons := make([]trakt.SyncHistoryShowSeason, 0, len(episodesBySeason))
			for season, episodes := range episodesBySeason {
				seasons = append(seasons, trakt.SyncHistoryShowSeason{
					Number:   season,
					Episodes: episodes,
				})
			}
			showsToAdd = append(showsToAdd, trakt.SyncHistoryShow{
				SyncHistoryParamsItem: trakt.SyncHistoryParamsItem{
					Ids: trakt.ListItemIds{IMDB: item.Id},
				},
				Seasons: seasons,
			})
		}

		if len(newIdMaps) > 0 {
			util.LogError(ctx.log, meta.SetIdMaps(newIdMaps, meta.IdProviderIMDB), "failed to set id maps")
		}

		if len(showsToAdd) == 0 {
			return nil
		}

		_, err := ctx.traktClient.AddToHistory(&trakt.AddToHistoryParams{
			Shows: showsToAdd,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced series from stremio to trakt", "count", len(showsToAdd))
		return nil
	}

	syncMovieFromTraktToStremio := func(ctx *Ctx) error {
		var itemsToUpdate []stremio_api.LibraryItem

		stremioItemByImdbId := map[string]stremio_api.LibraryItem{}
		for _, item := range ctx.stremioMovies {
			stremioItemByImdbId[item.Id] = item
		}

		if !ctx.isFullSync {
			var idsToFetch []string
			for _, item := range ctx.traktMovies {
				if _, exists := stremioItemByImdbId[item.Movie.Ids.IMDB]; !exists {
					idsToFetch = append(idsToFetch, item.Movie.Ids.IMDB)
				}
			}
			if len(idsToFetch) > 0 {
				res, err := ctx.stremioClient.GetAllLibraryItems(&stremio_api.GetAllLibraryItemsParams{
					Ctx: stremio_api.Ctx{APIKey: ctx.stremioToken},
					Ids: idsToFetch,
				})
				if err != nil {
					return err
				}
				for _, item := range res.Data {
					if item.Type == "movie" {
						stremioItemByImdbId[item.Id] = item
					}
				}
			}
		}

		for _, item := range ctx.traktMovies {
			imdbId := item.Movie.Ids.IMDB
			libraryItem, ok := stremioItemByImdbId[imdbId]
			if ok {
				if libraryItem.State.TimesWatched > 0 {
					continue
				}
				libraryItem.MTime = stremio_api.JSONTime{Time: ctx.now}
			} else {
				meta, err := cinemeta.FetchMeta("movie", imdbId)
				if err != nil {
					return err
				}
				libraryItem = createLibraryItem(ctx, meta, stremio_api.LibraryItemState{})
			}
			libraryItem.State.TimesWatched = 1
			if item.WatchedAt.After(libraryItem.State.LastWatched) {
				libraryItem.State.LastWatched = item.WatchedAt
			}
			itemsToUpdate = append(itemsToUpdate, libraryItem)
		}

		if len(itemsToUpdate) == 0 {
			return nil
		}

		_, err := ctx.stremioClient.UpdateLibraryItems(&stremio_api.UpdateLibraryItemsParams{
			Ctx:     stremio_api.Ctx{APIKey: ctx.stremioToken},
			Changes: itemsToUpdate,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced movies from trakt to stremio", "count", len(itemsToUpdate))
		return nil
	}

	syncSeriesFromTraktToStremio := func(ctx *Ctx) error {
		var itemsToUpdate []stremio_api.LibraryItem

		stremioItemByImdbId := map[string]stremio_api.LibraryItem{}
		for _, item := range ctx.stremioSeries {
			stremioItemByImdbId[item.Id] = item
		}

		traktItemsByImdbId := map[string][]trakt.HistoryItem{}
		for _, item := range ctx.traktEpisodes {
			traktItemsByImdbId[item.Show.Ids.IMDB] = append(traktItemsByImdbId[item.Show.Ids.IMDB], item)
		}

		if !ctx.isFullSync {
			var idsToFetch []string
			for showId := range traktItemsByImdbId {
				if _, exists := stremioItemByImdbId[showId]; !exists {
					idsToFetch = append(idsToFetch, showId)
				}
			}
			if len(idsToFetch) > 0 {
				res, err := ctx.stremioClient.GetAllLibraryItems(&stremio_api.GetAllLibraryItemsParams{
					Ctx: stremio_api.Ctx{APIKey: ctx.stremioToken},
					Ids: idsToFetch,
				})
				if err != nil {
					return err
				}
				for _, item := range res.Data {
					if item.Type == "series" {
						stremioItemByImdbId[item.Id] = item
					}
				}
			}
		}

		for imdbId, traktItems := range traktItemsByImdbId {
			meta, err := cinemeta.FetchMeta("series", imdbId)
			if err != nil {
				return err
			}
			var videoIds []string
			for _, video := range meta.Videos {
				videoIds = append(videoIds, video.Id)
			}

			libraryItem, exists := stremioItemByImdbId[imdbId]
			var wbf *stremio_watched_bitfield.WatchedBitField
			if exists && libraryItem.State.Watched != "" {
				if wbf, err = stremio_watched_bitfield.NewWatchedBitFieldFromString(libraryItem.State.Watched, videoIds); err != nil {
					return err
				}
			} else {
				wbf = stremio_watched_bitfield.NewWatchedBitField(stremio_watched_bitfield.NewBitField8(len(videoIds)), videoIds)
			}

			needsUpdate := false
			var lastWatched time.Time
			for _, item := range traktItems {
				if item.Episode == nil {
					continue
				}
				videoId := fmt.Sprintf("%s:%d:%d", imdbId, item.Episode.Season, item.Episode.Number)
				if !wbf.GetVideo(videoId) {
					wbf.SetVideo(videoId, true)
					needsUpdate = true
					if item.WatchedAt.After(lastWatched) {
						lastWatched = item.WatchedAt
					}
				}
			}

			if needsUpdate {
				watchedStr, err := wbf.String()
				if err != nil {
					return err
				}

				if exists {
					libraryItem.MTime = stremio_api.JSONTime{Time: ctx.now}
				} else {
					libraryItem = createLibraryItem(ctx, meta, stremio_api.LibraryItemState{})
				}
				libraryItem.State.Watched = watchedStr
				if lastWatched.After(libraryItem.State.LastWatched) {
					libraryItem.State.LastWatched = lastWatched
				}
				if videoId := wbf.GetNextUnwatchedVideoId(); videoId != libraryItem.State.VideoId {
					libraryItem.State.VideoId = videoId
					libraryItem.State.TimeOffset = 0
				}
				itemsToUpdate = append(itemsToUpdate, libraryItem)
			}
		}

		if len(itemsToUpdate) == 0 {
			return nil
		}

		_, err := ctx.stremioClient.UpdateLibraryItems(&stremio_api.UpdateLibraryItemsParams{
			Ctx:     stremio_api.Ctx{APIKey: ctx.stremioToken},
			Changes: itemsToUpdate,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced series from trakt to stremio", "count", len(itemsToUpdate))
		return nil
	}

	syncWatched := func(link *sync_stremio_trakt.SyncStremioTraktLink, log *logger.Logger) error {
		log = log.With(
			"stremio_account_id", link.StremioAccountId,
			"trakt_account_id", link.TraktAccountId,
		)

		ctx := &Ctx{
			log:  log,
			link: link,
		}

		stremioAccount, err := stremio_account.GetById(link.StremioAccountId)
		if err != nil || stremioAccount == nil {
			return fmt.Errorf("stremio account not found: %w", err)
		}
		ctx.stremioAccount = stremioAccount

		traktAccount, err := trakt_account.GetById(link.TraktAccountId)
		if err != nil || traktAccount == nil {
			return fmt.Errorf("trakt account not found: %w", err)
		}
		ctx.traktAccount = traktAccount

		stremioToken, err := stremioAccount.GetValidToken()
		if err != nil {
			return err
		}
		ctx.stremioToken = stremioToken

		ctx.stremioClient = stremio_api.NewClient(&stremio_api.ClientConfig{})

		ctx.traktClient = trakt.GetAPIClient(traktAccount.OAuthTokenId)

		ctx.now = time.Now()

		var startAt time.Time
		if link.SyncState.Watched.LastSyncedAt != nil {
			startAt = *link.SyncState.Watched.LastSyncedAt
		}

		ctx.isFullSync = startAt.IsZero()

		log.Debug("starting watched sync", "is_full_sync", ctx.isFullSync, "start_at", startAt)

		var stremioItemIds []string
		if !ctx.isFullSync {
			tsRes, err := ctx.stremioClient.GetAllLibraryItemTimestamps(&stremio_api.GetAllLibraryItemTimestampsParams{Ctx: stremio_api.Ctx{APIKey: stremioToken}})
			if err != nil {
				return err
			}
			for _, ts := range tsRes.Data {
				if !strings.HasPrefix(ts.Id, "tt") {
					continue
				}
				if ts.ModifiedAt.After(startAt) {
					stremioItemIds = append(stremioItemIds, ts.Id)
				}
			}
		}

		if ctx.isFullSync || len(stremioItemIds) > 0 {
			stremioLibItemsRes, err := ctx.stremioClient.GetAllLibraryItems(&stremio_api.GetAllLibraryItemsParams{
				Ctx: stremio_api.Ctx{APIKey: stremioToken},
				Ids: stremioItemIds,
			})
			if err != nil {
				return err
			}
			for _, item := range stremioLibItemsRes.Data {
				if !strings.HasPrefix(item.Id, "tt") {
					continue
				}
				switch item.Type {
				case "movie":
					ctx.stremioMovies = append(ctx.stremioMovies, item)
				case "series":
					ctx.stremioSeries = append(ctx.stremioSeries, item)
				}
			}
		}

		log.Debug("fetched stremio items", "movies", len(ctx.stremioMovies), "series", len(ctx.stremioSeries))

		traktHistoryParams := trakt.GetHistoryParams{}
		if !ctx.isFullSync {
			traktHistoryParams.StartAt = &startAt
		}
		traktHistoryItems, err := getAllTraktHistory(ctx.traktClient, traktHistoryParams)
		if err != nil {
			return err
		}
		for _, item := range traktHistoryItems {
			switch item.Type {
			case trakt.ItemTypeMovie:
				if item.Movie == nil || item.Movie.Ids.IMDB == "" {
					continue
				}
				ctx.traktMovies = append(ctx.traktMovies, item)
			case trakt.ItemTypeEpisode:
				if item.Show == nil || item.Show.Ids.IMDB == "" {
					continue
				}
				ctx.traktEpisodes = append(ctx.traktEpisodes, item)
			}
		}

		log.Debug("fetched trakt items", "trakt_movies", len(ctx.traktMovies), "trakt_episodes", len(ctx.traktEpisodes))

		if link.SyncConfig.Watched.Direction.ShouldSyncToTrakt() {
			if err := syncMovieFromStremioToTrakt(ctx); err != nil {
				log.Error("failed to sync movies from stremio to trakt", "error", err)
				return err
			}
			if err := syncSeriesFromStremioToTrakt(ctx); err != nil {
				log.Error("failed to sync series from stremio to trakt", "error", err)
				return err
			}
		}

		if link.SyncConfig.Watched.Direction.ShouldSyncToStremio() {
			if err := syncMovieFromTraktToStremio(ctx); err != nil {
				log.Error("failed to sync movies from trakt to stremio", "error", err)
				return err
			}
			if err := syncSeriesFromTraktToStremio(ctx); err != nil {
				log.Error("failed to sync series from trakt to stremio", "error", err)
				return err
			}
		}

		link.SyncState.Watched.LastSyncedAt = &ctx.now
		sync_stremio_trakt.SetSyncState(link.StremioAccountId, link.TraktAccountId, link.SyncState)
		return nil
	}

	conf.Executor = func(w *Worker) error {
		log := w.Log

		links, err := sync_stremio_trakt.GetAll()
		if err != nil {
			return err
		}

		for _, link := range links {
			if !link.SyncConfig.Watched.Direction.IsDisabled() {
				err := syncWatched(&link, log)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}
	return NewWorker(conf)
}
