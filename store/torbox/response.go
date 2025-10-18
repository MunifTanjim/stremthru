package torbox

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseStatus string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusError   ResponseStatus = "error"
)

type ResponseError struct {
	Detail string    `json:"detail"`
	Err    ErrorCode `json:"error"`
	Data   string    `json:"data"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type Response[T any] struct {
	response[T]
	errData any `json:"-"`
}

type response[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data,omitempty"`
	Detail  string    `json:"detail"`
	Error   ErrorCode `json:"error,omitempty"`
}

type ResponseEnvelop interface {
	GetError(res *http.Response) error
	Unmarshal(res *http.Response, body []byte, v any) error
}

func (r Response[any]) IsSuccess() bool {
	return r.Success && r.Error == ""
}

func (r Response[any]) GetError(res *http.Response) error {
	if r.IsSuccess() {
		return nil
	}
	err := ResponseError{
		Err:    r.Error,
		Detail: r.Detail,
	}
	if data, ok := r.errData.(string); ok {
		err.Data = data
	}
	return &err
}

func (r *Response[T]) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return r.unmarshalJSON(res.StatusCode, body)
	case strings.Contains(contentType, "text/plain") && res.StatusCode >= 400:
		r.Error = ErrorCodeUnknownError
		r.Detail = string(body)
		return nil
	default:
		return errors.New("unexpected content type: " + contentType)
	}
}

func (r *Response[T]) unmarshalJSON(statusCode int, body []byte) error {
	resp := response[T]{}
	respErr := core.UnmarshalJSON(statusCode, body, &resp)
	if respErr == nil {
		r.response = resp
		return nil
	}
	fallbackResp := response[any]{}
	if err := core.UnmarshalJSON(statusCode, body, &fallbackResp); err != nil {
		return err
	}
	if fallbackResp.Success {
		return respErr
	}
	r.Error = fallbackResp.Error
	r.Detail = fallbackResp.Detail
	r.errData = fallbackResp.Data
	return nil
}

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
	Detail     string
}

func newAPIResponse[T any](res *http.Response, data T, detail string) APIResponse[T] {
	apiResponse := APIResponse[T]{
		StatusCode: 503,
		Data:       data,
		Detail:     detail,
	}
	if res != nil {
		apiResponse.Header = res.Header
		apiResponse.StatusCode = res.StatusCode
	}
	return apiResponse
}
