package config

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type ConfigDisplayServer struct {
	Version        string    `json:"version"`
	Environment    string    `json:"environment"`
	ListenAddr     string    `json:"listen_addr"`
	LogLevel       string    `json:"log_level"`
	LogFormat      string    `json:"log_format"`
	DataDir        string    `json:"data_dir"`
	StartedAt      time.Time `json:"started_at"`
	PosthogEnabled bool      `json:"posthog_enabled"`
}

type ConfigDisplayInstance struct {
	ID               string `json:"id"`
	BaseURL          string `json:"base_url"`
	IsPublicInstance bool   `json:"is_public_instance"`
	StremioLocked    bool   `json:"stremio_locked"`
}

type ConfigDisplayTunnel struct {
	Disabled bool              `json:"disabled"`
	Default  string            `json:"default"`
	ByHost   map[string]string `json:"by_host"`
}

type ConfigDisplayNetwork struct {
	MachineIP   string            `json:"machine_ip"`
	TunnelIPs   map[string]string `json:"tunnel_ips,omitempty"`
	BuddyURL    string            `json:"buddy_url,omitempty"`
	PeerURL     string            `json:"peer_url,omitempty"`
	PeerFlags   string            `json:"peer_flags,omitempty"`
	PullPeerURL string            `json:"pull_peer_url,omitempty"`
}

type ConfigDisplayDatabase struct {
	URI         string   `json:"uri"`
	ReplicaURIs []string `json:"replica_uris,omitempty"`
}

type ConfigDisplayRedis struct {
	Disabled bool   `json:"disabled"`
	URI      string `json:"uri"`
}

type ConfigDisplayUser struct {
	Name                        string   `json:"name"`
	IsAdmin                     bool     `json:"is_admin"`
	Stores                      []string `json:"stores"`
	ContentProxyConnectionLimit uint32   `json:"content_proxy_connection_limit,omitempty"`
}

type ConfigDisplayStores struct {
	ClientUserAgent string               `json:"client_user_agent"`
	Items           []ConfigDisplayStore `json:"items"`
}

type ConfigDisplayStore struct {
	Name              string `json:"name"`
	Config            string `json:"config,omitempty"`
	CachedStaleTime   string `json:"cached_stale_time,omitempty"`
	UncachedStaleTime string `json:"uncached_stale_time,omitempty"`
}

type ConfigDisplayFeature struct {
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Settings map[string]string `json:"settings,omitempty"`
}

type ConfigDisplayIntegration struct {
	Name     string            `json:"name"`
	Enabled  bool              `json:"enabled"`
	Settings map[string]string `json:"settings,omitempty"`
}

type ConfigDisplayNewz struct {
	Disabled               bool              `json:"disabled"`
	MaxConnectionPerStream string            `json:"max_connection_per_stream"`
	NZBFileCacheSize       string            `json:"nzb_file_cache_size"`
	NZBFileCacheTTL        string            `json:"nzb_file_cache_ttl"`
	NZBFileMaxSize         string            `json:"nzb_file_max_size"`
	SegmentCacheSize       string            `json:"segment_cache_size"`
	StreamBufferSize       string            `json:"stream_buffer_size"`
	NZBLinkMode            map[string]string `json:"nzb_link_mode,omitempty"`
}

type ConfigDisplayTorz struct {
	TorrentFileCacheSize string `json:"torrent_file_cache_size"`
	TorrentFileCacheTTL  string `json:"torrent_file_cache_ttl"`
	TorrentFileMaxSize   string `json:"torrent_file_max_size"`
}

type ConfigDisplay struct {
	Server       ConfigDisplayServer        `json:"server"`
	Instance     ConfigDisplayInstance      `json:"instance"`
	Tunnel       ConfigDisplayTunnel        `json:"tunnel"`
	Network      ConfigDisplayNetwork       `json:"network"`
	Database     ConfigDisplayDatabase      `json:"database"`
	Redis        ConfigDisplayRedis         `json:"redis"`
	Users        []ConfigDisplayUser        `json:"users,omitempty"`
	Stores       ConfigDisplayStores        `json:"stores"`
	Features     []ConfigDisplayFeature     `json:"features"`
	Integrations []ConfigDisplayIntegration `json:"integrations"`
	Newz         ConfigDisplayNewz          `json:"newz"`
	Torz         ConfigDisplayTorz          `json:"torz"`
}

func redactURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return u.Redacted()
}

func redactToken(token string) string {
	if len(token) <= 6 {
		return strings.Repeat("*", len(token))
	}
	return token[0:3] + "..." + token[len(token)-3:]
}

func BuildConfigDisplay(storeNames []string) ConfigDisplay {
	data := ConfigDisplay{}

	data.Server.Version = Version
	data.Server.Environment = Environment
	data.Server.ListenAddr = ListenAddr
	data.Server.LogLevel = LogLevel.String()
	data.Server.LogFormat = LogFormat
	data.Server.DataDir = DataDir
	data.Server.StartedAt = ServerStartTime
	data.Server.PosthogEnabled = Posthog.IsEnabled()

	data.Instance.ID = InstanceId
	data.Instance.BaseURL = BaseURL.String()
	data.Instance.IsPublicInstance = IsPublicInstance
	data.Instance.StremioLocked = Stremio.Locked

	data.Tunnel.ByHost = map[string]string{}
	data.Tunnel.Disabled = !Tunnel.HasProxy()
	if !data.Tunnel.Disabled {
		Tunnel.RLock()
		for hostname, proxy := range Tunnel.data {
			if hostname == "*" {
				data.Tunnel.Default = proxy.Redacted()
			} else if proxy.Host == "" {
				data.Tunnel.ByHost[hostname] = "(disabled)"
			} else {
				data.Tunnel.ByHost[hostname] = proxy.Redacted()
			}
		}
		Tunnel.RUnlock()
	}

	data.Network.MachineIP = IP.GetMachineIP()
	if HasBuddy {
		data.Network.BuddyURL = BuddyURL
	}
	if HasPeer {
		peerUrl, err := url.Parse(PeerURL)
		if err == nil {
			peerUrl.User = url.UserPassword("", PeerAuthToken)
			data.Network.PeerURL = peerUrl.Redacted()
		}

		var peerFlags []string
		if PeerFlag.Lazy {
			peerFlags = append(peerFlags, "lazy")
		}
		if PeerFlag.NoSpillTorz {
			peerFlags = append(peerFlags, "no_spill_torz")
		}
		if len(peerFlags) > 0 {
			data.Network.PeerFlags = strings.Join(peerFlags, ",")
		}
	}
	if PullPeerURL != "" {
		data.Network.PullPeerURL = redactURI(PullPeerURL)
	}
	if !data.Tunnel.Disabled {
		if tunnelIPs, err := IP.GetTunnelIPByProxyHost(); err == nil && len(tunnelIPs) > 0 {
			data.Network.TunnelIPs = tunnelIPs
		}
	}

	data.Database.URI = redactURI(DatabaseURI)
	if len(DatabaseReplicaURIs) > 0 {
		for _, replicaURI := range DatabaseReplicaURIs {
			data.Database.ReplicaURIs = append(data.Database.ReplicaURIs, redactURI(replicaURI))
		}
	}

	data.Redis.Disabled = RedisURI == ""
	if !data.Redis.Disabled {
		data.Redis.URI = redactURI(RedisURI)
	}

	if !IsPublicInstance {
		for user := range Auth.ListUsers() {
			stores := StoreAuthToken.ListStores(user)
			preferredStore := StoreAuthToken.GetPreferredStore(user)
			if len(stores) == 0 {
				stores = append(stores, preferredStore)
			} else if len(stores) > 1 {
				for i := range stores {
					if stores[i] == preferredStore {
						stores[i] = "*" + stores[i]
					}
				}
			}

			configUser := ConfigDisplayUser{
				Name:    user,
				IsAdmin: Auth.IsAdmin(user),
				Stores:  stores,
			}
			if cpcl := ContentProxyConnectionLimit.Get(user); cpcl > 0 {
				configUser.ContentProxyConnectionLimit = uint32(cpcl)
			}
			data.Users = append(data.Users, configUser)
		}
	}

	data.Stores.ClientUserAgent = StoreClientUserAgent
	for _, storeName := range storeNames {
		storeConfig := ""
		if !IsPublicInstance && StoreContentProxy.IsEnabled(storeName) {
			storeConfig += "content_proxy"
		}
		if !data.Tunnel.Disabled {
			if StoreTunnel.IsEnabledForAPI(storeName) {
				if storeConfig != "" {
					storeConfig += ","
				}
				storeConfig += "tunnel:api"
				if !IsPublicInstance && StoreTunnel.GetTypeForStream(storeName) == TUNNEL_TYPE_FORCED {
					storeConfig += "+stream"
				}
			}
		}
		cs := ConfigDisplayStore{
			Name:   storeName,
			Config: storeConfig,
		}
		if cachedStaleTime := StoreContentCachedStaleTime.GetStaleTime(true, storeName); cachedStaleTime > 0 {
			cs.CachedStaleTime = cachedStaleTime.String()
		}
		if uncachedStaleTime := StoreContentCachedStaleTime.GetStaleTime(false, storeName); uncachedStaleTime > 0 {
			cs.UncachedStaleTime = uncachedStaleTime.String()
		}
		data.Stores.Items = append(data.Stores.Items, cs)
	}

	data.Features = buildConfigFeatures()

	data.Integrations = buildConfigIntegrations()

	data.Newz.Disabled = !Feature.HasVault()
	if !data.Newz.Disabled {
		data.Newz.MaxConnectionPerStream = strconv.Itoa(Newz.MaxConnectionPerStream)
		data.Newz.NZBFileCacheSize = util.ToSize(Newz.NZBFileCacheSize)
		data.Newz.NZBFileCacheTTL = Newz.NZBFileCacheTTL.String()
		data.Newz.NZBFileMaxSize = util.ToSize(Newz.NZBFileMaxSize)
		data.Newz.SegmentCacheSize = util.ToSize(Newz.SegmentCacheSize)
		data.Newz.StreamBufferSize = util.ToSize(Newz.StreamBufferSize)
		data.Newz.NZBLinkMode = make(map[string]string, len(NewzNZBLinkMode))
		for hostname, mode := range NewzNZBLinkMode {
			data.Newz.NZBLinkMode[hostname] = string(mode)
		}
	}

	data.Torz.TorrentFileMaxSize = util.ToSize(Torz.TorrentFileMaxSize)
	if !IsPublicInstance {
		data.Torz.TorrentFileCacheSize = util.ToSize(Torz.TorrentFileCacheSize)
		data.Torz.TorrentFileCacheTTL = Torz.TorrentFileCacheTTL.String()
	}

	return data
}

func buildConfigFeatures() []ConfigDisplayFeature {
	var items []ConfigDisplayFeature

	for _, name := range features {
		var enabled bool
		switch name {
		case FeatureDMMHashlist:
			enabled = Feature.HasDMMHashlist()
		case FeatureIMDBTitle:
			enabled = Feature.HasIMDBTitle()
		case FeatureVault:
			enabled = Feature.HasVault()
		case FeatureStremioList:
			enabled = Feature.HasStremioList()
		case FeatureStremioNewz:
			enabled = Feature.HasStremioNewz()
		default:
			enabled = Feature.IsEnabled(name)
		}

		feature := ConfigDisplayFeature{
			Name:    name,
			Enabled: enabled,
		}

		if enabled {
			switch name {
			case FeatureStremioList:
				feature.Settings = map[string]string{
					"public_max_list_count": strconv.Itoa(Stremio.List.PublicMaxListCount),
				}
			case FeatureStremioNewz:
				feature.Settings = map[string]string{
					"indexer_max_timeout": Stremio.Newz.IndexerMaxTimeout.String(),
					"playback_wait_time":  Stremio.Newz.PlaybackWaitTime.String(),
				}
			case FeatureStremioStore:
				feature.Settings = map[string]string{
					"catalog_item_limit": strconv.Itoa(Stremio.Store.CatalogItemLimit),
					"catalog_cache_time": Stremio.Store.CatalogCacheTime.String(),
				}
			case FeatureStremioTorz:
				settings := map[string]string{
					"indexer_max_timeout":      Stremio.Torz.IndexerMaxTimeout.String(),
					"public_max_indexer_count": strconv.Itoa(Stremio.Torz.PublicMaxIndexerCount),
					"public_max_store_count":   strconv.Itoa(Stremio.Torz.PublicMaxStoreCount),
				}
				if Stremio.Torz.LazyPull {
					settings["lazy_pull"] = "true"
				}
				feature.Settings = settings
			case FeatureStremioWrap:
				feature.Settings = map[string]string{
					"public_max_upstream_count": strconv.Itoa(Stremio.Wrap.PublicMaxUpstreamCount),
					"public_max_store_count":    strconv.Itoa(Stremio.Wrap.PublicMaxStoreCount),
				}
			case FeatureVault:
				feature.Settings = map[string]string{
					"secret": strings.Repeat("*", len(VaultSecret)),
				}
			}
		}

		items = append(items, feature)
	}

	return items
}

func buildConfigIntegrations() []ConfigDisplayIntegration {
	var items []ConfigDisplayIntegration

	anilistEnabled := Feature.IsEnabled(FeatureAnime)
	anilist := ConfigDisplayIntegration{
		Name:    "anilist.co",
		Enabled: anilistEnabled,
	}
	if anilistEnabled {
		anilist.Settings = map[string]string{
			"list_stale_time": Integration.AniList.ListStaleTime.String(),
		}
	}
	items = append(items, anilist)

	if Integration.Bitmagnet.IsEnabled() {
		items = append(items, ConfigDisplayIntegration{
			Name:    "bitmagnet.io",
			Enabled: true,
			Settings: map[string]string{
				"base_url":     Integration.Bitmagnet.BaseURL.String(),
				"database_uri": redactURI(Integration.Bitmagnet.DatabaseURI),
			},
		})
	}

	githubEnabled := Integration.GitHub.HasDefaultCredentials()
	github := ConfigDisplayIntegration{
		Name:    "github.com",
		Enabled: githubEnabled,
	}
	if githubEnabled {
		github.Settings = map[string]string{
			"user":  Integration.GitHub.User,
			"token": redactToken(Integration.GitHub.Token),
		}
	}
	items = append(items, github)

	kitsuEnabled := Feature.IsEnabled(FeatureAnime) && Integration.Kitsu.HasDefaultCredentials()
	kitsu := ConfigDisplayIntegration{
		Name:    "kitsu.app",
		Enabled: kitsuEnabled,
	}
	if kitsuEnabled {
		settings := map[string]string{
			"email":    Integration.Kitsu.Email,
			"password": "*******",
		}
		if Integration.Kitsu.ClientId != "" {
			settings["client_id"] = redactToken(Integration.Kitsu.ClientId)
		}
		if Integration.Kitsu.ClientSecret != "" {
			settings["client_secret"] = redactToken(Integration.Kitsu.ClientSecret)
		}
		kitsu.Settings = settings
	}
	items = append(items, kitsu)

	letterboxdEnabled := Integration.Letterboxd.IsEnabled()
	letterboxdInfo := ""
	if Integration.Letterboxd.IsPiggybacked() {
		letterboxdInfo = "piggybacked"
	}
	letterboxd := ConfigDisplayIntegration{
		Name:    "letterboxd.com",
		Enabled: letterboxdEnabled || Integration.Letterboxd.IsPiggybacked(),
	}
	if letterboxdEnabled {
		letterboxd.Settings = map[string]string{
			"client_id":       redactToken(Integration.Letterboxd.ClientId),
			"client_secret":   redactToken(Integration.Letterboxd.ClientSecret),
			"user_agent":      Integration.Letterboxd.UserAgent,
			"list_stale_time": Integration.Letterboxd.ListStaleTime.String(),
		}
	} else if letterboxdInfo != "" {
		letterboxd.Settings = map[string]string{
			"mode":            letterboxdInfo,
			"list_stale_time": Integration.Letterboxd.ListStaleTime.String(),
		}
	}
	items = append(items, letterboxd)

	items = append(items, ConfigDisplayIntegration{
		Name:    "mdblist.com",
		Enabled: true,
		Settings: map[string]string{
			"list_stale_time": Integration.MDBList.ListStaleTime.String(),
		},
	})

	items = append(items, ConfigDisplayIntegration{
		Name:    "serializd.com",
		Enabled: Integration.TMDB.IsEnabled(),
		Settings: map[string]string{
			"list_stale_time": Integration.Serializd.ListStaleTime.String(),
		},
	})

	tmdbEnabled := Integration.TMDB.IsEnabled()
	tmdb := ConfigDisplayIntegration{
		Name:    "themoviedb.org",
		Enabled: tmdbEnabled,
	}
	if tmdbEnabled {
		tmdb.Settings = map[string]string{
			"access_token":    redactToken(Integration.TMDB.AccessToken),
			"list_stale_time": Integration.TMDB.ListStaleTime.String(),
		}
	}
	items = append(items, tmdb)

	traktEnabled := Integration.Trakt.IsEnabled()
	trakt := ConfigDisplayIntegration{
		Name:    "trakt.tv",
		Enabled: traktEnabled,
	}
	if traktEnabled {
		trakt.Settings = map[string]string{
			"client_id":       redactToken(Integration.Trakt.ClientId),
			"client_secret":   redactToken(Integration.Trakt.ClientSecret),
			"list_stale_time": Integration.Trakt.ListStaleTime.String(),
		}
	}
	items = append(items, trakt)

	tvdbEnabled := Integration.TVDB.IsEnabled()
	tvdb := ConfigDisplayIntegration{
		Name:    "thetvdb.com",
		Enabled: tvdbEnabled,
	}
	if tvdbEnabled {
		tvdb.Settings = map[string]string{
			"api_key":         redactToken(Integration.TVDB.APIKey),
			"list_stale_time": Integration.TVDB.ListStaleTime.String(),
		}
	}
	items = append(items, tvdb)

	return items
}
