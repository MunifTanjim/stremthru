package config

import (
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
