package shared

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
	"golang.org/x/sync/singleflight"
)

type TorrentFile struct {
	Blob []byte
	Name string
	Link string
	Mod  time.Time
}

func (f *TorrentFile) CacheSize() int64 {
	return int64(len(f.Blob))
}

func (f *TorrentFile) ToFileHeader() (*multipart.FileHeader, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", f.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(f.Blob); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(f.CacheSize() + 1024)
	if err != nil {
		return nil, fmt.Errorf("failed to read form: %w", err)
	}

	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("failed to extract file header")
	}

	return files[0], nil
}

var torrentFileCache = cache.NewCache[TorrentFile](&cache.CacheConfig{
	Name:       "torz_torrent",
	Lifetime:   config.Torz.TorrentFileCacheTTL,
	DiskBacked: true,
	MaxSize:    config.Torz.TorrentFileCacheSize,
})

var torrentFetchErrCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "torz_torrent_fetch_failure",
	Lifetime: 5 * time.Minute,
})

func cleanTorrentFileLink(link string) string {
	link, _, ok := strings.Cut(link, "?")
	if !ok {
		link, _, _ = strings.Cut(link, "&")
	}
	link, _, _ = strings.Cut(link, "#")
	return link
}

func hashTorrentFileLink(link string) string {
	return util.MD5Hash(cleanTorrentFileLink(link))
}

var torrentFileFetchSG singleflight.Group

var torrentFileFetcher = func() *http.Client {
	client := config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if strings.EqualFold(req.URL.Scheme, "magnet") {
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return client
}()

func FetchTorrentFile(link string, name string, log *logger.Logger) (string, *multipart.FileHeader, error) {
	clink := cleanTorrentFileLink(link)

	var cacheKey string
	if !config.IsPublicInstance {
		cacheKey = hashTorrentFileLink(link)
	}

	var torrentFile TorrentFile
	if cacheKey != "" && torrentFileCache.Get(cacheKey, &torrentFile) {
		if log != nil {
			log.Debug("fetch torrent - cache hit", "link", clink)
		}
		fh, err := torrentFile.ToFileHeader()
		return "", fh, err
	}

	if fetchErr := ""; cacheKey != "" && torrentFetchErrCache.Get(cacheKey, &fetchErr) {
		if log != nil {
			log.Debug("fetch torrent - cached failure", "link", clink)
		}
		return "", nil, fmt.Errorf("cached failure: %s", fetchErr)
	}

	maxSize := config.Torz.TorrentFileMaxSize

	if log != nil {
		log.Debug("fetch torrent - cache miss", "link", clink)
	}
	result, err, _ := torrentFileFetchSG.Do(cacheKey, func() (ret any, err error) {
		defer func() {
			if err == nil {
				return
			}
			if err := torrentFetchErrCache.Add(cacheKey, err.Error()); err != nil && log != nil {
				log.Warn("fetch torrent - failed to cache failure", "error", err, "link", clink)
			}
		}()

		res, err := torrentFileFetcher.Get(link)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if http.StatusMovedPermanently <= res.StatusCode && res.StatusCode <= http.StatusPermanentRedirect {
			location := res.Header.Get("Location")
			if strings.HasPrefix(location, "magnet:") {
				return location, nil
			}
		}

		if res.ContentLength > maxSize {
			return nil, fmt.Errorf("file too large: %d bytes (max %d)", res.ContentLength, maxSize)
		}

		blob, err := io.ReadAll(io.LimitReader(res.Body, maxSize+1024))
		if err != nil {
			if log != nil {
				log.Error("fetch torrent - failed", "error", err, "link", clink)
			}
			return nil, err
		}

		if int64(len(blob)) == 0 {
			return nil, fmt.Errorf("empty torrent file response")
		}

		if int64(len(blob)) > maxSize {
			return nil, fmt.Errorf("torrent file too large: %d bytes (max %d)", len(blob), maxSize)
		}

		if log != nil {
			log.Debug("fetch torrent - completed", "link", clink)
		}

		if name == "" {
			name = "unknown.torrent"
		}
		filename := name
		if cd := res.Header.Get("Content-Disposition"); cd != "" {
			_, params, _ := mime.ParseMediaType(cd)
			if fn := params["filename"]; fn != "" {
				filename = fn
			}
		}
		if filename == name {
			if fn := path.Base(link); strings.HasSuffix(fn, ".torrent") {
				filename = fn
			}
		}
		if !strings.HasSuffix(filename, ".torrent") {
			filename += ".torrent"
		}

		file := TorrentFile{
			Blob: blob,
			Name: filename,
			Link: link,
			Mod:  time.Now(),
		}
		err = torrentFileCache.Add(cacheKey, file)
		if err != nil {
			if log != nil {
				log.Warn("fetch torrent - failed to cache", "error", err, "link", clink)
			}
		}
		return file, nil
	})

	if err != nil {
		if log != nil {
			log.Error("fetch torrent - failed", "error", err, "link", clink)
		}
		return "", nil, err
	}

	switch v := result.(type) {
	case string:
		return v, nil, nil
	case TorrentFile:
		fh, err := v.ToFileHeader()
		return "", fh, err
	default:
		return "", nil, fmt.Errorf("unexpected result type: %T", result)
	}
}
