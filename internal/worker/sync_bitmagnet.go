package worker

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/kv"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
)

func InitSyncBitmagnetWorker(conf *WorkerConfig) *Worker {
	cursor := kv.NewKVStore[string](&kv.KVStoreConfig{
		Type: "job:sync-bitmagnet:cursor",
	})

	type DHTTorrent struct {
		Hash        string
		Title       string
		Seeders     int
		Leechers    int
		Size        int64
		FilesCount  int
		FilesStatus string
		Files       ts.Files
		ContentType string
		UpdatedAt   time.Time
	}

	limit := 500

	query_get_torrents := fmt.Sprintf(`
SELECT encode(tc.info_hash, 'hex'::text) AS hash,
       min(t.name)                       AS t_title,
       min(coalesce(tc.seeders, 0))      AS seeders,
       min(coalesce(tc.leechers, 0))     AS leechers,
       min(tc.size)                      AS size,
       min(tc.files_count)               AS files_count,
       min(t.files_status)               AS files_status,
       json_agg(json_build_object('i', coalesce(tf.index, 0),
                                  'n', coalesce(tf.path, t.name),
                                  's', coalesce(tf.size, tc.size))
                ORDER BY tf.index)       AS files,
       min(tc.content_type)              AS content_type,
       min(tc.updated_at)                AS updated_at
FROM torrent_contents tc
         LEFT JOIN torrents t ON tc.info_hash = t.info_hash
         LEFT JOIN torrent_files tf on t.info_hash = tf.info_hash
WHERE tc.info_hash IN (SELECT tc.info_hash
                       FROM torrent_contents tc
                       WHERE tc.updated_at >= $1
                         AND tc.content_type IN ('movie', 'tv_show')
                       ORDER BY tc.updated_at
                       LIMIT %d OFFSET $2)
GROUP BY tc.info_hash
ORDER BY min(tc.updated_at), tc.info_hash
`, limit)

	conf.Executor = func(w *Worker) error {
		log := w.Log

		if !isIMDBSyncedInLast24Hours() {
			log.Info("IMDB not synced yet today, skipping")
			return nil
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

		last_cursor_updated_at := ""
		if err := cursor.GetValue("updated_at", &last_cursor_updated_at); err != nil {
			return err
		} else if last_cursor_updated_at == "" {
			last_cursor_updated_at = "2020-01-01T00:00:00Z"
		}
		cursor_updated_at, err := time.Parse(time.RFC3339, last_cursor_updated_at)
		if err != nil {
			return err
		}

		hasMore := true
		offset := 0
		for hasMore {
			rows, err := database.Query(query_get_torrents, cursor_updated_at, offset)
			if err != nil {
				return err
			}
			defer rows.Close()

			torrents := []torrent_info.TorrentInfoInsertData{}

			for rows.Next() {
				t := DHTTorrent{}
				if err := rows.Scan(
					&t.Hash,
					&t.Title,
					&t.Seeders,
					&t.Leechers,
					&t.Size,
					&t.FilesCount,
					&t.FilesStatus,
					&t.Files,
					&t.ContentType,
					&t.UpdatedAt,
				); err != nil {
					return err
				}

				torrent := torrent_info.TorrentInfoInsertData{
					Hash:         t.Hash,
					TorrentTitle: t.Title,
					Size:         t.Size,
					Source:       torrent_info.TorrentInfoSourceDHT,
					Seeders:      t.Seeders,
					Leechers:     t.Leechers,
					Files:        make(ts.Files, len(t.Files)),
				}
				for i := range t.Files {
					f := &t.Files[i]
					torrent.Files[i] = ts.File{
						Name: filepath.Base(f.Name),
						Idx:  f.Idx,
						Size: f.Size,
					}
				}
				torrents = append(torrents, torrent)
				last_cursor_updated_at = t.UpdatedAt.UTC().Format(time.RFC3339)
			}

			if err := rows.Err(); err != nil {
				return err
			}

			if err := torrent_info.Upsert(torrents, "", false); err != nil {
				return err
			} else {
				log.Info("upserted torrents", "count", len(torrents))
			}

			if err := cursor.Set("updated_at", last_cursor_updated_at); err != nil {
				return err
			}

			hasMore = len(torrents) == limit
			offset += limit
		}

		return nil
	}

	return NewWorker(conf)
}
