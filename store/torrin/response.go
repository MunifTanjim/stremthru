package torrin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseError struct {
	Code    int    `json:"code"` // http status code
	Message string `json:"message"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseContainer struct {
	Error *ResponseError `json:"error,omitempty"`
}

func (r *ResponseContainer) GetError(res *http.Response) error {
	if r.Error != nil {
		if r.Error.Code == 0 {
			r.Error.Code = res.StatusCode
		}
		return r.Error
	}
	if res.StatusCode >= http.StatusBadRequest {
		return &ResponseError{
			Code:    res.StatusCode,
			Message: "unexpected error",
		}
	}
	return nil
}

func (r *ResponseContainer) Unmarshal(res *http.Response, body []byte, v any) error {
	if res.StatusCode == 204 {
		return nil
	}

	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return core.UnmarshalJSON(res.StatusCode, body, v)
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

type Response[T any] struct {
	ResponseContainer
	Data T `json:"data,omitempty"`
}
