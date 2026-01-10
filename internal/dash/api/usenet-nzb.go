package dash_api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	usenet_server "github.com/MunifTanjim/stremthru/internal/usenet/server"
)

type NzbSegmentResponse struct {
	Bytes     int64  `json:"bytes"`
	Number    int    `json:"number"`
	MessageId string `json:"message_id"`
}

type NzbFileResponse struct {
	Name     string               `json:"name"`
	Subject  string               `json:"subject"`
	Poster   string               `json:"poster"`
	Date     time.Time            `json:"date"`
	Groups   []string             `json:"groups"`
	Size     int64                `json:"size"`
	Segments []NzbSegmentResponse `json:"segments"`
}

type NzbParseResponse struct {
	Meta  map[string]string `json:"meta"`
	Size  int64             `json:"size"`
	Files []NzbFileResponse `json:"files"`
}

func toNzbParseResponse(parsed *nzb.NZB) NzbParseResponse {
	head := make(map[string]string)
	if parsed.Head != nil {
		for _, m := range parsed.Head.Meta {
			head[m.Type] = m.Value
		}
	}

	files := make([]NzbFileResponse, len(parsed.Files))
	for i, file := range parsed.Files {
		segments := make([]NzbSegmentResponse, len(file.Segments))
		for j, segment := range file.Segments {
			segments[j] = NzbSegmentResponse{
				Bytes:     segment.Bytes,
				Number:    segment.Number,
				MessageId: segment.MessageId,
			}
		}

		files[i] = NzbFileResponse{
			Name:     file.GetName(),
			Subject:  file.Subject,
			Poster:   file.Poster,
			Date:     time.Unix(file.Date, 0),
			Groups:   file.Groups,
			Size:     file.TotalSize(),
			Segments: segments,
		}
	}

	return NzbParseResponse{
		Meta:  head,
		Size:  parsed.TotalSize(),
		Files: files,
	}
}

func handleParseNzb(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		ErrorUnsupportedMediaType(r).Send(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		SendError(w, r, err)
		return
	}
	if r.MultipartForm.File == nil {
		ErrorBadRequest(r, "missing file").Send(w, r)
		return
	}
	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		ErrorBadRequest(r, "missing file").Send(w, r)
		return
	}
	if len(fileHeaders) > 1 {
		ErrorBadRequest(r, "multiple files provided").Send(w, r)
		return
	}
	fileHeader := fileHeaders[0]
	file, err := fileHeader.Open()
	if err != nil {
		SendError(w, r, err)
		return
	}
	defer file.Close()

	parsed, err := nzb.Parse(file)
	if err != nil {
		if parseErr, ok := err.(*nzb.ParseError); ok {
			ErrorBadRequest(r, parseErr.Error()).Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toNzbParseResponse(parsed))
}

type NzbDownloadRequest struct {
	Name     string               `json:"name"`
	Groups   []string             `json:"groups"`
	Segments []NzbSegmentResponse `json:"segments"`
}

func createUsenetPoolFromVault(ctx context.Context, log *logger.Logger) (*usenet_pool.Pool, error) {
	servers, err := usenet_server.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get usenet servers: %w", err)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no usenet servers configured")
	}

	providers := make([]usenet_pool.ProviderConfig, len(servers))
	for i, server := range servers {
		password, err := server.GetPassword()
		if err != nil {
			return nil, fmt.Errorf("failed to get password for server %s: %w", server.Name, err)
		}

		providers[i] = usenet_pool.ProviderConfig{
			PoolConfig: nntp.PoolConfig{
				ConnectionConfig: nntp.ConnectionConfig{
					Host:          server.Host,
					Port:          server.Port,
					Username:      server.Username,
					Password:      password,
					TLS:           server.TLS,
					TLSSkipVerify: server.TLSSkipVerify,
					Deadline:      time.Now().Add(30 * time.Second),
					DialTimeout:   15 * time.Second,
					KeepAliveTime: 60 * time.Second,
				},
				MaxSize: 10, // Max connections per provider
			},
			IsBackup: false, // All servers treated as primary for now
		}
	}

	poolConfig := &usenet_pool.Config{
		Log:                  log,
		Providers:            providers,
		RequiredCapabilities: []string{},
		MinConnections:       0,
	}

	pool, err := usenet_pool.NewPool(poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create usenet pool: %w", err)
	}

	return pool, nil
}

func handleDownloadNzb(w http.ResponseWriter, r *http.Request) {
	ctx := GetReqCtx(r)

	err := r.ParseForm()
	if err != nil {
		ErrorBadRequest(r, "invalid form data").WithCause(err).Send(w, r)
		return
	}

	nzbFile := &NzbDownloadRequest{}
	if err := json.Unmarshal([]byte(r.FormValue("nzb_file")), nzbFile); err != nil {
		SendError(w, r, err)
		return
	}

	if nzbFile.Name == "" {
		ErrorBadRequest(r, "missing name").Send(w, r)
		return
	}
	if len(nzbFile.Segments) == 0 {
		ErrorBadRequest(r, "missing segments").Send(w, r)
		return
	}

	ctx.Log.Trace("nzb download request", "name", nzbFile.Name, "segments", len(nzbFile.Segments))

	pool, err := createUsenetPoolFromVault(r.Context(), ctx.Log)
	if err != nil {
		if err.Error() == "no usenet servers configured" {
			ctx.Log.Warn("no usenet servers configured")
			ErrorBadRequest(r, "no usenet servers configured").Send(w, r)
			return
		}
		ctx.Log.Error("failed to create usenet pool", "error", err)
		SendError(w, r, err)
		return
	}
	defer pool.Close()

	nzbSegments := make([]nzb.Segment, len(nzbFile.Segments))
	var totalSize int64
	for i, seg := range nzbFile.Segments {
		nzbSegments[i] = nzb.Segment{
			Bytes:     seg.Bytes,
			Number:    seg.Number,
			MessageId: seg.MessageId,
		}
		totalSize += seg.Bytes
	}

	ctx.Log.Trace("starting usenet stream", "total_bytes", totalSize)

	stream, err := pool.StreamSegments(r.Context(), usenet_pool.StreamSegmentsConfig{
		Segments: nzbSegments,
		Groups:   nzbFile.Groups,
		Buffer:   5,
	})
	if err != nil {
		ctx.Log.Error("failed to create stream", "error", err)
		SendError(w, r, err)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", nzbFile.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stream.Size))

	w.WriteHeader(200)

	ctx.Log.Trace("starting io.Copy", "content_length", stream.Size)
	written, err := io.Copy(w, stream)
	ctx.Log.Trace("io.Copy completed", "bytes_written", written, "error", err)
	if err != nil {
		ctx.Log.Error("error streaming file", "error", err, "bytes_written", written)
		return
	}

	ctx.Log.Info("file download completed", "name", nzbFile.Name, "bytes", written)
}

func AddUsenetNzbEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/usenet/nzb/parse", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleParseNzb(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))

	router.HandleFunc("/usenet/nzb/download", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleDownloadNzb(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
