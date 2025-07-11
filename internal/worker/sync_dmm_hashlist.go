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

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/dmm_hashlist"
	"github.com/MunifTanjim/stremthru/internal/lzstring"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

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
	REPO_URL := util.MustParseURL("https://github.com/debridmediamanager/hashlists.git")
	if config.Integration.GitHub.User != "" && config.Integration.GitHub.Token != "" {
		REPO_URL.User = url.UserPassword(config.Integration.GitHub.User, config.Integration.GitHub.Token)
	}

	REPO_DIR := path.Join(config.DataDir, "hashlists")
	hashlistFilenameRegex := regexp.MustCompile(`\S{8}-\S{4}-\S{4}-\S{4}-\S{12}\.html`)

	ensureRepository := func(w *Worker) error {
		repoDirExists, err := util.DirExists(REPO_DIR)
		if err != nil {
			return err
		}
		if repoDirExists {
			w.Log.Info("updating repository")
			cmd := util.NewCommand("git", "-C", REPO_DIR, "remote", "get-url", "origin")
			err = cmd.Run()
			if err != nil {
				w.Log.Error("failed to get remote url", "error", err, "cmd_error", cmd.Error())
				return err
			}
			if remote_url := strings.TrimSpace(cmd.Output()); remote_url != REPO_URL.String() {
				cmd = util.NewCommand("git", "-C", REPO_DIR, "config", "credential.helper", "")
				err = cmd.Run()
				if err != nil {
					w.Log.Error("failed to set credential helper", "error", err, "cmd_error", cmd.Error())
					return err
				}
				cmd = util.NewCommand("git", "-C", REPO_DIR, "remote", "set-url", "origin", REPO_URL.String())
				err = cmd.Run()
				if err != nil {
					w.Log.Error("failed to set remote url", "error", err, "cmd_error", cmd.Error())
					return err
				}
			}
			cmd = util.NewCommand("git", "-C", REPO_DIR, "fetch", "--depth=1")
			err = cmd.Run()
			if err != nil {
				w.Log.Error("failed to fetch repository", "error", err, "cmd_error", cmd.Error())
				return err
			}
			cmd = util.NewCommand("git", "-C", REPO_DIR, "reset", "--hard", "origin/main")
			err = cmd.Run()
			if err != nil {
				w.Log.Error("failed to reset repository", "error", err, "cmd_error", cmd.Error())
				return err
			}
			w.Log.Info("repository updated")
		} else {
			w.Log.Info("cloning repository")
			cmd := exec.Command("git", "clone", "--config=credential.helper=", "--depth=1", "--single-branch", "--branch=main", REPO_URL.String(), REPO_DIR)
			err = cmd.Run()
			if err != nil {
				w.Log.Error("failed to clone repository", "error", err, "cmd_error", cmd.Stderr)
				return err
			}
			w.Log.Info("repository cloned")
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

	processHashlistFile := func(w *Worker, filename string, hashSeen *cache.LRUCache[struct{}], totalCount int) (int, error) {
		id := strings.TrimSuffix(filename, ".html")

		if exists, err := dmm_hashlist.Exists(id); err != nil {
			return totalCount, Error{"failed to check if hashlist already processed", err}
		} else if exists {
			w.Log.Debug("hashlist already processed", "id", id)
			return totalCount, nil
		}

		w.Log.Info("processing hashlist", "id", id)

		items, err := extractHashlistItems(filename)
		if err != nil {
			return totalCount, err
		}

		hashes := []string{}
		itemByHash := map[string]DMMHashlistItem{}
		for _, item := range items {
			magnet, err := core.ParseMagnetLink(item.Hash)
			if err != nil || len(magnet.Hash) != 40 {
				continue
			}
			hash := magnet.Hash
			if hashSeen.Get(hash, &struct{}{}) {
				continue
			}
			if _, found := itemByHash[hash]; found {
				continue
			}
			if item.Bytes == 0 || item.Filename == "" || item.Filename == "Magnet" {
				continue
			}
			hashes = append(hashes, hash)
			itemByHash[hash] = item
		}

		existsMap, err := torrent_info.ExistsByHash(hashes)
		if err != nil {
			w.Log.Error("failed to get torrent info", "error", err)
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
		w.Log.Info("upserted entries", "id", id, "count", hTotalCount)
		err = dmm_hashlist.Insert(id, len(items))
		return totalCount + hTotalCount, err
	}

	conf.Executor = func(w *Worker) error {
		hashSeenLru := cache.NewLRUCache[struct{}](&cache.CacheConfig{
			Name:          "worker:dmm_hashlist:seen",
			LocalCapacity: 100000,
		})

		if err := ensureRepository(w); err != nil {
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
			newTotalCount, err := processHashlistFile(w, filename, hashSeenLru, totalCount)
			if err != nil {
				return err
			}
			if newTotalCount != totalCount {
				w.Log.Info("upserted entries", "totalCount", totalCount)
			}
			totalCount = newTotalCount
		}

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
