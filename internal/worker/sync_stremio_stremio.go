package worker

import (
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	stremio_account "github.com/MunifTanjim/stremthru/internal/stremio/account"
	stremio_api "github.com/MunifTanjim/stremthru/internal/stremio/api"
	"github.com/MunifTanjim/stremthru/internal/stremio/cinemeta"
	"github.com/MunifTanjim/stremthru/internal/sync/stremio_stremio"
	"github.com/MunifTanjim/stremthru/internal/util"
	stremio_watched_bitfield "github.com/MunifTanjim/stremthru/stremio/watched_bitfield"
)

func InitSyncStremioStremioWorker(conf *WorkerConfig) *Worker {
	type Ctx struct {
		now        time.Time
		log        *logger.Logger
		link       *sync_stremio_stremio.SyncStremioStremioLink
		isFullSync bool
		includeIds []string

		accountA       *stremio_account.StremioAccount
		clientA        *stremio_api.Client
		tokenA         string
		accountAMovies []stremio_api.LibraryItem
		accountASeries []stremio_api.LibraryItem

		accountB       *stremio_account.StremioAccount
		clientB        *stremio_api.Client
		tokenB         string
		accountBMovies []stremio_api.LibraryItem
		accountBSeries []stremio_api.LibraryItem
	}

	type SyncParams struct {
		SourceMovies []stremio_api.LibraryItem
		TargetMovies []stremio_api.LibraryItem
		SourceSeries []stremio_api.LibraryItem
		TargetSeries []stremio_api.LibraryItem
		TargetClient *stremio_api.Client
		TargetToken  string
		Direction    string
	}

	syncMovies := func(ctx *Ctx, params *SyncParams) error {
		targetItemByImdbId := map[string]stremio_api.LibraryItem{}
		for _, item := range params.TargetMovies {
			targetItemByImdbId[item.Id] = item
		}

		var itemsToUpdate []stremio_api.LibraryItem

		for _, sourceItem := range params.SourceMovies {
			if sourceItem.State.TimesWatched == 0 {
				continue
			}

			targetItem, exists := targetItemByImdbId[sourceItem.Id]
			if exists {
				if targetItem.State.TimesWatched > 0 {
					continue
				}
				targetItem.MTime = stremio_api.JSONTime{Time: ctx.now}
			} else {
				targetItem = sourceItem
				targetItem.State = stremio_api.LibraryItemState{}
				targetItem.Removed = false
				targetItem.Temp = false
				targetItem.CTime = stremio_api.JSONTime{Time: ctx.now}
				targetItem.MTime = stremio_api.JSONTime{Time: ctx.now}
			}

			targetItem.State.TimesWatched = 1
			if sourceItem.State.LastWatched.After(targetItem.State.LastWatched) {
				targetItem.State.LastWatched = sourceItem.State.LastWatched
			}
			itemsToUpdate = append(itemsToUpdate, targetItem)
		}

		if len(itemsToUpdate) == 0 {
			return nil
		}

		_, err := params.TargetClient.UpdateLibraryItems(&stremio_api.UpdateLibraryItemsParams{
			Ctx:     stremio_api.Ctx{APIKey: params.TargetToken},
			Changes: itemsToUpdate,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced movies from account "+params.Direction, "count", len(itemsToUpdate))
		return nil
	}

	syncSeries := func(ctx *Ctx, params *SyncParams) error {
		targetItemByImdbId := map[string]stremio_api.LibraryItem{}
		for _, item := range params.TargetSeries {
			targetItemByImdbId[item.Id] = item
		}

		var itemsToUpdate []stremio_api.LibraryItem

		for _, sourceItem := range params.SourceSeries {
			if sourceItem.State.Watched == "" {
				continue
			}

			meta, err := cinemeta.FetchMeta("series", sourceItem.Id)
			if err != nil {
				ctx.log.Error("failed to fetch meta", "error", err, "imdb_id", sourceItem.Id)
				return err
			}
			var videoIds []string
			for _, video := range meta.Videos {
				videoIds = append(videoIds, video.Id)
			}

			sourceWbf, err := stremio_watched_bitfield.NewWatchedBitFieldFromString(sourceItem.State.Watched, videoIds)
			if err != nil {
				ctx.log.Error("failed to parse source watched bitfield", "error", err, "imdb_id", sourceItem.Id)
				return err
			}

			targetItem, exists := targetItemByImdbId[sourceItem.Id]
			var targetWbf *stremio_watched_bitfield.WatchedBitField
			if exists && targetItem.State.Watched != "" {
				if targetWbf, err = stremio_watched_bitfield.NewWatchedBitFieldFromString(targetItem.State.Watched, videoIds); err != nil {
					ctx.log.Error("failed to parse target watched bitfield", "error", err, "imdb_id", sourceItem.Id)
					return err
				}
			} else {
				targetWbf = stremio_watched_bitfield.NewWatchedBitField(stremio_watched_bitfield.NewBitField8(len(videoIds)), videoIds)
			}

			needsUpdate := false
			var lastWatched time.Time
			for _, videoId := range videoIds {
				if sourceWbf.GetVideo(videoId) && !targetWbf.GetVideo(videoId) {
					targetWbf.SetVideo(videoId, true)
					needsUpdate = true
					if sourceItem.State.LastWatched.After(lastWatched) {
						lastWatched = sourceItem.State.LastWatched
					}
				}
			}

			if needsUpdate {
				watchedStr, err := targetWbf.String()
				if err != nil {
					ctx.log.Error("failed to serialize watched bitfield", "error", err, "imdb_id", sourceItem.Id)
					return err
				}

				if exists {
					targetItem.MTime = stremio_api.JSONTime{Time: ctx.now}
				} else {
					targetItem = sourceItem
					targetItem.State = stremio_api.LibraryItemState{}
					targetItem.Removed = false
					targetItem.Temp = false
					targetItem.CTime = stremio_api.JSONTime{Time: ctx.now}
					targetItem.MTime = stremio_api.JSONTime{Time: ctx.now}
				}
				targetItem.State.Watched = watchedStr
				if lastWatched.After(targetItem.State.LastWatched) {
					targetItem.State.LastWatched = lastWatched
				}
				targetItem.State.VideoId = targetWbf.GetFirstUnwatchedVideoId()
				itemsToUpdate = append(itemsToUpdate, targetItem)
			}
		}

		if len(itemsToUpdate) == 0 {
			return nil
		}

		_, err := params.TargetClient.UpdateLibraryItems(&stremio_api.UpdateLibraryItemsParams{
			Ctx:     stremio_api.Ctx{APIKey: params.TargetToken},
			Changes: itemsToUpdate,
		})
		if err != nil {
			return err
		}

		ctx.log.Debug("synced series from account "+params.Direction, "count", len(itemsToUpdate))
		return nil
	}

	syncWatched := func(link *sync_stremio_stremio.SyncStremioStremioLink, log *logger.Logger) error {
		log = log.With(
			"account_a_id", link.AccountAId,
			"account_b_id", link.AccountBId,
		)

		ctx := &Ctx{
			log:  log,
			link: link,
		}

		includeIds := link.SyncConfig.Watched.Ids
		if len(includeIds) == 0 {
			log.Debug("skipping sync: include list is empty")
			return nil
		}
		ctx.includeIds = includeIds

		includeIdSet := util.NewSet[string]()
		for _, id := range includeIds {
			includeIdSet.Add(id)
		}

		accountA, err := stremio_account.GetById(link.AccountAId)
		if err != nil || accountA == nil {
			return fmt.Errorf("account A not found: %w", err)
		}
		ctx.accountA = accountA

		accountB, err := stremio_account.GetById(link.AccountBId)
		if err != nil || accountB == nil {
			return fmt.Errorf("account B not found: %w", err)
		}
		ctx.accountB = accountB

		tokenA, err := accountA.GetValidToken()
		if err != nil {
			return fmt.Errorf("failed to get valid token for account A: %w", err)
		}
		ctx.tokenA = tokenA

		tokenB, err := accountB.GetValidToken()
		if err != nil {
			return fmt.Errorf("failed to get valid token for account B: %w", err)
		}
		ctx.tokenB = tokenB

		ctx.clientA = stremio_api.NewClient(&stremio_api.ClientConfig{})
		ctx.clientB = stremio_api.NewClient(&stremio_api.ClientConfig{})

		ctx.now = time.Now()

		var startAt time.Time
		if link.SyncState.Watched.LastSyncedAt != nil {
			startAt = *link.SyncState.Watched.LastSyncedAt
		}

		ctx.isFullSync = startAt.IsZero()

		log.Debug("starting watched sync", "is_full_sync", ctx.isFullSync, "start_at", startAt, "include_count", len(includeIds))

		libItemsA, err := ctx.clientA.GetAllLibraryItems(&stremio_api.GetAllLibraryItemsParams{
			Ctx: stremio_api.Ctx{APIKey: tokenA},
			Ids: includeIds,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch library items for account A: %w", err)
		}
		for _, item := range libItemsA.Data {
			if !strings.HasPrefix(item.Id, "tt") || item.Removed || !includeIdSet.Has(item.Id) {
				continue
			}
			switch item.Type {
			case "movie":
				ctx.accountAMovies = append(ctx.accountAMovies, item)
			case "series":
				ctx.accountASeries = append(ctx.accountASeries, item)
			}
		}

		libItemsB, err := ctx.clientB.GetAllLibraryItems(&stremio_api.GetAllLibraryItemsParams{
			Ctx: stremio_api.Ctx{APIKey: tokenB},
			Ids: includeIds,
		})
		if err != nil {
			return fmt.Errorf("failed to fetch library items for account B: %w", err)
		}
		for _, item := range libItemsB.Data {
			if !strings.HasPrefix(item.Id, "tt") || item.Removed || !includeIdSet.Has(item.Id) {
				continue
			}
			switch item.Type {
			case "movie":
				ctx.accountBMovies = append(ctx.accountBMovies, item)
			case "series":
				ctx.accountBSeries = append(ctx.accountBSeries, item)
			}
		}

		log.Debug("fetched library items",
			"account_a_movies", len(ctx.accountAMovies),
			"account_a_series", len(ctx.accountASeries),
			"account_b_movies", len(ctx.accountBMovies),
			"account_b_series", len(ctx.accountBSeries),
		)

		if link.SyncConfig.Watched.Direction.ShouldSyncAToB() {
			params := &SyncParams{
				SourceMovies: ctx.accountAMovies,
				TargetMovies: ctx.accountBMovies,
				SourceSeries: ctx.accountASeries,
				TargetSeries: ctx.accountBSeries,
				TargetClient: ctx.clientB,
				TargetToken:  ctx.tokenB,
				Direction:    "A to B",
			}
			if err := syncMovies(ctx, params); err != nil {
				log.Error("failed to sync movies from A to B", "error", err)
				return err
			}
			if err := syncSeries(ctx, params); err != nil {
				log.Error("failed to sync series from A to B", "error", err)
				return err
			}
		}

		if link.SyncConfig.Watched.Direction.ShouldSyncBToA() {
			params := &SyncParams{
				SourceMovies: ctx.accountBMovies,
				TargetMovies: ctx.accountAMovies,
				SourceSeries: ctx.accountBSeries,
				TargetSeries: ctx.accountASeries,
				TargetClient: ctx.clientA,
				TargetToken:  ctx.tokenA,
				Direction:    "B to A",
			}
			if err := syncMovies(ctx, params); err != nil {
				log.Error("failed to sync movies from B to A", "error", err)
				return err
			}
			if err := syncSeries(ctx, params); err != nil {
				log.Error("failed to sync series from B to A", "error", err)
				return err
			}
		}

		link.SyncState.Watched.LastSyncedAt = &ctx.now
		err = sync_stremio_stremio.SetSyncState(link.AccountAId, link.AccountBId, link.SyncState)
		if err != nil {
			return err
		}

		return nil
	}

	conf.Executor = func(w *Worker) error {
		log := w.Log

		links, err := sync_stremio_stremio.GetAll()
		if err != nil {
			return err
		}

		for _, link := range links {
			if !link.SyncConfig.Watched.Direction.IsDisabled() {
				if err := syncWatched(&link, log); err != nil {
					log.Error("failed to sync link", "error", err,
						"account_a_id", link.AccountAId,
						"account_b_id", link.AccountBId,
					)
				}
			}
		}

		return nil
	}
	return NewWorker(conf)
}
