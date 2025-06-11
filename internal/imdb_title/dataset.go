package imdb_title

import (
	"path"
	"slices"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var datasetSyncMutex sync.Mutex

func SyncDataset() error {
	log = logger.Scoped("imdb_title/dataset")

	if !datasetSyncMutex.TryLock() {
		log.Warn("dataset sync already in progress, skipping")
		return nil
	}
	defer datasetSyncMutex.Unlock()

	isAllowedType := func(tType string) bool {
		switch tType {
		case "short", "movie", "tvShort", "tvMovie", "tvSeries", "tvMiniSeries", "tvSpecial":
			return true
		default:
			return false
		}
	}

	batch_size := 1000
	if db.Dialect == db.DBDialectPostgres {
		batch_size = 10000
	}
	writer := util.NewDatasetWriter(util.DatasetWriterConfig[IMDBTitle]{
		BatchSize: batch_size,
		Log:       log,
		Upsert: func(titles []IMDBTitle) error {
			return Upsert(titles)
		},
		SleepDuration: 200 * time.Millisecond,
	})

	ds := util.NewTSVDataset(&util.TSVDatasetConfig[IMDBTitle]{
		DatasetConfig: util.DatasetConfig{
			Archive:     "gz",
			DownloadDir: path.Join(config.DataDir, "imdb"),
			IsStale: func(t time.Time) bool {
				return t.Before(time.Now().Add(-24 * time.Hour))
			},
			Log: log,
			URL: "https://datasets.imdbws.com/title.basics.tsv.gz",
		},
		GetRowKey: func(row []string) string {
			return row[0]
		},
		HasHeaders: true,
		IsValidHeaders: func(headers []string) bool {
			return slices.Equal(headers, []string{
				"tconst",
				"titleType",
				"primaryTitle",
				"originalTitle",
				"isAdult",
				"startYear",
				"endYear",
				"runtimeMinutes",
				"genres",
			})
		},
		ParseRow: func(row []string) (*IMDBTitle, error) {
			nilValue := `\N`

			tType, err := util.TSVGetValue(row, 1, "", nilValue)
			if err != nil {
				return nil, err
			}
			if !isAllowedType(tType) {
				return nil, nil
			}

			tId, err := util.TSVGetValue(row, 0, "", nilValue)
			if err != nil {
				return nil, err
			}
			title, err := util.TSVGetValue(row, 2, "", nilValue)
			if err != nil {
				return nil, err
			}
			origTitle, err := util.TSVGetValue(row, 3, "", nilValue)
			if err != nil {
				return nil, err
			}
			isAdult, err := util.TSVGetValue(row, 4, false, nilValue)
			if err != nil {
				return nil, err
			}
			year, err := util.TSVGetValue(row, 5, 0, nilValue)
			if err != nil {
				return nil, err
			}

			if origTitle == title {
				origTitle = ""
			}

			return &IMDBTitle{
				TId:       tId,
				Title:     title,
				OrigTitle: origTitle,
				Year:      year,
				Type:      tType,
				IsAdult:   isAdult,
			}, nil
		},
		Writer: writer,
	})

	if err := ds.Process(); err != nil {
		return err
	}

	log.Info("rebuilding fts...")
	if err := RebuildFTS(); err != nil {
		return err
	}

	return nil
}
