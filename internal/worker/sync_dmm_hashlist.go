package worker

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/dmm_hashlist"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/lzstring"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"

	"github.com/madflojo/tasks"
)

func getSyncDMMHashlistJobId() string {
	return time.Now().Format(time.DateOnly + " 15")
}

type DMMHashlistItem struct {
	Filename string `json:"filename"`
	Hash     string `json:"hash"`
	Bytes    int64  `json:"bytes"`
}

type wrappedDMMHashlistItems struct {
	Title    string            `json:"title"`
	Torrents []DMMHashlistItem `json:"torrents"`
}

func InitSyncDMMHashlistWorker(conf *WorkerConfig) *Worker {
	if !config.Feature.IsEnabled("dmm_hashlist") {
		return nil
	}

	log := logger.Scoped("worker/sync_dmm_hashlist")

	jobTracker := NewJobTracker("sync-dmm-hashlist", func(id string, job *Job[struct{}]) bool {
		date, err := time.Parse(time.DateOnly+" 15", id)
		if err != nil {
			return true
		}
		return date.Before(time.Now().Add(-7 * 24 * time.Hour))
	})

	HASHLISTS_REPO := "https://github.com/debridmediamanager/hashlists.git"
	REPO_DIR := path.Join(config.DataDir, "hashlists")
	hashlistFilenameRegex := regexp.MustCompile(`\S{8}-\S{4}-\S{4}-\S{4}-\S{12}\.html`)

	ensureRepository := func() error {
		repoDirExists, err := util.DirExists(REPO_DIR)
		if err != nil {
			return err
		}
		if repoDirExists {
			log.Info("updating repository")
			cmd := exec.Command("git", "-C", REPO_DIR, "fetch", "--depth=1")
			err = cmd.Start()
			if err != nil {
				return err
			}
			err = cmd.Wait()
			if err != nil {
				return err
			}
			cmd = exec.Command("git", "-C", REPO_DIR, "reset", "--hard", "origin/main")
			err = cmd.Start()
			if err != nil {
				return err
			}
			err = cmd.Wait()
			if err != nil {
				return err
			}
			log.Info("repository updated")
		} else {
			log.Info("cloning repository")
			cmd := exec.Command("git", "clone", "--depth=1", "--single-branch", "--branch=main", HASHLISTS_REPO, REPO_DIR)
			err = cmd.Start()
			if err != nil {
				return err
			}
			err = cmd.Wait()
			if err != nil {
				return err
			}
			log.Info("repository cloned")
		}
		return nil
	}

	// <iframe .. src="https://..."
	urlRegex := regexp.MustCompile(`<iframe *src="(.+)".*>`)
	// <meta .. content="0;url=https://..."
	fallbackUrlRegex := regexp.MustCompile(`content="(.+)".*`)
	extractHashlistItems := func(filename string) ([]DMMHashlistItem, error) {
		file, err := os.Open(path.Join(REPO_DIR, filename))
		if err != nil {
			return nil, Error{"failed to get working directory", err}
		}
		defer file.Close()
		fileContent, err := io.ReadAll(file)
		if err != nil {
			return nil, Error{"failed to read file", err}
		}
		dataUrl := ""
		matches := urlRegex.FindAllStringSubmatch(string(fileContent), -1)
		if len(matches) > 0 {
			dataUrl = matches[0][1]
		}
		if dataUrl == "" {
			matches = fallbackUrlRegex.FindAllStringSubmatch(string(fileContent), -1)
			if len(matches) > 0 {
				dataUrl = matches[0][1]
				dataUrl = strings.TrimPrefix(dataUrl, "0;url=")
			}
		}
		if dataUrl == "" {
			return nil, errors.New("failed to extract data url")
		}
		u, err := url.Parse(dataUrl)
		if err != nil {
			return nil, Error{"failed to parse data url", err}
		}
		encodedData := u.Fragment
		if encodedData == "" {
			return nil, nil
		}
		blob, err := lzstring.DecompressFromEncodedUriComponent(encodedData)
		if err != nil {
			return nil, Error{"failed to decompress data", err}
		}
		items := []DMMHashlistItem{}
		if strings.HasPrefix(blob, "{") {
			wrappedItems := wrappedDMMHashlistItems{}
			err := json.Unmarshal([]byte(blob), &wrappedItems)
			if err != nil {
				return nil, Error{"failed to unmarshal wrapped hashlist items", err}
			}
			items = wrappedItems.Torrents
		} else {
			err := json.Unmarshal([]byte(blob), &items)
			if err != nil {
				return nil, Error{"failed to unmarshal hashlist items", err}
			}
		}
		return items, nil
	}

	processHashlistFile := func(filename string, hashSeen *cache.LRUCache[struct{}], totalCount int) (int, error) {
		id := strings.TrimSuffix(filename, ".html")

		if exists, err := dmm_hashlist.Exists(id); err != nil {
			return totalCount, Error{"failed to check if hashlist already processed", err}
		} else if exists {
			log.Debug("hashlist already processed", "id", id)
			return totalCount, nil
		}

		log.Info("processing hashlist", "id", id)

		items, err := extractHashlistItems(filename)
		if err != nil {
			return totalCount, err
		}

		hashes := []string{}
		itemByHash := map[string]DMMHashlistItem{}
		for _, item := range items {
			if hashSeen.Get(item.Hash, &struct{}{}) {
				continue
			}
			if _, found := itemByHash[item.Hash]; found {
				continue
			}
			if item.Bytes == 0 || item.Filename == "" || item.Filename == "Magnet" {
				continue
			}
			hashes = append(hashes, item.Hash)
			itemByHash[item.Hash] = item
		}

		existsMap, err := torrent_info.ExistsByHash(hashes)
		if err != nil {
			log.Error("failed to get torrent info", "error", err)
			return totalCount, err
		}
		for hash, exists := range existsMap {
			if exists {
				hashSeen.Add(hash, struct{}{})
			}
		}
		hTotalCount := 0
		for cHashes := range slices.Chunk(hashes, 500) {
			tInfos := []torrent_info.TorrentInfoInsertData{}
			for _, hash := range cHashes {
				if hashSeen.Get(hash, &struct{}{}) {
					continue
				}
				item := itemByHash[hash]
				tInfos = append(tInfos, torrent_info.TorrentInfoInsertData{
					Hash:         item.Hash,
					TorrentTitle: item.Filename,
					Size:         item.Bytes,
					Source:       torrent_info.TorrentInfoSourceDMM,
				})
			}
			hTotalCount += len(tInfos)
			torrent_info.Upsert(tInfos, "", true)
		}
		log.Info("upserted entries", "id", id, "count", hTotalCount)
		err = dmm_hashlist.Insert(id, len(items))
		return totalCount + hTotalCount, err
	}

	worker := &Worker{
		scheduler:  tasks.New(),
		shouldWait: conf.ShouldWait,
		onStart:    conf.OnStart,
		onEnd:      conf.OnEnd,
	}

	jobId := ""
	id, err := worker.scheduler.Add(&tasks.Task{
		Interval:          time.Duration(6 * time.Hour),
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

			jobId = getSyncDMMHashlistJobId()

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

			hashSeenLru := cache.NewLRUCache[struct{}](&cache.CacheConfig{
				Name:          "worker:dmm_hashlist:seen",
				LocalCapacity: 100000,
			})

			if err := ensureRepository(); err != nil {
				return err
			}

			files, err := fs.Glob(os.DirFS(REPO_DIR), "*.html")
			if err != nil {
				return err
			}

			totalCount := 0
			for _, filename := range files {
				if !hashlistFilenameRegex.MatchString(filename) {
					continue
				}
				newTotalCount, err := processHashlistFile(filename, hashSeenLru, totalCount)
				if err != nil {
					return err
				}
				if newTotalCount != totalCount {
					log.Info("upserted entries", "totalCount", totalCount)
				}
				totalCount = newTotalCount
			}

			err = jobTracker.Set(jobId, "done", "", nil)
			if err != nil {
				log.Error("failed to set job status", "error", err, "jobId", jobId, "status", "done")
				return err
			}

			log.Info("finished")
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
