package worker

import (
	"sync"

	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	torznab_indexer "github.com/MunifTanjim/stremthru/internal/torznab/indexer"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

func InitTorznabIndexerSyncerWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		pendingItems, err := torznab_indexer_syncinfo.GetPending()
		if err != nil {
			log.Error("failed to get pending sync", "error", err)
			return err
		}

		if len(pendingItems) == 0 {
			log.Debug("no pending sync items")
			return nil
		}

		log.Info("processing pending sync items", "count", len(pendingItems))

		// Group items by indexer for efficiency
		type indexerKey struct {
			Type torznab_indexer.IndexerType
			Id   string
		}
		itemsByIndexer := make(map[string][]torznab_indexer_syncinfo.TorznabIndexerSyncInfo)
		for _, item := range pendingItems {
			key := string(item.Type) + ":" + item.Id
			itemsByIndexer[key] = append(itemsByIndexer[key], item)
		}

		// Get all indexers
		indexers, err := torznab_indexer.GetAll()
		if err != nil {
			log.Error("failed to get indexers", "error", err)
			return err
		}

		indexerById := make(map[string]*torznab_indexer.TorznabIndexer)
		for i := range indexers {
			indexer := &indexers[i]
			key := string(indexer.Type) + ":" + indexer.Id
			indexerById[key] = indexer
		}

		// Process each indexer's items
		for key, items := range itemsByIndexer {
			indexer, ok := indexerById[key]
			if !ok {
				log.Warn("indexer not found in vault", "key", key)
				continue
			}

			var client tznc.Indexer
			switch indexer.Type {
			case torznab_indexer.IndexerTypeJackett:
				c, err := indexer.GetClient()
				if err != nil {
					log.Error("failed to create torznab client", "error", err, "type", indexer.Type, "id", indexer.Id)
					return err
				}
				client = c
			default:
				log.Warn("unsupported indexer type", "type", indexer.Type)
				continue
			}

			log.Info("processing items for indexer", "indexer", indexer.Name, "count", len(items))

			// Process items for this indexer
			var wg sync.WaitGroup
			for i := range items {
				item := &items[i]

				wg.Add(1)
				go func(syncItem *torznab_indexer_syncinfo.TorznabIndexerSyncInfo) {
					defer wg.Done()

					// Parse the SId to get IMDB ID and optional season/episode
					nsid, err := torrent_stream.NormalizeStreamId(syncItem.SId)
					if err != nil {
						log.Error("failed to normalize stream ID", "error", err, "sid", syncItem.SId)
						return
					}

					// Get IMDB title for the query
					it, err := imdb_title.Get(nsid.Id)
					if err != nil {
						log.Error("failed to get IMDB title", "error", err, "imdb_id", nsid.Id)
						return
					}
					if it == nil {
						log.Warn("IMDB title not found", "imdb_id", nsid.Id)
						return
					}

					// Build search query
					query, err := client.NewSearchQuery(func(caps tznc.Caps) tznc.Function {
						if nsid.IsSeries() && caps.SupportsFunction(tznc.FunctionSearchTV) {
							return tznc.FunctionSearchTV
						}
						if caps.SupportsFunction(tznc.FunctionSearchMovie) {
							return tznc.FunctionSearchMovie
						}
						return tznc.FunctionSearch
					})
					if err != nil {
						log.Error("failed to create search query", "error", err, "indexer", client.GetId())
						return
					}

					query.SetLimit(-1)
					if query.IsSupported(tznc.SearchParamIMDBId) {
						query.Set(tznc.SearchParamIMDBId, nsid.Id)
						if nsid.IsSeries() {
							if query.IsSupported(tznc.SearchParamSeason) && nsid.Season != "" {
								query.Set(tznc.SearchParamSeason, nsid.Season)
								if query.IsSupported(tznc.SearchParamEp) && nsid.Episode != "" {
									query.Set(tznc.SearchParamEp, nsid.Episode)
								}
							}
						}
					}

					// Execute search
					results, err := client.Search(query)
					if err != nil {
						log.Error("indexer search failed", "error", err, "indexer", client.GetId(), "sid", syncItem.SId)
						return
					}

					log.Debug("indexer search completed", "indexer", client.GetId(), "sid", syncItem.SId, "count", len(results))

					if len(results) == 0 {
						// Mark as synced even if no results
						torznab_indexer_syncinfo.MarkSynced(syncItem.Type, syncItem.Id, syncItem.SId)
						return
					}

					// Process and save torrents
					tInfosToUpsert := []torrent_info.TorrentItem{}
					for i := range results {
						item := &results[i]
						if item.HasMissingData() {
							continue
						}

						tInfo := torrent_info.TorrentItem{
							Hash:         item.Hash,
							TorrentTitle: item.Title,
							Size:         item.Size,
							Indexer:      item.Indexer,
							Source:       torrent_info.TorrentInfoSourceIndexer,
							Seeders:      item.Seeders,
							Leechers:     item.Leechers,
							Private:      item.Private,
							Files:        item.Files,
						}
						tInfosToUpsert = append(tInfosToUpsert, tInfo)
					}

					if len(tInfosToUpsert) > 0 {
						category := torrent_info.TorrentInfoCategoryUnknown
						if nsid.IsSeries() {
							category = torrent_info.TorrentInfoCategorySeries
						} else {
							category = torrent_info.TorrentInfoCategoryMovie
						}

						if err := torrent_info.Upsert(tInfosToUpsert, category, false); err != nil {
							log.Error("failed to upsert torrent info", "error", err, "count", len(tInfosToUpsert))
							return
						}

						log.Debug("saved torrents", "indexer", client.GetId(), "sid", syncItem.SId, "count", len(tInfosToUpsert))
					}

					// Mark as synced
					torznab_indexer_syncinfo.MarkSynced(syncItem.Type, syncItem.Id, syncItem.SId)
				}(item)
			}

			wg.Wait()
		}

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
