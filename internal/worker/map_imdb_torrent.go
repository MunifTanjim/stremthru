package worker

import (
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
)

type MapIMDBErrorHandler func(message string, err error, args ...any)

type MapIMDBResult struct {
	Item     *imdb_torrent.IMDBTorrent
	Category torrent_info.TorrentInfoCategory
}

func MapTorrentToIMDB(hash string, tInfo torrent_info.TorrentInfo, onError MapIMDBErrorHandler) *MapIMDBResult {
	if !tInfo.IsParsed() {
		return nil
	}

	ito := imdb_torrent.IMDBTorrent{
		Hash: hash,
	}

	if tInfo.Title == "" {
		return &MapIMDBResult{Item: &ito}
	}

	var category torrent_info.TorrentInfoCategory
	titleType := imdb_title.SearchTitleTypeUnknown
	if tInfo.Category == torrent_info.TorrentInfoCategoryMovie {
		titleType = imdb_title.SearchTitleTypeMovie
		category = torrent_info.TorrentInfoCategoryMovie
	} else if tInfo.Category == torrent_info.TorrentInfoCategorySeries || len(tInfo.Seasons) > 0 || len(tInfo.Episodes) > 0 {
		titleType = imdb_title.SearchTitleTypeShow
		category = torrent_info.TorrentInfoCategorySeries
	} else if tInfo.Category == torrent_info.TorrentInfoCategoryXXX {
		// ¯\_(ツ)_/¯
	} else {
		titleType = imdb_title.SearchTitleTypeMovie
		category = torrent_info.TorrentInfoCategoryMovie
	}

	imdbTitle, err := imdb_title.SearchOne(tInfo.Title, titleType, tInfo.Year, false)
	if err != nil {
		if onError != nil {
			onError("failed to search imdb title", err, "title", tInfo.Title, "year", tInfo.Year)
		}
		return nil
	}
	if imdbTitle != nil {
		switch {
		case category == torrent_info.TorrentInfoCategorySeries && !imdb_title.IMDBTitleType(imdbTitle.Type).IsShow():
		default:
			ito.TId = imdbTitle.TId
		}
	}
	return &MapIMDBResult{Item: &ito, Category: category}
}

func InitMapIMDBTorrentWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		if !isIMDBSyncedInLast24Hours() {
			w.Log.Info("IMDB not synced yet today, skipping")
			return nil
		}

		batch_size := 10000
		chunk_size := 1000
		if db.Dialect == db.DBDialectPostgres {
			batch_size = 20000
			chunk_size = 2000
		}

		totalCount := 0
		for {
			hashes, err := torrent_info.GetIMDBUnmappedHashes(batch_size)
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			for cHashes := range slices.Chunk(hashes, chunk_size) {
				wg.Go(func() {

					items := []imdb_torrent.IMDBTorrent{}
					tInfoByHash, err := torrent_info.GetByHashes(cHashes)
					if err != nil {
						w.Log.Error("failed to get torrent info", "error", err)
						return
					}
					hashesByCategory := map[torrent_info.TorrentInfoCategory][]string{
						torrent_info.TorrentInfoCategoryMovie:  {},
						torrent_info.TorrentInfoCategorySeries: {},
					}
					for hash, tInfo := range tInfoByHash {
						result := MapTorrentToIMDB(hash, tInfo, func(message string, err error, args ...any) {
							if err != nil {
								w.Log.Error(message, append([]any{"error", err}, args...)...)
							}
						})
						if result == nil {
							continue
						}
						items = append(items, *result.Item)
						if result.Category != "" {
							hashesByCategory[result.Category] = append(hashesByCategory[result.Category], hash)
						}
					}

					if err := imdb_torrent.Insert(items); err != nil {
						w.Log.Error("failed to map imdb torrent", "error", err)
						return
					}
					torrent_info.SetMissingCategory(hashesByCategory)

					w.Log.Info("mapped imdb torrent", "count", len(items))
				})
			}
			wg.Wait()

			count := len(hashes)
			totalCount += count
			w.Log.Info("processed torrents", "totalCount", totalCount)

			if count < batch_size {
				break
			}

			time.Sleep(200 * time.Millisecond)
		}

		return nil
	}

	return NewWorker(conf)
}
