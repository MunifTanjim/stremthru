package bitmagnet

import (
	"errors"
	"net/http"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
)

type BitmagnetStatus struct {
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"info"`
	Status string `json:"status"`
}

func (bs *BitmagnetStatus) Unmarshal(res *http.Response, body []byte, v any) error {
	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return core.UnmarshalJSON(res.StatusCode, body, v)
	}
	return errors.New("unexpected content type: " + contentType)
}

func (bs *BitmagnetStatus) GetError(res *http.Response) error {
	if bs.Status == "up" {
		return nil
	}
	return errors.New("bitmagnet status is not up: " + bs.Status)
}

func GetVersion() (string, error) {
	res, err := config.DefaultHTTPClient.Get(config.Integration.Bitmagnet.BaseURL.JoinPath("/status").String())
	if err != nil {
		return "", err
	}
	response := BitmagnetStatus{}
	if err = request.ProcessResponseBody(res, err, &response); err != nil {
		return "", err
	}
	return response.Info.Version, nil
}
