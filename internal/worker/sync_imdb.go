package worker

import (
	"compress/gzip"
	"encoding/csv"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"

	"github.com/madflojo/tasks"
)

var syncIMDBJobTracker JobTracker[struct{}]

func generateSyncIMDBJobId() string {
	return time.Now().Format(time.DateOnly)
}

func isIMDBSyncedToday() bool {
	jobId := generateSyncIMDBJobId()
	job, err := syncIMDBJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

func InitSyncIMDBWorker() *tasks.Scheduler {
	log := logger.Scoped("worker/sync_imdb")

	syncIMDBJobTracker = NewJobTracker("sync-imdb", func(id string, job *Job[struct{}]) bool {
		date, err := time.Parse(time.DateOnly, id)
		if err != nil {
			return true
		}
		return date.Before(time.Now().Add(-7 * 24 * time.Hour))
	})

	jobTracker := syncIMDBJobTracker

	wd, err := os.Getwd()
	if err != nil {
		log.Error("failed to get working directory", "error", err)
		os.Exit(1)
		return nil
	}

	DOWNLOAD_DIR := path.Join(wd, "data", "imdb")
	err = util.EnsureDir(DOWNLOAD_DIR)
	if err != nil {
		log.Error("failed to ensure directory", "error", err)
		os.Exit(1)
		return nil
	}

	DATASET_URL := "https://datasets.imdbws.com/title.basics.tsv.gz"
	DATASET_FILENAME := filepath.Base(DATASET_URL)

	cleanArchives := func(date string) error {
		files, err := fs.Glob(os.DirFS(DOWNLOAD_DIR), "*.gz")
		if err != nil {
			return Error{"failed to list archives", err}
		}
		for _, filename := range files {
			if !strings.HasPrefix(filename, date+"-") {
				err := os.Remove(path.Join(DOWNLOAD_DIR, filename))
				if err != nil {
					log.Warn("failed to remove old archive", "filename", filename, "error", err)
				}
			}
		}
		return nil
	}

	downloadFile := func(url, filePath string) error {
		if exists, err := util.FileExists(filePath); err != nil {
			return Error{"failed to check existing file", err}
		} else if exists {
			return nil
		}

		log.Info("downloading...", "filename", filepath.Base(filePath))
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err == nil {
			log.Info("downloaded", "filename", filepath.Base(filePath))
		}
		return err
	}

	extractArchive := func(archivePath, outputPath string) error {
		if exists, err := util.FileExists(outputPath); err != nil {
			return Error{"failed to check existing file", err}
		} else if exists {
			isNewer, err := util.IsFileNewer(archivePath, outputPath)
			if err != nil {
				return Error{"failed to if archive is newer", err}
			}
			if !isNewer {
				return nil
			}
		}

		log.Info("extracting...", "filename", filepath.Base(outputPath))
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer f.Close()

		gzr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gzr.Close()

		out, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, gzr)
		if err == nil {
			log.Info("extracted", "filename", filepath.Base(outputPath))
		}
		return err
	}

	isAllowedType := func(tType string) bool {
		switch tType {
		case "short", "movie", "tvShort", "tvMovie", "tvSeries", "tvMiniSeries", "tvSpecial":
			return true
		default:
			return false
		}
	}

	nilValue := `\N`

	processDataset := func(filePath string, parse func(row []string) (*imdb_title.IMDBTitle, error)) error {
		f, err := os.Open(filePath)
		if err != nil {
			return Error{"failed to open file", err}
		}

		r := csv.NewReader(f)
		r.Comma = '\t'

		batch_size := 1000
		if db.Dialect == db.DBDialectPostgres {
			batch_size = 10000
		}
		titles := make([]imdb_title.IMDBTitle, batch_size)
		idx := -1
		batch_idx := 0

		log.Info("processing...")
		for {
			row, err := r.Read()

			if idx == -1 {
				idx++
				continue
			}

			if err == io.EOF {
				break
			}

			t, err := parse(row)
			if err != nil {
				return err
			}

			if t == nil {
				continue
			}

			titles[idx] = *t
			idx++
			if idx == batch_size {
				batch_idx++
				if err := imdb_title.Upsert(titles); err != nil {
					return err
				}
				log.Info("upserted titles", "count", batch_idx*batch_size)
				idx = 0
				time.Sleep(200 * time.Millisecond)
			}
		}
		titles = titles[0:idx]
		if err := imdb_title.Upsert(titles); err != nil {
			return err
		}
		log.Info("upserted titles", "count", batch_idx*batch_size+idx)
		return nil
	}

	scheduler := tasks.New()

	jobId := ""
	id, err := scheduler.Add(&tasks.Task{
		Interval:          time.Duration(24 * time.Hour),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.RecoverPanic(true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else {
					jobId = ""
				}
			}()

			if jobId != "" {
				return nil
			}

			jobId = generateSyncIMDBJobId()

			job, err := jobTracker.Get(jobId)
			if err != nil {
				return err
			}

			if job != nil && (job.Status == "done" || job.Status == "started") {
				log.Info("already done or started", "jobId", jobId, "status", job.Status)
				return nil
			}

			err = jobTracker.Set(jobId, "started", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "started")
				return err
			}

			archivePath := path.Join(DOWNLOAD_DIR, jobId+"-"+DATASET_FILENAME)
			datasetPath := path.Join(DOWNLOAD_DIR, strings.TrimSuffix(DATASET_FILENAME, ".gz"))

			if err := cleanArchives(jobId); err != nil {
				return err
			}

			if err = downloadFile(DATASET_URL, archivePath); err != nil {
				return err
			}

			if err = extractArchive(archivePath, datasetPath); err != nil {
				return err
			}

			err = processDataset(datasetPath, func(row []string) (*imdb_title.IMDBTitle, error) {
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

				return &imdb_title.IMDBTitle{
					TId:       tId,
					Title:     title,
					OrigTitle: origTitle,
					Year:      year,
					Type:      tType,
					IsAdult:   isAdult,
				}, nil
			})

			if err != nil {
				return err
			}

			err = jobTracker.Set(jobId, "done", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "done")
				return err
			}

			log.Info("rebuilding fts...")
			if err := imdb_title.RebuildFTS(); err != nil {
				return err
			}

			log.Info("done", "date", jobId)

			return nil
		},
		ErrFunc: func(err error) {
			log.Error("Worker Failure", "error", err)

			if terr := jobTracker.Set(jobId, "failed", err.Error(), nil); terr != nil {
				log.Error("failed to set job status", "error", terr, "jobId", jobId, "status", "failed")
			}

			jobId = ""
		},
	})

	if err != nil {
		panic(err)
	}

	log.Info("Started Worker", "id", id)

	if task, err := scheduler.Lookup(id); err == nil && task != nil {
		t := task.Clone()
		t.Interval = 30 * time.Second
		t.RunOnce = true
		scheduler.Add(t)
	}

	return scheduler
}
