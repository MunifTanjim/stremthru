package config

import (
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type stremioConfigList struct {
	PublicMaxListCount int
}

type stremioConfigStore struct {
	CatalogItemLimit int
	CatalogCacheTime time.Duration
}

type stremioConfigTorz struct {
	IndexerMaxTimeout     time.Duration
	LazyPull              bool
	PublicMaxIndexerCount int
	PublicMaxStoreCount   int
}

type stremioConfigWrap struct {
	PublicMaxUpstreamCount int
	PublicMaxStoreCount    int
}

type NZBLinkType string

var (
	NZBLinkTypeProxy    NZBLinkType = "proxy"
	NZBLinkTypeRedirect NZBLinkType = "redirect"
)

type nzbLinkTypeMap map[string]NZBLinkType

func (m nzbLinkTypeMap) Proxy(hostname string) bool {
	if mode, ok := m[hostname]; ok {
		return mode == NZBLinkTypeProxy
	}
	if mode, ok := m["*"]; ok {
		return mode == NZBLinkTypeProxy
	}
	return false
}

func (m nzbLinkTypeMap) Redirect(hostname string) bool {
	if mode, ok := m[hostname]; ok {
		return mode == NZBLinkTypeRedirect
	}
	if mode, ok := m["*"]; ok {
		return mode == NZBLinkTypeRedirect
	}
	return false
}

type stremioConfigNewz struct {
	IndexerMaxTimeout time.Duration
	NZBLinkType       nzbLinkTypeMap
}

type StremioConfig struct {
	List  stremioConfigList
	Store stremioConfigStore
	Torz  stremioConfigTorz
	Wrap  stremioConfigWrap
	Newz  stremioConfigNewz
}

func parseStremio() StremioConfig {
	stremio := StremioConfig{
		List: stremioConfigList{
			PublicMaxListCount: util.MustParseInt(getEnv("STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT")),
		},
		Store: stremioConfigStore{
			CatalogItemLimit: util.MustParseInt(getEnv("STREMTHRU_STREMIO_STORE_CATALOG_ITEM_LIMIT")),
			CatalogCacheTime: mustParseDuration("store catalog cache time", getEnv("STREMTHRU_STREMIO_STORE_CATALOG_CACHE_TIME"), 1*time.Minute),
		},
		Torz: stremioConfigTorz{
			IndexerMaxTimeout:     mustParseDuration("stremio torz indexer max timeout", getEnv("STREMTHRU_STREMIO_TORZ_INDEXER_MAX_TIMEOUT"), 2*time.Second, 60*time.Second),
			LazyPull:              strings.ToLower(getEnv("STREMTHRU_STREMIO_TORZ_LAZY_PULL")) == "true",
			PublicMaxIndexerCount: util.MustParseInt(getEnv("STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_INDEXER_COUNT")),
			PublicMaxStoreCount:   util.MustParseInt(getEnv("STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_STORE_COUNT")),
		},
		Wrap: stremioConfigWrap{
			PublicMaxUpstreamCount: util.MustParseInt(getEnv("STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_UPSTREAM_COUNT")),
			PublicMaxStoreCount:    util.MustParseInt(getEnv("STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_STORE_COUNT")),
		},
		Newz: stremioConfigNewz{
			IndexerMaxTimeout: mustParseDuration("stremio newz indexer max timeout", getEnv("STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT"), 2*time.Second, 60*time.Second),
			NZBLinkType: nzbLinkTypeMap{
				"*": NZBLinkTypeProxy,
			},
		},
	}

	for _, entry := range strings.FieldsFunc(getEnv("STREMTHRU_STREMIO_NEWZ_NZB_LINK_TYPE"), func(c rune) bool {
		return c == ','
	}) {
		if hostname, lType, ok := strings.Cut(entry, ":"); ok && hostname != "" && lType != "" {
			stremio.Newz.NZBLinkType[hostname] = NZBLinkType(lType)
		}
	}

	return stremio
}

var Stremio = parseStremio()
