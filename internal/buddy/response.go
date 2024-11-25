package buddy

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	StatusCode int       `json:"status_code"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type Response[T interface{}] struct {
	Data  T              `json:"data,omitempty"`
	Error *ResponseError `json:"error,omitempty"`
}

type ResponseEnvelop interface {
	GetError() *ResponseError
}

func (r Response[any]) GetError() *ResponseError {
	return r.Error
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

	return v.GetError()
}
