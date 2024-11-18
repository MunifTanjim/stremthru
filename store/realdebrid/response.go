package realdebrid

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseError struct {
	Err     string    `json:"error,omitempty"`
	ErrCode ErrorCode `json:"error_code,omitempty"`
}

func (e *ResponseError) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseContainer interface {
	GetError() *ResponseError
}

func (r *ResponseError) GetError() *ResponseError {
	if r == nil || r.Err == "" || r.ErrCode == 0 {
		return nil
	}
	return r
}

type APIResponse[T any] struct {
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

func extractResponseError(statusCode int, body []byte, v ResponseContainer) error {
	if err := v.GetError(); err != nil {
		return err
	}
	if statusCode >= http.StatusBadRequest {
		return errors.New(string(body))
	}
	return nil
}

func processResponseBody(res *http.Response, err error, v ResponseContainer) error {
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	log.Println("res body: ", string(body))

	if err != nil {
		return err
	}

	err = core.UnmarshalJSON(res.StatusCode, body, v)
	if err != nil {
		return err
	}

	return extractResponseError(res.StatusCode, body, v)
}
