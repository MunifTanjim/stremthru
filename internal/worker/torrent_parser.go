package worker

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/madflojo/tasks"
)

func InitParseTorrentWorker() *tasks.Scheduler {
	log := logger.Scoped("worker/torrent_parser")

	var parseTorrentInfo = func(t *ti.TorrentInfo) *ti.TorrentInfo {
		if t.ParserVersion > ptt.Version().Int() {
			return nil
		}

		r, err := util.ParseTorrentTitle(t.TorrentTitle)

		if err != nil {
			log.Warn("failed to parse", "error", err, "title", t.TorrentTitle)
			return nil
		}

		t.ParsedAt = db.Timestamp{Time: time.Now()}
		t.ParserVersion = ptt.Version().Int()
		t.ParserInput = t.TorrentTitle

		t.Audio = r.Audio
		t.BitDepth = r.BitDepth
		t.Channels = r.Channels
		t.Codec = r.Codec
		t.Commentary = r.Commentary
		t.Complete = r.Complete
		t.Container = r.Container
		t.Convert = r.Convert
		if r.Date != "" {
			if date, err := time.Parse(time.DateOnly, r.Date); err == nil {
				t.Date = db.DateOnly{Time: date}
			}
		}
		t.Documentary = r.Documentary
		t.Dubbed = r.Dubbed
		t.Edition = r.Edition
		t.EpisodeCode = r.EpisodeCode
		t.Episodes = r.Episodes
		t.Extended = r.Extended
		t.Extension = r.Extension
		t.Group = r.Group
		t.HDR = r.HDR
		t.Hardcoded = r.Hardcoded
		t.Languages = r.Languages
		t.Network = r.Network
		t.Proper = r.Proper
		t.Quality = r.Quality
		t.Region = r.Region
		t.Remastered = r.Remastered
		t.Repack = r.Repack
		t.Resolution = r.Resolution
		t.Retail = r.Retail
		t.Seasons = r.Seasons
		t.Site = r.Site
		if r.Size != "" {
			t.Size = util.ToBytes(r.Size)
		}
		t.Subbed = r.Subbed
		t.ThreeD = r.ThreeD
		t.Title = r.Title
		t.Uncensored = r.Uncensored
		t.Unrated = r.Unrated
		t.Upscaled = r.Upscaled
		t.Volumes = r.Volumes
		if r.Year != "" {
			year, year_end, _ := strings.Cut(r.Year, "-")
			t.Year, _ = strconv.Atoi(year)
			if year_end != "" {
				t.YearEnd, _ = strconv.Atoi(year_end)
			}
		}

		return t
	}

	scheduler := tasks.New()

	id, err := scheduler.Add(&tasks.Task{
		Interval:          time.Duration(5 * time.Minute),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.RecoverPanic(true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				}
			}()

			for {
				tInfos, err := ti.GetUnparsed(5000)
				if err != nil {
					return err
				}

				for cTInfos := range slices.Chunk(tInfos, 500) {
					parsedTInfos := []*ti.TorrentInfo{}
					for i := range cTInfos {
						if t := parseTorrentInfo(&cTInfos[i]); t != nil {
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
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	return scheduler
}
