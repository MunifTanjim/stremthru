package seedr

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseContainer struct {
	Err          string `json:"error,omitempty"`
	ErrDesc      string `json:"error_description,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	ReasonPhrase string `json:"reason_phrase,omitempty"` // not_enough_space_added_to_wishlist
}

func (e *ResponseContainer) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseEnvelop interface {
	GetError(res *http.Response) error
	Unmarshal(res *http.Response, body []byte, v any) error
}

func (r *ResponseContainer) HasError() bool {
	return r.Err != "" || (r.StatusCode >= 400 && r.ReasonPhrase != "")
}

func (r *ResponseContainer) GetError(res *http.Response) error {
	if r.HasError() {
		return r
	}
	if res.StatusCode >= http.StatusBadRequest {
		return errors.New("unexpected status code: " + http.StatusText(res.StatusCode))
	}
	return nil
}

func (r *ResponseContainer) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}
