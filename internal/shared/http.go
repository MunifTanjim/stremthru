package shared

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
)

func IsMethod(r *http.Request, method string) bool {
	return r.Method == method
}

func GetQueryInt(queryParams url.Values, name string, defaultValue int) (int, error) {
	if qVal, ok := queryParams[name]; ok {
		v := qVal[0]
		if v == "" {
			return defaultValue, nil
		}

		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, errors.New("invalid " + name)
		}
		return val, nil
	}
	return defaultValue, nil
}

func ReadRequestBodyJSON[T any](r *http.Request, payload T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ErrorUnsupportedMediaType(r)
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err == nil {
		return err
	}
	if err == io.EOF {
		return ErrorBadRequest(r, "missing body")
	}
	error := core.NewAPIError("failed to decode body")
	error.Cause = err
	return error
}

type response struct {
	Data  any   `json:"data,omitempty"`
	Error error `json:"error,omitempty"`
}

func (res response) send(w http.ResponseWriter, r *http.Request, statusCode int) {
	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(res); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func SendError(w http.ResponseWriter, r *http.Request, err error) {
	var e core.StremThruError
	if sterr, ok := err.(core.StremThruError); ok {
		e = sterr
	} else if aerr, ok := err.(*server.APIError); ok {
		e = &core.Error{
			RequestId:  aerr.RequestId,
			Type:       core.ErrorTypeAPI,
			Code:       aerr.Code,
			Msg:        aerr.Message,
			Method:     aerr.Method,
			Path:       aerr.Path,
			StatusCode: aerr.StatusCode,
			Cause:      aerr.Cause,
		}
	} else {
		e = &core.Error{Cause: err}
	}
	e.Pack(r)

	ctx := server.GetReqCtx(r)
	ctx.Error = err

	res := &response{}
	res.Error = e.GetError()

	res.send(w, r, e.GetStatusCode())
}

func SendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any, err error) {
	if err != nil {
		SendError(w, r, err)
		return
	}

	res := &response{}
	res.Data = data

	res.send(w, r, statusCode)
}

func SendHTML(w http.ResponseWriter, statusCode int, data bytes.Buffer) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	data.WriteTo(w)
}

func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func SendXML(w http.ResponseWriter, r *http.Request, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)
	encoder := xml.NewEncoder(w)
	encoder.Indent("", " ")
	w.Write([]byte(xml.Header))
	if err := encoder.Encode(v); err != nil {
		core.LogError(r, "failed to encode xml", err)
	}
}

func copyHeaders(src http.Header, dest http.Header, stripIpHeaders bool) {
	for key, values := range src {
		if stripIpHeaders {
			switch strings.ToLower(key) {
			case "x-client-ip", "x-forwarded-for", "cf-connecting-ip", "do-connecting-ip", "fastly-client-ip", "true-client-ip", "x-real-ip", "x-cluster-client-ip", "x-forwarded", "forwarded-for", "forwarded", "x-appengine-user-ip", "cf-pseudo-ipv4":
				continue
			}
		}
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

var proxyHttpClientByTunnelType = map[config.TunnelType]*http.Client{
	config.TUNNEL_TYPE_NONE: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_NONE)
		return &http.Client{
			Transport: transport,
		}
	}(),
	config.TUNNEL_TYPE_AUTO: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_AUTO)
		return &http.Client{
			Transport: transport,
		}
	}(),
	config.TUNNEL_TYPE_FORCED: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_FORCED)
		return &http.Client{
			Transport: transport,
		}
	}(),
}

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
// file that are safe to serve, and whether the file is fully available.
type SafeBytesFunc func() (safeBytes int64, done bool)

const progressStallTimeout = 120 * time.Second

func ProxyResponse(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType, safeBytesFn SafeBytesFunc) (bytesWritten int64, err error) {
	request, err := http.NewRequest(r.Method, url, nil)
	if err != nil {
		e := ErrorInternalServerError(r, "failed to create request")
		e.Cause = err
		SendError(w, r, e)
		return
	}

	copyHeaders(r.Header, request.Header, true)

	// For progress-aware streaming: if the Range start is beyond the
	// currently downloaded region, wait for the download to catch up.
	if safeBytesFn != nil {
		if rangeHeader := request.Header.Get("Range"); rangeHeader != "" {
			if start, _, ok := parseByteRange(rangeHeader); ok {
				safeBytes, done := safeBytesFn()
				if start >= safeBytes && !done {
					deadline := time.Now().Add(progressStallTimeout)
					for start >= safeBytes && !done && time.Now().Before(deadline) {
						time.Sleep(2 * time.Second)
						safeBytes, done = safeBytesFn()
					}
					if start >= safeBytes && !done {
						w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(safeBytes, 10))
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

	// No progress tracking — normal proxy.
	if safeBytesFn == nil {
		return io.Copy(w, response.Body)
	}

	// Progress-aware streaming: read from nginx but pace to match the
	// qBittorrent download so we never forward pre-allocated zero bytes.
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

		safeBytes, done := safeBytesFn()
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

func extractRequestScheme(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")

	if scheme == "" {
		scheme = r.URL.Scheme
	}

	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	return scheme
}

func extractRequestHost(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		host = r.Host
	}

	return host
}

func GetReversedHostname(r *http.Request) string {
	hostname := extractRequestHost(r)
	hostname, _, _ = strings.Cut(hostname, ":")
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return "local.stremthru"
	}
	parts := strings.Split(hostname, ".")
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func ExtractRequestBaseURL(r *http.Request) *url.URL {
	return &url.URL{
		Scheme: extractRequestScheme(r),
		Host:   extractRequestHost(r),
	}
}

func GetClientIP(r *http.Request, ctx *storecontext.Context) string {
	if !ctx.IsProxyAuthorized {
		return core.GetClientIP(r)
	}
	if ctx.Store != nil && config.StoreTunnel.GetTypeForAPI(string(ctx.Store.GetName())) == config.TUNNEL_TYPE_NONE {
		return config.IP.GetMachineIP()
	}
	return ""
}
