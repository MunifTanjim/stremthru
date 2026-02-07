package config

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

var NewzNZBCacheSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_NZB_CACHE_SIZE"))
var NewzNZBCacheTTL = mustParseDuration("newz nzb cache ttl", getEnv("STREMTHRU_NEWZ_NZB_CACHE_TTL"), 6*time.Hour)
var NewzNZBMaxFileSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_NZB_MAX_FILE_SIZE"))

var NewzSegmentCacheSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE"))
var NewzStreamBufferSize = util.ToBytes(getEnv("STREMTHRU_NEWZ_STREAM_BUFFER_SIZE"))
var NewzMaxConnectionPerStream = util.MustParseInt(getEnv("STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM"))

type NZBLinkMode string

var (
	NZBLinkModeProxy    NZBLinkMode = "proxy"
	NZBLinkModeRedirect NZBLinkMode = "redirect"
)

type nzbLinkModeMap map[string]NZBLinkMode

func (m nzbLinkModeMap) Proxy(hostname string) bool {
	if mode, ok := m[hostname]; ok {
		return mode == NZBLinkModeProxy
	}
	if mode, ok := m["*"]; ok {
		return mode == NZBLinkModeProxy
	}
	return false
}

func (m nzbLinkModeMap) Redirect(hostname string) bool {
	if mode, ok := m[hostname]; ok {
		return mode == NZBLinkModeRedirect
	}
	if mode, ok := m["*"]; ok {
		return mode == NZBLinkModeRedirect
	}
	return false
}

var NewzNZBLinkMode = func() nzbLinkModeMap {
	var newzNZBLinkModeMap = nzbLinkModeMap{
		"*": NZBLinkModeProxy,
	}
	for _, entry := range strings.FieldsFunc(getEnv("STREMTHRU_NEWZ_NZB_LINK_MODE"), func(c rune) bool {
		return c == ','
	}) {
		if hostname, mode, ok := strings.Cut(entry, ":"); ok && hostname != "" && mode != "" {
			newzNZBLinkModeMap[hostname] = NZBLinkMode(mode)
		}
	}
	return newzNZBLinkModeMap
}()

type NewzIndexerRequestQueryType string

const (
	NewzIndexerRequestQueryTypeAny   NewzIndexerRequestQueryType = "*"
	NewzIndexerRequestQueryTypeMovie NewzIndexerRequestQueryType = "movie"
	NewzIndexerRequestQueryTypeTV    NewzIndexerRequestQueryType = "tv"
)

type newzIndexerRequestHeaderByType map[NewzIndexerRequestQueryType]http.Header

type newzIndexerRequestHeaderMap struct {
	Query newzIndexerRequestHeaderByType
	Grab  http.Header
}

func (mbt newzIndexerRequestHeaderByType) Get(queryType NewzIndexerRequestQueryType) http.Header {
	if h, ok := mbt[queryType]; ok && len(h) > 0 {
		return h.Clone()
	}
	if queryType != "*" {
		return mbt.Get("*").Clone()
	}
	return nil
}

type newzConfig struct {
	IndexerRequestHeader newzIndexerRequestHeaderMap
}

func parseNewzIndexerRequestHeader(queryHeaderBlob, grabHeaderBlob string) newzIndexerRequestHeaderMap {
	chromeHeaderBlob := util.MustDecodeBase64("VXNlci1BZ2VudDogTW96aWxsYS81LjAgKE1hY2ludG9zaDsgSW50ZWwgTWFjIE9TIFggMTBfMTVfNykgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzE0My4wLjAuMCBTYWZhcmkvNTM3LjM2CkFjY2VwdDogdGV4dC9odG1sLGFwcGxpY2F0aW9uL3hodG1sK3htbCxhcHBsaWNhdGlvbi94bWw7cT0wLjksaW1hZ2UvYXZpZixpbWFnZS93ZWJwLGltYWdlL2FwbmcsKi8qO3E9MC44LGFwcGxpY2F0aW9uL3NpZ25lZC1leGNoYW5nZTt2PWIzO3E9MC43CkFjY2VwdC1MYW5ndWFnZTogZW4tVVMsZW47cT0wLjkKUHJpb3JpdHk6IHU9MCwgaQpTZWMtQ2gtVWE6ICJHb29nbGUgQ2hyb21lIjt2PSIxNDMiLCAiQ2hyb21pdW0iO3Y9IjE0MyIsICJOb3QgQShCcmFuZCI7dj0iMjQiClNlYy1DaC1VYS1Nb2JpbGU6ID8wClNlYy1DaC1VYS1QbGF0Zm9ybTogIm1hY09TIgpTZWMtRmV0Y2gtRGVzdDogZG9jdW1lbnQKU2VjLUZldGNoLU1vZGU6IG5hdmlnYXRlClNlYy1GZXRjaC1TaXRlOiBzYW1lLXNpdGUKU2VjLUZldGNoLVVzZXI6ID8xClVwZ3JhZGUtSW5zZWN1cmUtUmVxdWVzdHM6IDE=")
	presetQueryHeaderBlob := map[string]string{
		"chrome":   chromeHeaderBlob,
		"prowlarr": util.MustDecodeBase64("QWNjZXB0OiBhcHBsaWNhdGlvbi9yc3MreG1sLCB0ZXh0L3Jzcyt4bWwsIGFwcGxpY2F0aW9uL3htbCwgdGV4dC94bWwKVXNlci1BZ2VudDogUHJvd2xhcnIvMi4zLjAuNTIzNiAoYWxwaW5lIDMuMjMuMyk="),
		"radarr":   util.MustDecodeBase64("QWNjZXB0OiBhcHBsaWNhdGlvbi9yc3MreG1sLCB0ZXh0L3Jzcyt4bWwsIGFwcGxpY2F0aW9uL3htbCwgdGV4dC94bWwKVXNlci1BZ2VudDogUmFkYXJyLzYuMC41LjEwMjkxIChhbHBpbmUgMy4yMy4zKQ=="),
		"sonarr":   util.MustDecodeBase64("QWNjZXB0OiBhcHBsaWNhdGlvbi9yc3MreG1sLCB0ZXh0L3Jzcyt4bWwsIGFwcGxpY2F0aW9uL3htbCwgdGV4dC94bWwKVXNlci1BZ2VudDogU29uYXJyLzQuMC4xNi4yOTQ0IChhbHBpbmUgMy4yMy4zKQ=="),
	}
	presetGrabHeaderBlob := map[string]string{
		"chrome":  chromeHeaderBlob,
		"nzbget":  util.MustDecodeBase64("QWNjZXB0OiAqLyoKVXNlci1BZ2VudDogbnpiZ2V0LzIxLjE="),
		"sabnzbd": util.MustDecodeBase64("VXNlci1BZ2VudDogU0FCbnpiZC80LjUuNQ=="),
	}

	indexerRequestHeader := newzIndexerRequestHeaderMap{
		Query: newzIndexerRequestHeaderByType{
			"*": http.Header{},
		},
		Grab: http.Header{},
	}

	parseNewzIndexerRequestHeaderBlob := func(blob string, presets map[string]string, supportQueryType bool, setHeader func(queryType NewzIndexerRequestQueryType, key string, value string)) {
		currQueryType := NewzIndexerRequestQueryTypeAny
		for line := range strings.SplitSeq(blob, "\n") {
			line = strings.TrimSpace(line)

			if line == "" {
				if supportQueryType {
					currQueryType = ""
				}
				continue
			}

			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				if !supportQueryType {
					panic("newz header: query type not supported")
				}
				queryTypeVal := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
				switch queryType := NewzIndexerRequestQueryType(queryTypeVal); queryType {
				case NewzIndexerRequestQueryTypeTV, NewzIndexerRequestQueryTypeMovie, "*":
					currQueryType = queryType
				default:
					panic("newz header: invalid query type: " + currQueryType)
				}
				continue
			}

			if supportQueryType && currQueryType == "" {
				panic("newz header: missing query type after empty line")
			}

			if strings.HasPrefix(line, ":") && strings.HasSuffix(line, ":") {
				presetName := strings.TrimSuffix(strings.TrimPrefix(line, ":"), ":")
				if presetBlob, ok := presets[presetName]; ok {
					for header := range strings.SplitSeq(presetBlob, "\n") {
						if k, v, ok := strings.Cut(header, ": "); ok {
							setHeader(currQueryType, k, v)
						}
					}
				} else {
					panic("invalid newz header preset: " + presetName)
				}
				continue
			}

			if k, v, ok := strings.Cut(line, ": "); ok {
				setHeader(currQueryType, k, v)
			}
		}
	}

	parseNewzIndexerRequestHeaderBlob(queryHeaderBlob, presetQueryHeaderBlob, true, func(queryType NewzIndexerRequestQueryType, key, value string) {
		if _, ok := indexerRequestHeader.Query[queryType]; !ok {
			indexerRequestHeader.Query[queryType] = http.Header{}
		}
		indexerRequestHeader.Query[queryType].Set(key, value)
	})

	parseNewzIndexerRequestHeaderBlob(grabHeaderBlob, presetGrabHeaderBlob, false, func(_ NewzIndexerRequestQueryType, key, value string) {
		indexerRequestHeader.Grab.Set(key, value)
	})
	return indexerRequestHeader
}

var Newz = func() newzConfig {
	newz := newzConfig{
		IndexerRequestHeader: parseNewzIndexerRequestHeader(getEnv("STREMTHRU_NEWZ_QUERY_HEADER"), getEnv("STREMTHRU_NEWZ_GRAB_HEADER")),
	}

	return newz
}()
