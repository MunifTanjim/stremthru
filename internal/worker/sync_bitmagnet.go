package worker

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/bitmagnet"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/kv"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func InitSyncBitmagnetWorker(conf *WorkerConfig) *Worker {
	whitelistedExtension := map[string]struct{}{
		".vob": {},
		".iso": {},
	}

	cursor := kv.NewKVStore[string](&kv.KVStoreConfig{
		Type: "job:sync-bitmagnet:cursor",
	})

	conf.Executor = func(w *Worker) error {
		log := w.Log

		if !isIMDBSyncedInLast24Hours() {
			log.Info("IMDB not synced yet today, skipping")
			return nil
		}

		if version, err := bitmagnet.GetVersion(); err != nil {
			return err
		} else if !strings.HasPrefix(version, "v0.10.") {
			return errors.New("unsupported bitmagnet version: " + version + ". upgrade to v0.10.x")
		}

		connUri, err := db.ParseConnectionURI(config.Integration.Bitmagnet.DatabaseURI)
		if err != nil {
			return err
		}

		database, err := sql.Open(connUri.DriverName, connUri.DSN())
		if err != nil {
			return err
		}
		defer database.Close()

		last_stored_cursor_updated_at := ""
		last_cursor_updated_at := ""
		if err := cursor.GetValue("updated_at", &last_cursor_updated_at); err != nil {
			return err
		} else if last_cursor_updated_at == "" {
			last_cursor_updated_at = "2020-01-01T00:00:00Z"
		}
		cursor_updated_at := util.MustParseTime(time.RFC3339, last_cursor_updated_at).UTC()

		totalCount := 0
		limit := 500
		offset := 0
		hasMore := true
		for hasMore {
			items, err := bitmagnet.GetTorrents(database, limit, offset, cursor_updated_at)
			if err != nil {
				return err
			}

			torrents := []torrent_info.TorrentInfoInsertData{}

			for i := range items {
				t := &items[i]
				torrent := torrent_info.TorrentInfoInsertData{
					Hash:         t.Hash,
					TorrentTitle: t.Title,
					Size:         t.Size,
					Source:       torrent_info.TorrentInfoSourceDHT,
					Seeders:      t.Seeders,
					Leechers:     t.Leechers,
					Private:      t.Private,
					Files:        make(ts.Files, len(t.Files)),
				}
				hasValidFiles := false
				for i := range t.Files {
					f := &t.Files[i]
					path := f.Path
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					name := filepath.Base(path)
					if !hasValidFiles {
						hasValidFiles = core.HasVideoExtension(name)
						if !hasValidFiles {
							ext := strings.ToLower(filepath.Ext(name))
							_, hasValidFiles = whitelistedExtension[ext]
						}
					}

					torrent.Files[i] = ts.File{
						Path:   path,
						Name:   name,
						Idx:    f.Idx,
						Size:   f.Size,
						Source: string(torrent_info.TorrentInfoSourceDHT),
					}
				}
				if hasValidFiles {
					torrents = append(torrents, torrent)
				}
				last_cursor_updated_at = t.UpdatedAt.UTC().Format(time.RFC3339)
			}

			if err := torrent_info.Upsert(torrents, "", false); err != nil {
				return err
			} else {
				count := len(torrents)
				totalCount += count
				log.Info("upserted torrents", "count", count, "total_count", totalCount)
			}

			if last_stored_cursor_updated_at != last_cursor_updated_at {
				if err := cursor.Set("updated_at", last_cursor_updated_at); err != nil {
					return err
				}
				last_stored_cursor_updated_at = last_cursor_updated_at
				log.Debug("stored cursor", "updated_at", last_cursor_updated_at)
			}

			hasMore = len(items) >= limit
			offset += limit
		}

		return nil
	}

	return NewWorker(conf)
}
