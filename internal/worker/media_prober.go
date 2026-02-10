package worker

import (
	"context"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/media_probe"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
)

var query_update_media_info = fmt.Sprintf(
	"UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s = ? AND %s = ''",
	"torrent_stream",
	"mi",
	"uat", db.CurrentTimestamp,
	"h",
	"p",
	"mi",
)

var query_check_media_info = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ? AND %s = ?",
	"mi",
	"torrent_stream",
	"h",
	"p",
)

func InitMediaProberWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		log := w.Log

		worker_queue.MediaProberQueue.Process(func(item worker_queue.MediaProberQueueItem) error {
			var existing string
			row := db.QueryRow(query_check_media_info, item.Hash, item.Path)
			if err := row.Scan(&existing); err == nil && existing != "" {
				log.Debug("media info already exists", "hash", item.Hash, "path", item.Path)
				return nil
			}

			mi, err := media_probe.Probe(context.Background(), item.Link)
			if err != nil {
				log.Error("probe failed", "hash", item.Hash, "path", item.Path, "error", err)
				return nil
			}

			if _, err := db.Exec(query_update_media_info, mi, item.Hash, item.Path); err != nil {
				log.Error("failed to store media info", "hash", item.Hash, "path", item.Path, "error", err)
				return nil
			}

			log.Info("stored media info", "hash", item.Hash, "path", item.Path)
			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
