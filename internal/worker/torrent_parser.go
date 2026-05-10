package worker

import (
	"slices"
	"time"

	"github.com/MunifTanjim/go-ptt"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
)

func InitParseTorrentWorker(conf *WorkerConfig) *Worker {
	if err := ti.MarkForReparseBelowVersion(9000); err != nil {
		panic(err)
	}

	var parseTorrentInfo = func(w *Worker, t *ti.TorrentInfo) *ti.TorrentInfo {
		if t.ParserVersion > ptt.Version().Int() {
			return nil
		}

		err := t.ForceParse()
		if err != nil {
			w.Log.Warn("failed to parse", "error", err, "title", t.TorrentTitle)
			return nil
		}

		return t
	}

	conf.Executor = func(w *Worker) error {
		log := w.Log
		for {
			tInfos, err := ti.GetUnparsed(5000)
			if err != nil {
				return err
			}

			for cTInfos := range slices.Chunk(tInfos, 500) {
				parsedTInfos := []*ti.TorrentInfo{}
				for i := range cTInfos {
					if t := parseTorrentInfo(w, &cTInfos[i]); t != nil {
						parsedTInfos = append(parsedTInfos, t)
					}
				}
				if err := ti.UpsertParsed(parsedTInfos); err != nil {
					return err
				}
				log.Info("upserted parsed torrent info", "count", len(parsedTInfos))
				time.Sleep(1 * time.Second)
			}

			if len(tInfos) < 5000 {
				break
			}

			time.Sleep(5 * time.Second)
		}

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
