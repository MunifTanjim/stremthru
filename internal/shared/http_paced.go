// http_paced.go — progress-aware proxy for streaming-while-downloading.
//
// Problem: torrent clients pre-allocate files at their full size on disk. A
// file server (e.g. nginx) serving that file will happily return zero bytes
// for regions that haven't been downloaded yet. Video players interpret those
// zeros as corrupt data and crash or stall.
//
// Solution: this file wraps the normal proxy flow with download-progress
// awareness. It tracks how far the torrent client has sequentially downloaded
// ("safe bytes") and only forwards data up to that frontier, pausing when
// the player catches up to the download. For non-sequential seeks (e.g.
// ffprobe reading the moov atom at the end of a video file), it checks
// piece-level availability before proxying — the firstLastPiecePrio flag
// ensures the last piece is downloaded early, so these seeks usually succeed
// immediately.
//
// The pacing logic is torrent-client agnostic. Store-specific entry points
// (e.g. ProxyTorrentResponse for qBittorrent) wire up the progress callbacks
// from their respective APIs.

package shared

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

// parseByteRange extracts start and end from a "bytes=START-END" Range header.
// Returns start, end, ok. end == -1 means unbounded (e.g. "bytes=100-").
func parseByteRange(rangeHeader string) (start int64, end int64, ok bool) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(rangeHeader, "bytes=")
	if idx := strings.Index(spec, ","); idx >= 0 {
		spec = spec[:idx]
	}
	if strings.HasPrefix(spec, "-") {
		return 0, 0, false
	}
	parts := strings.SplitN(spec, "-", 2)
	s, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	e := int64(-1)
	if len(parts) == 2 && parts[1] != "" {
		e, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}
	return s, e, true
}

// SafeBytesFunc returns how many contiguous bytes from file offset 0 are fully
// downloaded and safe to serve, the total file size, and whether the download
// is complete.
type SafeBytesFunc func() (safeBytes int64, fileSize int64, done bool)

// IsRangeAvailableFunc checks whether every piece covering [start, end] is
// downloaded. This handles seeks outside the sequential frontier — typically
// the last piece (moov atom) which qBittorrent downloads early via
// firstLastPiecePrio.
type IsRangeAvailableFunc func(start, end int64) bool

// progressStallTimeout is how long we wait without any download progress
// before giving up. Two minutes covers slow torrents with few seeders.
const progressStallTimeout = 120 * time.Second

// ProxyTorrentResponse is the entry point called from the proxy handler when
// torrent_hash is present in the URL query. It builds progress callbacks from
// the torrent store API and delegates to proxyResponsePaced.
// Currently backed by qBittorrent, but the interface supports any torrent
// client that can report piece-level download progress.
func ProxyTorrentResponse(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType, user string, torrentHash string) (int64, error) {
	fileIdx := 0
	if fidxStr := r.URL.Query().Get("torrent_fidx"); fidxStr != "" {
		if v, err := strconv.Atoi(fidxStr); err == nil {
			fileIdx = v
		}
	}
	safeBytesFn := func() (int64, int64, bool) {
		safe, fileSize, done, err := GetQbitSafeBytes(user, torrentHash, fileIdx)
		if err != nil {
			return 0, 0, true
		}
		return safe, fileSize, done
	}
	isRangeAvailFn := func(start, end int64) bool {
		avail, err := IsQbitFileRangeAvailable(user, torrentHash, fileIdx, start, end)
		if err != nil {
			return false
		}
		return avail
	}
	return proxyResponsePaced(w, r, url, tunnelType, safeBytesFn, isRangeAvailFn)
}

// proxyResponsePaced proxies an HTTP response while pacing reads to match
// the download progress reported by safeBytesFn.
//
// Flow:
//  1. If the client sends a Range request beyond the sequential frontier,
//     check piece-level availability first (handles ffprobe seeks to EOF).
//     If available, proxy directly without pacing.
//  2. If not yet available, poll safeBytesFn every 2s until the download
//     catches up or progressStallTimeout is reached.
//  3. During streaming, read from the upstream file server in 64KB chunks
//     but never read past the safe byte frontier. When the player catches
//     up, pause and wait for more data.
func proxyResponsePaced(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType, safeBytesFn SafeBytesFunc, isRangeAvailFn IsRangeAvailableFunc) (bytesWritten int64, err error) {
	request, err := http.NewRequestWithContext(r.Context(), r.Method, url, nil)
	if err != nil {
		e := ErrorInternalServerError(r, "failed to create request")
		e.Cause = err
		SendError(w, r, e)
		return
	}

	copyHeaders(r.Header, request.Header, true)

	rangeVerified := false

	if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
		if start, end, ok := parseByteRange(rangeHeader); ok {
			safeBytes, fileSize, done := safeBytesFn()
			if start >= safeBytes && !done {
				// Range starts beyond what's been sequentially downloaded.
				// Check if those specific pieces are already available
				// (common for the last piece due to firstLastPiecePrio).
				if isRangeAvailFn != nil {
					rangeEnd := end
					if rangeEnd < 0 {
						rangeEnd = fileSize - 1
					}
					if isRangeAvailFn(start, rangeEnd) {
						rangeVerified = true
					}
				}

				// Not available at piece level — wait for sequential download
				// to catch up.
				if !rangeVerified {
					deadline := time.Now().Add(progressStallTimeout)
					for start >= safeBytes && !done && time.Now().Before(deadline) {
						time.Sleep(2 * time.Second)
						safeBytes, fileSize, done = safeBytesFn()
					}
					if start >= safeBytes && !done {
						w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(fileSize, 10))
						w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
						return 0, nil
					}
				}
			}
		}
	}

	proxyHttpClient := proxyHttpClientByTunnelType[tunnelType]

	response, err := proxyHttpClient.Do(request)
	if err != nil {
		e := ErrorBadGateway(r, "failed to request url")
		e.Cause = err
		SendError(w, r, e)
		return
	}
	defer response.Body.Close()

	copyHeaders(response.Header, w.Header(), false)
	w.WriteHeader(response.StatusCode)

	// Pieces confirmed downloaded — no pacing needed.
	if rangeVerified {
		return io.Copy(w, response.Body)
	}

	// Paced copy: read from upstream but never past the safe byte frontier.
	// When the player catches up to the download, pause until more data is
	// available.
	rangeStart := int64(0)
	if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
		if start, _, ok := parseByteRange(rangeHeader); ok {
			rangeStart = start
		}
	}

	buf := make([]byte, 64*1024)
	lastSafe := int64(0)
	stallDeadline := time.Now().Add(progressStallTimeout)

	for {
		select {
		case <-r.Context().Done():
			return bytesWritten, r.Context().Err()
		default:
		}

		safeBytes, _, done := safeBytesFn()
		if safeBytes > lastSafe {
			lastSafe = safeBytes
			stallDeadline = time.Now().Add(progressStallTimeout)
		}

		filePos := rangeStart + bytesWritten
		available := safeBytes - filePos

		if available <= 0 && !done {
			if time.Now().After(stallDeadline) {
				return bytesWritten, fmt.Errorf("download stalled: no progress for %s", progressStallTimeout)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		toRead := int64(len(buf))
		if !done && available < toRead {
			toRead = available
		}

		n, readErr := response.Body.Read(buf[:toRead])
		if n > 0 {
			nw, writeErr := w.Write(buf[:n])
			bytesWritten += int64(nw)
			if writeErr != nil {
				return bytesWritten, writeErr
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				readErr = nil
			}
			return bytesWritten, readErr
		}
	}
}
