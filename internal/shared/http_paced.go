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
	// Take the first range only
	if idx := strings.Index(spec, ","); idx >= 0 {
		spec = spec[:idx]
	}
	if strings.HasPrefix(spec, "-") {
		// Suffix range (-N), pass through without modification
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

// SafeBytesFunc returns the number of contiguous bytes from the start of the
// file that are safe to serve, the total file size, and whether the file is
// fully available.
type SafeBytesFunc func() (safeBytes int64, fileSize int64, done bool)

// IsRangeAvailableFunc checks whether the byte range [start, end] within the
// file is fully downloaded at the piece level. Used for Range seeks (e.g.
// ffprobe reading container metadata at the end of a file) that fall outside
// the contiguous-from-start region.
type IsRangeAvailableFunc func(start, end int64) bool

const progressStallTimeout = 120 * time.Second

// ProxyQbitResponse is the entry point for qBittorrent progress-aware proxy
// streaming. It parses qBit query params, builds the progress callbacks, and
// delegates to proxyResponsePaced.
func ProxyQbitResponse(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType, user string, qbitHash string) (int64, error) {
	qbitFileIdx := 0
	if fidxStr := r.URL.Query().Get("qbit_fidx"); fidxStr != "" {
		if v, err := strconv.Atoi(fidxStr); err == nil {
			qbitFileIdx = v
		}
	}
	safeBytesFn := func() (int64, int64, bool) {
		safe, fileSize, done, err := GetQbitSafeBytes(user, qbitHash, qbitFileIdx)
		if err != nil {
			return 0, 0, true
		}
		return safe, fileSize, done
	}
	isRangeAvailFn := func(start, end int64) bool {
		avail, err := IsQbitFileRangeAvailable(user, qbitHash, qbitFileIdx, start, end)
		if err != nil {
			return false
		}
		return avail
	}
	return proxyResponsePaced(w, r, url, tunnelType, safeBytesFn, isRangeAvailFn)
}

// proxyResponsePaced is a progress-aware variant of ProxyResponse for
// streaming-while-downloading. It paces the proxy read to match the download
// progress so pre-allocated zero bytes are never forwarded, and handles Range
// seeks to non-contiguous regions via piece-level availability checks.
func proxyResponsePaced(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType, safeBytesFn SafeBytesFunc, isRangeAvailFn IsRangeAvailableFunc) (bytesWritten int64, err error) {
	request, err := http.NewRequestWithContext(r.Context(), r.Method, url, nil)
	if err != nil {
		e := ErrorInternalServerError(r, "failed to create request")
		e.Cause = err
		SendError(w, r, e)
		return
	}

	copyHeaders(r.Header, request.Header, true)

	// rangeVerified is set when we've confirmed (via piece-level check) that
	// the requested Range is fully downloaded. In that case we skip pacing
	// and proxy the response directly.
	rangeVerified := false

	// If the Range start is beyond the currently downloaded region, check
	// piece-level availability first, then fall back to waiting for the
	// sequential download to catch up.
	if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
		if start, end, ok := parseByteRange(rangeHeader); ok {
			safeBytes, fileSize, done := safeBytesFn()
			if start >= safeBytes && !done {
				// The Range start is beyond the sequential download frontier.
				// Check if the specific byte range is available at piece level
				// (e.g. last piece downloaded via firstLastPiecePrio).
				if isRangeAvailFn != nil {
					rangeEnd := end
					if rangeEnd < 0 {
						// Open-ended range: check up to the file size
						rangeEnd = fileSize - 1
					}
					if isRangeAvailFn(start, rangeEnd) {
						rangeVerified = true
					}
				}

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

	// Range was verified available at piece level — proxy directly.
	if rangeVerified {
		return io.Copy(w, response.Body)
	}

	// Progress-aware streaming: read from the upstream file server but pace
	// to match the qBittorrent download so we never forward pre-allocated
	// zero bytes.
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
		// Check for client disconnect.
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
