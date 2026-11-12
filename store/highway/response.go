package highway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

// Responses of the HighWay transfer API are plain JSON objects; errors look like
// {"error": "<code>", "message": "..."}. The API only adds fields over time.

type ResponseError struct {
	StatusCode int    `json:"-"`
	ErrorCode  string `json:"error"`
	Message    string `json:"message"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseContainer struct {
	Err     string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r *ResponseContainer) GetError(res *http.Response) error {
	if r.Err == "" && res.StatusCode < http.StatusBadRequest {
		return nil
	}
	msg := r.Message
	if msg == "" {
		msg = http.StatusText(res.StatusCode)
	}
	return &ResponseError{
		StatusCode: res.StatusCode,
		ErrorCode:  r.Err,
		Message:    msg,
	}
}

func (r *ResponseContainer) Unmarshal(res *http.Response, body []byte, v any) error {
	if res.StatusCode == http.StatusNoContent {
		return nil
	}

	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	case res.StatusCode >= http.StatusBadRequest:
		// e.g. an HTML error page from the proxy; GetError reports the status.
		return nil
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

type transferFile struct {
	Id          string  `json:"id"`
	Index       *int    `json:"index"`
	Path        string  `json:"path"`
	Size        int64   `json:"size"`
	Partial     bool    `json:"partial"`
	DownloadURL *string `json:"download_url"`
}

type transfer struct {
	Id       string         `json:"id"`
	Service  string         `json:"service"`
	Name     string         `json:"name"`
	State    string         `json:"state"` // queued, downloading, completed, error
	Progress float64        `json:"progress"`
	Size     int64          `json:"size"`
	Hash     *string        `json:"hash"`
	AddedAt  *int64         `json:"added_at"`
	Files    []transferFile `json:"files"`
}

type transferResponse struct {
	ResponseContainer
	transfer
}

type listResponse struct {
	ResponseContainer
	Transfers []transfer `json:"transfers"`
	Total     int        `json:"total"`
}

type addResponse struct {
	ResponseContainer
	Id       string `json:"id"`
	Service  string `json:"service"`
	State    string `json:"state"`
	Hash     string `json:"hash"`      // only for nzb_url: md5 of the NZB content
	LinkHash string `json:"link_hash"` // only for nzb_url: md5 of the link, as in util.HashNZBFileLink
}

type fileResponse struct {
	ResponseContainer
	Id    string       `json:"id"`
	State string       `json:"state"`
	File  transferFile `json:"file"`
}

type checkItem struct {
	Known bool    `json:"known"`
	Ready bool    `json:"ready"`
	Id    *string `json:"id"`
	Name  *string `json:"name"`
	Size  int64   `json:"size"`
}

type checkResponse struct {
	ResponseContainer
	Results map[string]checkItem `json:"results"`
}

type removeResponse struct {
	ResponseContainer
	Removed bool   `json:"removed"`
	Reason  string `json:"reason"` // only when removed is false: not_found or not_owner
}

type statsResponse struct {
	ResponseContainer
	Username string `json:"username"`
	UserId   int64  `json:"user_id"`
	Plan     string `json:"plan"`
	Premium  bool   `json:"premium"`
}
