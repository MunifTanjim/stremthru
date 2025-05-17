package worker

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"

	"github.com/madflojo/tasks"
)

var syncIMDBJobTracker JobTracker[struct{}]

func getTodayDateOnly() string {
	return time.Now().Format(time.DateOnly)
}

func isIMDBSyncedToday() bool {
	jobId := getTodayDateOnly()
	job, err := syncIMDBJobTracker.Get(jobId)
	if err != nil {
		return false
	}
	return job != nil && job.Status == "done"
}

type imdbTitleWriter struct {
	batch_idx  int
	batch_size int
	idx        int
	is_done    bool
	log        *slog.Logger
	titles     []imdb_title.IMDBTitle
}

func (w *imdbTitleWriter) Write(t *imdb_title.IMDBTitle) error {
	if w.is_done {
		return nil
	}

	if t == nil {
		return nil
	}

	w.titles[w.idx] = *t
	w.idx++
	if w.idx == w.batch_size {
		w.batch_idx++
		if err := imdb_title.Upsert(w.titles); err != nil {
			return err
		}
		w.log.Info("upserted titles", "count", w.batch_idx*w.batch_size)
		w.idx = 0
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func (w *imdbTitleWriter) Done() error {
	if w.is_done {
		return nil
	}

	w.is_done = true

	w.titles = w.titles[0:w.idx]
	if err := imdb_title.Upsert(w.titles); err != nil {
		return err
	}
	w.is_done = true
	w.log.Info("upserted titles", "count", w.batch_idx*w.batch_size+w.idx)
	return nil
}

func newIMDBTitleWriter(log *slog.Logger) imdbTitleWriter {
	batch_size := 1000
	if db.Dialect == db.DBDialectPostgres {
		batch_size = 10000
	}
	return imdbTitleWriter{
		batch_idx:  0,
		batch_size: batch_size,
		idx:        0,
		log:        log,
		titles:     make([]imdb_title.IMDBTitle, batch_size),
	}
}

func InitSyncIMDBWorker(conf *WorkerConfig) *Worker {
	if !config.Feature.IsEnabled("imdb_title") {
		return nil
	}

	log := logger.Scoped("worker/sync_imdb")

	syncIMDBJobTracker = NewJobTracker("sync-imdb", func(id string, job *Job[struct{}]) bool {
		date, err := time.Parse(time.DateOnly, id)
		if err != nil {
			return true
		}
		return date.Before(time.Now().Add(-7 * 24 * time.Hour))
	})

	jobTracker := syncIMDBJobTracker

	DOWNLOAD_DIR := path.Join(config.DataDir, "imdb")
	err := util.EnsureDir(DOWNLOAD_DIR)
	if err != nil {
		log.Error("failed to ensure directory", "error", err)
		os.Exit(1)
		return nil
	}

	DATASET_ARCHIVE_URL := "https://datasets.imdbws.com/title.basics.tsv.gz"
	DATASET_ARCHIVE_FILENAME := filepath.Base(DATASET_ARCHIVE_URL)

	listDatasets := func() ([]string, error) {
		return fs.Glob(os.DirFS(DOWNLOAD_DIR), "*-title.basics.tsv")
	}
	getOldDatasetFilename := func(newDatasetFilename string) (string, error) {
		filenames, err := listDatasets()
		if err != nil {
			return "", Error{"failed to list dataset files", err}
		}
		var lastFilename string
		var lastDate time.Time
		for _, filename := range filenames {
			if filename == newDatasetFilename {
				continue
			}
			fileDateString := filename[0:len(time.DateOnly)]
			fileDate, err := time.Parse(time.DateOnly, fileDateString)
			if err != nil {
				log.Error("failed to parse date", "filename", filename, "error", err)
				continue
			}
			if fileDate.After(lastDate) {
				lastDate = fileDate
				lastFilename = filename
			}
		}
		return lastFilename, nil
	}
	cleanupFiles := func(newFilename string, oldFilename string) error {
		filenames, err := listDatasets()
		if err != nil {
			return Error{"failed to list dataset files", err}
		}
		for _, filename := range filenames {
			if filename == newFilename || filename == oldFilename {
				continue
			}
			if err := os.Remove(path.Join(DOWNLOAD_DIR, filename)); err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Warn("failed to remove old dataset", "filename", filename, "error", err)
			}
			if err := os.Remove(path.Join(DOWNLOAD_DIR, filename+".gz")); err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Warn("failed to remove old archive", "filename", filename+".gz", "error", err)
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

	parseRow := func(row []string) (*imdb_title.IMDBTitle, error) {
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
	}

	csvReadRecord := func(r *csv.Reader) ([]string, error) {
		for {
			if record, err := r.Read(); err == nil || err == io.EOF {
				return record, err
			} else {
				log.Debug("failed to read row", "error", err)
			}
		}
	}

	processWholeDataset := func(filePath string) error {
		f, err := os.Open(filePath)
		if err != nil {
			return Error{"failed to open file", err}
		}

		r := csv.NewReader(f)
		r.Comma = '\t'
		r.LazyQuotes = true
		r.ReuseRecord = true

		w := newIMDBTitleWriter(log)

		_, _ = csvReadRecord(r)
		for {
			row, err := csvReadRecord(r)
			if err == io.EOF {
				break
			}

			t, err := parseRow(row)
			if err != nil {
				return err
			}

			if err := w.Write(t); err != nil {
				return err
			}
		}
		return w.Done()
	}

	processDiffDataset := func(oldDatasetPath, newDatasetPath string) error {
		w := newIMDBTitleWriter(log)

		oldFile, err := os.Open(oldDatasetPath)
		defer oldFile.Close()
		if err != nil {
			return err
		}
		newFile, err := os.Open(newDatasetPath)
		defer newFile.Close()
		if err != nil {
			return err
		}

		oldR := csv.NewReader(oldFile)
		oldR.Comma = '\t'
		oldR.LazyQuotes = true
		oldR.ReuseRecord = true

		newR := csv.NewReader(newFile)
		newR.Comma = '\t'
		newR.LazyQuotes = true
		newR.ReuseRecord = true

		_, _ = csvReadRecord(oldR)
		_, _ = csvReadRecord(newR)

		oldRec, oldErr := csvReadRecord(oldR)
		newRec, newErr := csvReadRecord(newR)

		for oldErr == nil && newErr == nil {
			oldKey := oldRec[0]
			newKey := newRec[0]

			switch {
			case oldKey < newKey:
				// removed
				oldRec, oldErr = csvReadRecord(oldR)
			case oldKey > newKey:
				// added
				if t, err := parseRow(newRec); err != nil {
					return err
				} else if err := w.Write(t); err != nil {
					return err
				}
				newRec, newErr = csvReadRecord(newR)
			default:
				if !slices.Equal(oldRec, newRec) {
					// changed
					if t, err := parseRow(newRec); err != nil {
						return err
					} else if err := w.Write(t); err != nil {
						return err
					}
				}
				oldRec, oldErr = csvReadRecord(oldR)
				newRec, newErr = csvReadRecord(newR)
			}
		}

		for newErr == nil {
			// added
			if t, err := parseRow(newRec); err != nil {
				return err
			} else if err := w.Write(t); err != nil {
				return err
			}
			newRec, newErr = csvReadRecord(newR)
		}
		if err := w.Done(); err != nil {
			return err
		}

		for oldErr == nil {
			// removed
			oldRec, oldErr = csvReadRecord(oldR)
		}
		return nil
	}

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	jobId := ""
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(24 * time.Hour),
		RunSingleInstance: true,
		TaskFunc: func() (err error) {
			defer func() {
				if perr, stack := util.HandlePanic(recover(), true); perr != nil {
					err = perr
					log.Error("Worker Panic", "error", err, "stack", stack)
				} else if err == nil {
					jobId = ""
				}
				worker.onEnd()
			}()

			for {
				wait, reason := worker.shouldWait()
				if !wait {
					break
				}
				log.Info("waiting, " + reason)
				time.Sleep(5 * time.Minute)
			}
			worker.onStart()

			if jobId != "" {
				return nil
			}

			jobId = getTodayDateOnly()

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

			newDate := jobId
			newDatasetFilename := newDate + "-" + strings.TrimSuffix(DATASET_ARCHIVE_FILENAME, ".gz")
			newArchivePath := path.Join(DOWNLOAD_DIR, newDatasetFilename+".gz")
			newDatasetPath := path.Join(DOWNLOAD_DIR, newDatasetFilename)

			oldDatasetFilename, err := getOldDatasetFilename(newDatasetFilename)
			if err != nil {
				return err
			}

			if err := cleanupFiles(newDatasetFilename, oldDatasetFilename); err != nil {
				return err
			}

			if err = downloadFile(DATASET_ARCHIVE_URL, newArchivePath); err != nil {
				return err
			}

			if err = extractArchive(newArchivePath, newDatasetPath); err != nil {
				return err
			}

			if oldDatasetFilename == "" {
				log.Info("processing whole dataset...")

				err = processWholeDataset(newDatasetPath)
				if err != nil {
					return err
				}
			} else {
				log.Info("processing diff dataset...")

				oldDatasetPath := path.Join(DOWNLOAD_DIR, oldDatasetFilename)
				err = processDiffDataset(oldDatasetPath, newDatasetPath)
				if err != nil {
					return err
				}
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

	if task, err := worker.scheduler.Lookup(id); err == nil && task != nil {
		t := task.Clone()
		t.Interval = 30 * time.Second
		t.RunOnce = true
		worker.scheduler.Add(t)
	}

	return worker
}
