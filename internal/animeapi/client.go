package animeapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
)

type textResponse string

func (tr *textResponse) Unmarshal(res *http.Response, body []byte, v any) error {
	*tr = textResponse(body)
	return nil
}
func (tr *textResponse) GetError(res *http.Response) error {
	if res.StatusCode >= http.StatusBadRequest {
		msg := string(*tr)
		if msg == "" {
			msg = res.Status
		}
		return errors.New(msg)
	}
	return nil
}

var httpClient = config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)

var baseUrl = func() *url.URL {
	baseUrl, err := url.Parse("https://animeapi.my.id")
	if err != nil {
		panic(err)
	}
	return baseUrl
}()

func getLastUpdated() (time.Time, error) {
	params := request.Ctx{}
	req, err := params.NewRequest(baseUrl, "GET", "/updated", func(header *http.Header, params request.Context) {}, func(query *url.Values, params request.Context) {})
	if err != nil {
		return time.Time{}, err
	}
	var response textResponse
	res, err := httpClient.Do(req)
	err = request.ProcessResponseBody(res, err, &response)
	if err != nil {
		return time.Time{}, err
	}
	timestring := strings.TrimPrefix(string(response), "Updated on ")
	updatedAt, err := time.Parse("01/02/2006 15:04:05 MST", timestring)
	return updatedAt, err
}
