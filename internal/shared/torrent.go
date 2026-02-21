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

	"github.com/MunifTanjim/stremthru/internal/config"
)

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

func FetchTorrentFile(link string) (string, *multipart.FileHeader, error) {
	maxSize := config.Torz.TorrentFileMaxSize
	res, err := torrentFileFetcher.Get(link)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if http.StatusMovedPermanently <= res.StatusCode && res.StatusCode <= http.StatusPermanentRedirect {
		location := res.Header.Get("Location")
		if strings.HasPrefix(location, "magnet:") {
			return location, nil, nil
		}
	}

	if res.ContentLength > maxSize {
		return "", nil, fmt.Errorf("torrent file too large: %d bytes (max %d)", res.ContentLength, maxSize)
	}

	blob, err := io.ReadAll(io.LimitReader(res.Body, maxSize+1))
	if err != nil {
		return "", nil, err
	}

	if int64(len(blob)) == 0 {
		return "", nil, fmt.Errorf("empty torrent file response")
	}

	if int64(len(blob)) > maxSize {
		return "", nil, fmt.Errorf("torrent file too large: %d bytes (max %d)", len(blob), maxSize)
	}

	filename := "unknown.torrent"
	if cd := res.Header.Get("Content-Disposition"); cd != "" {
		_, params, _ := mime.ParseMediaType(cd)
		if fn := params["filename"]; fn != "" {
			filename = fn
		}
	}
	if filename == "unknown.torrent" {
		if fn := path.Base(link); strings.HasSuffix(fn, ".torrent") {
			filename = "unknown.torrent"
		}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(blob); err != nil {
		return "", nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", nil, fmt.Errorf("failed to close writer: %w", err)
	}

	contentLength := res.ContentLength
	if contentLength <= 0 {
		contentLength = int64(len(blob))
	}
	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(min(maxSize, contentLength) + 1024) // Extra space for multipart headers
	if err != nil {
		return "", nil, fmt.Errorf("failed to read form: %w", err)
	}

	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return "", nil, fmt.Errorf("failed to extract file header")
	}

	return "", files[0], nil
}
