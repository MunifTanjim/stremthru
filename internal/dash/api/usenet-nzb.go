package dash_api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/rs/xid"
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
			Name:     file.Name(),
			Subject:  file.Subject,
			Poster:   file.Poster,
			Date:     time.Unix(file.Date, 0),
			Groups:   file.Groups,
			Size:     file.Size(),
			Segments: segments,
		}
	}

	return NzbParseResponse{
		Meta:  head,
		Size:  parsed.TotalSize(),
		Files: files,
	}
}

func handleParseNZB(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		ErrorUnsupportedMediaType(r).Send(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, config.NewzNZBMaxFileSize)
	if err := r.ParseMultipartForm(util.ToBytes("10MB")); err != nil {
		SendError(w, r, err)
		return
	}
	if r.MultipartForm.File == nil {
		ErrorBadRequest(r).WithMessage("missing file").Send(w, r)
		return
	}
	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		ErrorBadRequest(r).WithMessage("missing file").Send(w, r)
		return
	}
	if len(fileHeaders) > 1 {
		ErrorBadRequest(r).WithMessage("multiple files provided").Send(w, r)
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
			ErrorBadRequest(r).WithMessage(parseErr.Error()).Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toNzbParseResponse(parsed))
}

type NZBContentFileResponse struct {
	Type       string                   `json:"type"`
	Name       string                   `json:"name"`
	Size       int64                    `json:"size"`
	Streamable bool                     `json:"streamable"`
	Errors     []string                 `json:"errors,omitempty"`
	Files      []NZBContentFileResponse `json:"files,omitempty"`
	Parts      []NZBContentFileResponse `json:"parts,omitempty"`
}

type NZBResponse struct {
	Id         string                   `json:"id"`
	Hash       string                   `json:"hash"`
	Name       string                   `json:"name"`
	Size       int64                    `json:"size"`
	FileCount  int                      `json:"file_count"`
	Password   string                   `json:"password"`
	URL        string                   `json:"url"`
	Files      []NZBContentFileResponse `json:"files"`
	Streamable bool                     `json:"streamable"`
	Cached     bool                     `json:"cached"`
	User       string                   `json:"user"`
	CreatedAt  string                   `json:"created_at"`
	UpdatedAt  string                   `json:"updated_at"`
}

func toNZBContentFileResponse(file usenet_pool.NZBContentFile) NZBContentFileResponse {
	resp := NZBContentFileResponse{
		Type:       string(file.Type),
		Name:       file.Name,
		Size:       file.Size,
		Streamable: file.Streamable,
		Errors:     file.Errors,
	}
	if len(file.Files) > 0 {
		resp.Files = make([]NZBContentFileResponse, len(file.Files))
		for i, f := range file.Files {
			resp.Files[i] = toNZBContentFileResponse(f)
		}
	}
	if len(file.Parts) > 0 {
		resp.Parts = make([]NZBContentFileResponse, len(file.Parts))
		for i, f := range file.Parts {
			resp.Parts[i] = toNZBContentFileResponse(f)
		}
	}
	return resp
}

func toNZBResponse(info *nzb_info.NZBInfo) NZBResponse {
	var contentFiles []NZBContentFileResponse
	if info.ContentFiles.Data != nil {
		contentFiles = make([]NZBContentFileResponse, len(info.ContentFiles.Data))
		for i, f := range info.ContentFiles.Data {
			contentFiles[i] = toNZBContentFileResponse(f)
		}
	}
	return NZBResponse{
		Id:         info.Id,
		Hash:       info.Hash,
		Name:       info.Name,
		Size:       info.Size,
		FileCount:  info.FileCount,
		Password:   info.Password,
		URL:        info.URL,
		Files:      contentFiles,
		Streamable: info.Streamable,
		Cached:     nzb_info.IsNZBFileCached(info.Hash),
		User:       info.User,
		CreatedAt:  info.CAt.Format(time.RFC3339),
		UpdatedAt:  info.UAt.Format(time.RFC3339),
	}
}

func handleGetNZBs(w http.ResponseWriter, r *http.Request) {
	items, err := nzb_info.GetAll()
	if err != nil {
		SendError(w, r, err)
		return
	}

	data := make([]NZBResponse, len(items))
	for i := range items {
		data[i] = toNZBResponse(&items[i])
	}

	SendData(w, r, 200, data)
}

func handleDeleteNZB(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := nzb_info.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if existing == nil {
		ErrorNotFound(r).WithMessage("nzb info not found").Send(w, r)
		return
	}

	if err := nzb_info.DeleteById(id); err != nil {
		SendError(w, r, err)
		return
	}

	nzb_info.DeleteNZBFile(existing.URL)

	SendData(w, r, 204, nil)
}

func handleGetNZBXML(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	info, err := nzb_info.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if info == nil {
		ErrorNotFound(r).WithMessage("nzb info not found").Send(w, r)
		return
	}

	nzbFile := nzb_info.GetCachedNZBFile(info.Hash)
	if nzbFile == nil {
		ErrorNotFound(r).WithMessage("nzb file not available").Send(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, nzbFile.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(nzbFile.Blob)))
	w.WriteHeader(http.StatusOK)
	w.Write(nzbFile.Blob)
}

func handleUploadNZB(w http.ResponseWriter, r *http.Request) {
	ctx := GetReqCtx(r)

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		ErrorUnsupportedMediaType(r).Send(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, config.NewzNZBMaxFileSize)
	if err := r.ParseMultipartForm(util.ToBytes("10MB")); err != nil {
		SendError(w, r, err)
		return
	}
	if r.MultipartForm.File == nil {
		ErrorBadRequest(r).WithMessage("missing file").Send(w, r)
		return
	}
	fileHeaders := r.MultipartForm.File["file"]
	if len(fileHeaders) == 0 {
		ErrorBadRequest(r).WithMessage("missing file").Send(w, r)
		return
	}
	if len(fileHeaders) > 1 {
		ErrorBadRequest(r).WithMessage("multiple files provided").Send(w, r)
		return
	}
	fileHeader := fileHeaders[0]
	file, err := fileHeader.Open()
	if err != nil {
		SendError(w, r, err)
		return
	}
	defer file.Close()

	blob, err := io.ReadAll(file)
	if err != nil {
		SendError(w, r, err)
		return
	}

	nzbDoc, err := nzb.ParseBytes(blob)
	if err != nil {
		if parseErr, ok := err.(*nzb.ParseError); ok {
			ErrorBadRequest(r).WithMessage(parseErr.Error()).Send(w, r)
			return
		}
		SendError(w, r, err)
		return
	}

	nzbId := xid.New().String()
	link := config.BaseURL.JoinPath("/v0/newznab/getnzb/", nzbId)
	linkQuery := link.Query()
	apikey := util.Base64Encode(ctx.Session.User + ":" + config.AdminPassword.GetPassword(ctx.Session.User))
	linkQuery.Set("apikey", apikey)
	link.RawQuery = linkQuery.Encode()

	filename := fileHeader.Filename
	if !strings.HasSuffix(filename, ".nzb") {
		filename += ".nzb"
	}

	nzbFile := nzb_info.NZBFile{
		Blob: blob,
		Name: filename,
		Link: link.String(),
		Mod:  time.Now(),
	}

	hash := nzb_info.HashNZBFileLink(nzbFile.Link)
	if err := nzb_info.CacheNZBFile(hash, nzbFile); err != nil {
		SendError(w, r, err)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = filename
	}

	if err := nzb_info.Upsert(&nzb_info.NZBInfo{
		Id:        nzbId,
		Hash:      hash,
		Name:      name,
		Size:      nzbDoc.TotalSize(),
		FileCount: nzbDoc.FileCount(),
		Password:  "",
		URL:       nzbFile.Link,
		User:      ctx.Session.User,
	}); err != nil {
		SendError(w, r, err)
		return
	}

	queueId, err := nzb_info.QueueJob(ctx.Session.User, name, nzbFile.Link, "", 0, "")
	if err != nil {
		SendError(w, r, err)
		return
	}

	queueItem, err := nzb_info.GetJobById(queueId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toNzbQueueItemResponse(queueItem))
}

func handleRequeueNZB(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	info, err := nzb_info.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if info == nil {
		ErrorNotFound(r).WithMessage("nzb info not found").Send(w, r)
		return
	}

	queueId, err := nzb_info.QueueJob(info.User, info.Name, info.URL, "", 0, info.Password)
	if err != nil {
		SendError(w, r, err)
		return
	}

	queueItem, err := nzb_info.GetJobById(queueId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, toNzbQueueItemResponse(queueItem))
}

func handleStreamNZBFile(w http.ResponseWriter, r *http.Request) {
	ctx := GetReqCtx(r)

	id := r.PathValue("id")

	info, err := nzb_info.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if info == nil {
		ErrorNotFound(r).WithMessage("nzb info not found").Send(w, r)
		return
	}

	path := r.PathValue("path")
	if path == "" {
		ErrorBadRequest(r).WithMessage("missing path").Send(w, r)
		return
	}

	nzbFile, err := nzb_info.FetchNZBFile(info.URL, info.Name, ctx.Log)
	if err != nil {
		SendError(w, r, err)
		return
	}

	nzbDoc, err := nzb.ParseBytes(nzbFile.Blob)
	if err != nil {
		SendError(w, r, err)
		return
	}

	pool, err := usenetmanager.GetPool()
	if err != nil {
		SendError(w, r, err)
		return
	}
	if pool == nil {
		ErrorBadRequest(r).WithMessage("no NNTP providers configured").Send(w, r)
		return
	}

	streamConfig := &usenet_pool.StreamConfig{
		Password: info.Password,
	}
	stream, err := pool.StreamByContentPath(r.Context(), nzbDoc, strings.Split(path, "::"), streamConfig)
	if err != nil {
		SendError(w, r, err)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, stream.Name, nzbFile.Mod, stream)
}

func AddUsenetNZBEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/usenet/nzb/parse", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleParseNZB(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/nzb/upload", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleUploadNZB(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))

	router.HandleFunc("/usenet/nzb", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNZBs(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/nzb/{id}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			handleDeleteNZB(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/nzb/{id}/requeue", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleRequeueNZB(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/nzb/{id}/xml", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetNZBXML(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
	router.HandleFunc("/usenet/nzb/{id}/download/{path...}", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleStreamNZBFile(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
