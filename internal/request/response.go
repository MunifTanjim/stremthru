package request

import (
	"io"
	"net/http"
)

type APIResponse[T any] struct {
	Header     http.Header
	StatusCode int
	Data       T
}

func NewAPIResponse[T any](res *http.Response, data T) APIResponse[T] {
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

type ResponseContainer interface {
	GetError(res *http.Response) error
	Unmarshal(res *http.Response, body []byte, v any) error
}

func ProcessResponseBody(res *http.Response, err error, v ResponseContainer) error {
	if err != nil {
		return err
	}

	if res.Request.Method == http.MethodHead {
		return nil
	}

	body, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return err
	}

	err = v.Unmarshal(res, body, v)
	if err != nil {
		return err
	}

	return v.GetError(res)
}
