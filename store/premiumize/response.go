package premiumize

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

type ResponseStatus string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusError   ResponseStatus = "error"
)

type ResponseContainer struct {
	Status  ResponseStatus `json:"status"`
	Message string         `json:"message,omitempty"`
}

func (e *ResponseContainer) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseEnvelop interface {
	GetStatus() ResponseStatus
	GetError() *ResponseContainer
}

func (r *ResponseContainer) GetStatus() ResponseStatus {
	return r.Status
}

func (r *ResponseContainer) GetError() *ResponseContainer {
	if r.GetStatus() == ResponseStatusError {
		return r
	}
	return nil
}

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
}

func newAPIResponse[T any](res *http.Response, data T) APIResponse[T] {
	return APIResponse[T]{
		Header:     res.Header,
		StatusCode: res.StatusCode,
		Data:       data,
	}
}

func extractResponseError(statusCode int, body []byte, v ResponseEnvelop) error {
	if v.GetStatus() == ResponseStatusError {
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
