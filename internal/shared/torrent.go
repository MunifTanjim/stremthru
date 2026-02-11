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

// MagnetRedirectError is returned by FetchTorrentFile when the download URL
// redirects to a magnet: URI instead of serving a .torrent file.
type MagnetRedirectError struct {
	MagnetURI string
}

func (e *MagnetRedirectError) Error() string {
	return fmt.Sprintf("torrent URL redirected to magnet link: %s", e.MagnetURI)
}

var torrentFileFetcher = func() *http.Client {
	client := config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), "magnet:") {
			return &MagnetRedirectError{MagnetURI: req.URL.String()}
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}
	return client
}()

func FetchTorrentFile(link string, maxSize int64) (*multipart.FileHeader, error) {
	res, err := torrentFileFetcher.Get(link)
	if err != nil {
		// http.Client wraps CheckRedirect errors in *url.Error.
		// Use errors.As to unwrap and surface magnet redirects to callers.
		var magnetErr *MagnetRedirectError
		if errors.As(err, &magnetErr) {
			return nil, magnetErr
		}
		return nil, err
	}
	defer res.Body.Close()

	if res.ContentLength <= 0 {
		return nil, fmt.Errorf("unable to determine torrent file size")
	}

	if res.ContentLength > maxSize {
		return nil, fmt.Errorf("torrent file too large: %d bytes (max %d)", res.ContentLength, maxSize)
	}

	blob, err := io.ReadAll(io.LimitReader(res.Body, maxSize+1))
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(blob); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(min(maxSize, res.ContentLength) + 1024) // Extra space for multipart headers
	if err != nil {
		return nil, fmt.Errorf("failed to read form: %w", err)
	}

	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("failed to extract file header")
	}

	return files[0], nil
}
