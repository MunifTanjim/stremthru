package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func UnmarshalJSON(statusCode int, body []byte, v interface{}) error {
	if statusCode == 204 && len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	err := json.Unmarshal(body, v)
	if err == nil {
		return nil
	}

	bodySample := string(body)
	if len(bodySample) > 500 {
		bodySample = bodySample[0:500] + " ..."
	}

	bodySample = strings.Replace(bodySample, "\n", "\\n", -1)

	return fmt.Errorf(
		"Couldn't deserialize JSON (response status: %v, body sample: '%s'): %v",
		statusCode, bodySample, err,
	)
}

type MagnetLink struct {
	Hash     string // xt - exact topic
	Link     string
	Name     string   // dn - display name
	Trackers []string // tr - address tracker
}

func ParseMagnetLink(value string) (MagnetLink, error) {
	magnet := MagnetLink{}
	if !strings.HasPrefix(value, "magnet:") {
		magnet.Hash = strings.ToLower(value)
		magnet.Link = "magnet:?xt=urn:btih:" + magnet.Hash
		return magnet, nil
	}

	u, err := url.Parse(value)
	if err != nil {
		return magnet, err
	}
	params := u.Query()
	xt := params.Get("xt")

	if !strings.HasPrefix(xt, "urn:btih:") {
		return magnet, errors.New("invalid magnet")
	}

	magnet.Hash = strings.ToLower(strings.TrimPrefix(xt, "urn:btih:"))
	magnet.Name = params.Get("dn")
	if params.Has("tr") {
		magnet.Trackers = params["tr"]
		params.Del("tr")
	}
	magnet.Link = "magnet:?xt=" + "urn:btih:" + magnet.Hash
	if magnet.Name != "" {
		magnet.Link = magnet.Link + "&dn=" + magnet.Name
	}
	return magnet, nil
}
