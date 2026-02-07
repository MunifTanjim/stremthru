package core

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func UnmarshalJSON(statusCode int, body []byte, v any) error {
	if statusCode == 204 && len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	err := json.Unmarshal(body, v)
	if err == nil {
		return nil
	}

	bodySample := string(body)
	if len(bodySample) > 1000 {
		bodySample = bodySample[0:1000] + " ..."
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
	RawLink  string
}

func NormalizeMagnetHash(hash string) string {
	switch len(hash) {
	case 40:
		return strings.ToLower(hash)
	case 32:
		if decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(hash)); err != nil {
			return ""
		} else {
			return strings.ToLower(hex.EncodeToString(decoded))
		}
	default:
		return ""
	}
}

func ParseMagnetLink(value string) (MagnetLink, error) {
	magnet := MagnetLink{}
	if !strings.HasPrefix(value, "magnet:") {
		magnet.Hash = NormalizeMagnetHash(value)
		magnet.Link = "magnet:?xt=urn:btih:" + magnet.Hash
		magnet.RawLink = magnet.Link
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

	magnet.Hash = NormalizeMagnetHash(strings.TrimPrefix(xt, "urn:btih:"))
	magnet.Name = params.Get("dn")
	if params.Has("tr") {
		magnet.Trackers = params["tr"]
		params.Del("tr")
	}
	magnet.Link = "magnet:?xt=" + "urn:btih:" + magnet.Hash
	if magnet.Name != "" {
		magnet.Link = magnet.Link + "&dn=" + magnet.Name
	}
	magnet.RawLink = value
	return magnet, nil
}

var HasVideoExtension = func() func(filename string) bool {
	videoExtensions := map[string]bool{
		".3g2":  true,
		".3gp":  true,
		".amv":  true,
		".asf":  true,
		".avi":  true,
		".drc":  true,
		".f4a":  true,
		".f4b":  true,
		".f4p":  true,
		".f4v":  true,
		".flv":  true,
		".gif":  true,
		".gifv": true,
		".m2ts": true,
		".m2v":  true,
		".m4p":  true,
		".m4v":  true,
		".mk3d": true,
		".mkv":  true,
		".mng":  true,
		".mov":  true,
		".mp2":  true,
		".mp4":  true,
		".mpe":  true,
		".mpeg": true,
		".mpg":  true,
		".mpv":  true,
		".mxf":  true,
		".nsv":  true,
		".ogg":  true,
		".ogm":  true,
		".ogv":  true,
		".qt":   true,
		".rm":   true,
		".rmvb": true,
		".roq":  true,
		".svi":  true,
		".ts":   true,
		".webm": true,
		".wmv":  true,
		".yuv":  true,
	}

	return func(filename string) bool {
		_, found := videoExtensions[strings.ToLower(filepath.Ext(filename))]
		return found
	}
}()
