package debridlink

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseError struct {
	Err     ErrorCode `json:"error,omitempty"`
	ErrId   string    `json:"error_id,omitempty"`
	ErrDesc string    `json:"error_description,omitempty"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponsePagination struct {
	Page     int `json:"page"`
	Pages    int `json:"pages"`
	Next     int `json:"next"`
	Previous int `json:"previous"`
}

type Response[T interface{}] struct {
	*ResponseError
	Success bool `json:"success"`
	Value   T    `json:"value,omitempty"`
}

type PaginatedResponse[T interface{}] struct {
	Response[[]T]
	Pagination ResponsePagination `json:"pagination"`
}

type ResponseEnvelop interface {
	IsSuccess() bool
	GetError() *ResponseError
}

func (r Response[any]) IsSuccess() bool {
	return r.Success
}

func (r Response[any]) GetError() *ResponseError {
	if r.IsSuccess() {
		return nil
	}
	return r.ResponseError
}

type APIResponse[T interface{}] struct {
	Header     http.Header
	StatusCode int
	Data       T
}

func newAPIResponse[T interface{}](res *http.Response, data T) APIResponse[T] {
	return APIResponse[T]{
		Header:     res.Header,
		StatusCode: res.StatusCode,
		Data:       data,
	}
}

func extractResponseError(statusCode int, body []byte, v ResponseEnvelop) error {
	if !v.IsSuccess() {
		return v.GetError()
	}
	if statusCode >= http.StatusBadRequest {
		return errors.New(string(body))
	}
	return nil
}

func processResponseBody(res *http.Response, err error, v ResponseEnvelop) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return extractResponseError(res.StatusCode, body, v)
}
