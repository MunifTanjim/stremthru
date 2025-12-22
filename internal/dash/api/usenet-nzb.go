package dash_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
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
}
