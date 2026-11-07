package worker

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
)

func InitParseTorrentWorker(conf *WorkerConfig) *Worker {
	targetVersion := 9000
	if matchlab.Enabled() {
		targetVersion = matchlab.ParserVersion
	}
	if !matchlab.Enabled() {
		if err := ti.MarkForReparseBelowVersion(targetVersion); err != nil {
			panic(err)
		}
	}

	var parseTorrentInfo = func(w *Worker, t *ti.TorrentInfo) *ti.TorrentInfo {
		if !matchlab.Enabled() && t.ParserVersion > ptt.Version().Int() && t.ParserVersion != matchlab.ParserVersion {
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
		if matchlab.Enabled() {
			if err := matchlab.Default.Health(context.Background()); err != nil {
				return err
			}
		}
		for {
			tInfos, err := ti.GetUnparsed(5000)
			if err != nil {
				return err
			}

			for cTInfos := range slices.Chunk(tInfos, 500) {
				parsedTInfos := []*ti.TorrentInfo{}
				// Only already-MatchLab-parsed rows need an identity handover.
				// Fresh/native rows retain their original mappings.
				matchLabHandoverHashes := []string{}
				if matchlab.Enabled() {
					type parsedResult struct {
						info *ti.TorrentInfo
						err  error
					}
					results := make([]parsedResult, len(cTInfos))
					sem := make(chan struct{}, 8)
					var wg sync.WaitGroup
					for i := range cTInfos {
						wg.Add(1)
						sem <- struct{}{}
						go func(i int) {
							defer wg.Done()
							defer func() { <-sem }()
							t := &cTInfos[i]
							err := t.ForceParse()
							results[i] = parsedResult{info: t, err: err}
						}(i)
					}
					wg.Wait()
					for i, result := range results {
						if result.err != nil {
							log.Warn("MatchLab parser failed; retry deferred",
								"title", cTInfos[i].TorrentTitle, "error", result.err)
							if err := ti.RetryMatchLabParse(
								cTInfos[i].Hash, cTInfos[i].TorrentTitle, 5*time.Minute,
							); err != nil {
								return err
							}
						} else {
							parsedTInfos = append(parsedTInfos, result.info)
						}
					}
				} else {
					for i := range cTInfos {
						wasMatchLab := cTInfos[i].ParserVersion == matchlab.ParserVersion
						if t := parseTorrentInfo(w, &cTInfos[i]); t != nil {
							parsedTInfos = append(parsedTInfos, t)
							if wasMatchLab {
								matchLabHandoverHashes = append(matchLabHandoverHashes, t.Hash)
							}
						}
					}
				}
				// Clear authoritative MatchLab identities before writing the
				// native parser version. A failed native upsert remains retryable.
				if err := ti.ResetMatchLabIdentitiesForNativeMatching(matchLabHandoverHashes); err != nil {
					return err
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
