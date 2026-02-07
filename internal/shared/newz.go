package shared

import (
	"bytes"
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

type NZBFile struct {
	Blob []byte
	Name string
	Link string
}

func (b *NZBFile) CacheSize() int64 {
	return int64(len(b.Blob))
}

func (f *NZBFile) ToFileHeader() (*multipart.FileHeader, error) {
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

var nzbCache = cache.NewCache[NZBFile](&cache.CacheConfig{
	Name:       "newz_nzb",
	Lifetime:   config.NewzNZBCacheTTL,
	DiskBacked: true,
	MaxSize:    config.NewzNZBCacheSize,
})

func IsNZBCached(hash string) bool {
	return nzbCache.Has(hash)
}

var nzbFetchErrCache = cache.NewCache[string](&cache.CacheConfig{
	Name:     "newz_nzb_fetch_failure",
	Lifetime: 5 * time.Minute,
})

func HashNZBDownloadLink(link string) string {
	return util.MD5Hash(cleanNZBDownloadLink(link))
}

func cleanNZBDownloadLink(link string) string {
	link, _, ok := strings.Cut(link, "?")
	if !ok {
		link, _, _ = strings.Cut(link, "&")
	}
	return link
}

var nzbFileFetchSG singleflight.Group

var nzbFileFetcher = func() *http.Client {
	client := config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	client.Timeout = 60 * time.Second
	return client
}()

func FetchNZBFile(link string, name string, log *logger.Logger) (*NZBFile, error) {
	clink := cleanNZBDownloadLink(link)
	cacheKey := HashNZBDownloadLink(link)
	var nzbFile NZBFile
	if nzbCache.Get(cacheKey, &nzbFile) {
		if log != nil {
			log.Debug("fetch nzb - cache hit", "link", clink)
		}
	} else if fetchErr := ""; nzbFetchErrCache.Get(cacheKey, &fetchErr) {
		if log != nil {
			log.Debug("fetch nzb - cached failure", "link", clink)
		}
		return nil, fmt.Errorf("cached failure: %s", fetchErr)
	} else {
		if log != nil {
			log.Debug("fetch nzb - cache miss", "link", clink)
		}
		file, err, _ := nzbFileFetchSG.Do(cacheKey, func() (ret any, err error) {
			defer func() {
				if err == nil {
					return
				}
				if err := nzbFetchErrCache.Add(cacheKey, err.Error()); err != nil && log != nil {
					log.Warn("fetch nzb - failed to cache failure", "error", err, "link", clink)
				}
			}()

			req, err := http.NewRequest("GET", link, nil)
			if err != nil {
				return nil, err
			}
			req.Header = config.Newz.IndexerRequestHeader.Grab.Clone()
			res, err := nzbFileFetcher.Do(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()

			if res.StatusCode < 200 || 300 <= res.StatusCode {
				return nil, fmt.Errorf("failed to fetch nzb: status %d", res.StatusCode)
			}

			if res.ContentLength > config.NewzNZBMaxFileSize {
				return nil, fmt.Errorf("file too large: %d bytes (max %d)", res.ContentLength, config.NewzNZBMaxFileSize)
			}

			blob, err := io.ReadAll(io.LimitReader(res.Body, config.NewzNZBMaxFileSize+1024))
			if err != nil {
				if log != nil {
					log.Error("fetch nzb - failed", "error", err, "link", clink)
				}
				return nil, err
			}
			if size := int64(len(blob)); size > config.NewzNZBMaxFileSize {
				return nil, fmt.Errorf("file too large: %d+ bytes (max %d)", size, config.NewzNZBMaxFileSize)
			}
			if len(blob) == 0 {
				return nil, fmt.Errorf("empty response body")
			}
			if log != nil {
				log.Debug("fetch nzb - completed", "link", clink)
			}

			if name == "" {
				name = "unknown.nzb"
			}
			filename := name
			if cd := res.Header.Get("Content-Disposition"); cd != "" {
				_, params, _ := mime.ParseMediaType(cd)
				if fn := params["filename"]; fn != "" {
					filename = fn
				}
			}
			if filename == name {
				if fn := path.Base(link); strings.HasSuffix(fn, ".nzb") {
					filename = fn
				}
			}
			if !strings.HasSuffix(filename, ".nzb") {
				filename += ".nzb"
			}
			file := NZBFile{
				Blob: blob,
				Name: filename,
				Link: link,
			}
			err = nzbCache.Add(cacheKey, file)
			if err != nil && log != nil {
				log.Warn("fetch nzb - failed to cache", "error", err, "link", clink)
			}
			return file, nil
		})
		if err != nil {
			if log != nil {
				log.Error("fetch nzb - failed", "error", err, "link", clink)
			}
			return nil, err
		}
		nzbFile = file.(NZBFile)
	}
	return &nzbFile, nil
}
