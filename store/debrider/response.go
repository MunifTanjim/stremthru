package debrider

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseContainer struct {
	Message string `json:"message,omitempty"`
}

func (e *ResponseContainer) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

func (r *ResponseContainer) GetError(res *http.Response) error {
	if res.StatusCode >= http.StatusBadRequest {
		return r
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

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
}

func newAPIResponse[T any](res *http.Response, data T) APIResponse[T] {
	apiResponse := APIResponse[T]{
		StatusCode: 503,
		Data:       data,
	}
	if res != nil {
		apiResponse.Header = res.Header
		apiResponse.StatusCode = res.StatusCode
	}
	return apiResponse
}
