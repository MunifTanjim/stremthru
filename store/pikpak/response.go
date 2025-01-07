package pikpak

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
)

var ErrByCode = map[int]string{
	3:    "invalid_argument",
	5:    "not_found",       // 'file_not_found'
	9:    "captcha_invalid", // 'file_in_recycle_bin'
	16:   "unauthenticated",
	4002: "captcha_invalid",
	4022: "invalid_account_or_password",
}

type ErrorDetail struct {
	Type         string `json:"@type"`
	StackEntries []any  `json:"stack_entries"`
	Detail       string `json:"detail"`
}

type ResponseContainer struct {
	Err        string        `json:"error"`
	ErrCode    int           `json:"error_code"`
	ErrDesc    string        `json:"error_description"`
	ErrURL     string        `json:"error_url"`
	ErrDetails []ErrorDetail `json:"error_details"`
}

func (e *ResponseContainer) Error() string {
	ret, _ := json.Marshal(e)
	return string(ret)
}

type ResponseEnvelop interface {
	HasError() bool
	GetError() *ResponseContainer
}

func (r *ResponseContainer) HasError() bool {
	return r.Err != ""
}

func (r *ResponseContainer) GetError() *ResponseContainer {
	if r.HasError() {
		return r
	}
	return nil
}

func extractResponseError(statusCode int, body []byte, v ResponseEnvelop) error {
	if v.HasError() {
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
