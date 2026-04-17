package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	llog "github.com/MunifTanjim/stremthru/internal/logger/log"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/google/uuid"
)

const (
	EnvDev  string = "dev"
	EnvProd string = "prod"
	EnvTest string = "test"
)

var Environment = func() string {
	if testing.Testing() {
		return EnvTest
	}

	value, _ := os.LookupEnv("STREMTHRU_ENV")
	switch value {
	case "dev", "development":
		return EnvDev
	case "prod", "production":
		return EnvProd
	case "test":
		return EnvTest
	default:
		return ""
	}
}()

var defaultValueByEnv = map[string]map[string]string{
	EnvDev: {
		"STREMTHRU_LOG_FORMAT": "text",
		"STREMTHRU_LOG_LEVEL":  "DEBUG",
	},
	EnvProd: {},
	EnvTest: {
		"STREMTHRU_LOG_FORMAT": "text",
		"STREMTHRU_LOG_LEVEL":  "DEBUG",
		"STREMTHRU_DATA_DIR":   os.TempDir(),
	},
	"": {
		"STREMTHRU_BASE_URL":                               "http://localhost:8080",
		"STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT":         "*:0",
		"STREMTHRU_DATABASE_URI":                           "sqlite://./data/stremthru.db",
		"STREMTHRU_DATA_DIR":                               "./data",
		"STREMTHRU_LANDING_PAGE":                           "{}",
		"STREMTHRU_LOG_FORMAT":                             "json",
		"STREMTHRU_LOG_LEVEL":                              "INFO",
		"STREMTHRU_PORT":                                   "8080",
		"STREMTHRU_STORE_CONTENT_PROXY":                    "*:true",
		"STREMTHRU_STORE_TUNNEL":                           "*:true",
		"STREMTHRU_STORE_CLIENT_USER_AGENT":                "stremthru",
		"STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME":    "12h",
		"STREMTHRU_INTEGRATION_LETTERBOXD_LIST_STALE_TIME": "24h",
		"STREMTHRU_INTEGRATION_LETTERBOXD_USER_AGENT":      "stremthru",
		"STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME":    "12h",
		"STREMTHRU_INTEGRATION_SERIALIZD_LIST_STALE_TIME":  "24h",
		"STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME":       "12h",
		"STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME":      "12h",
		"STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME":       "12h",
		"STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT":     "10",
		"STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT":       "15s",
		"STREMTHRU_STREMIO_NEWZ_PLAYBACK_WAIT_TIME":        "15s",
		"STREMTHRU_STREMIO_STORE_CATALOG_ITEM_LIMIT":       "2000",
		"STREMTHRU_STREMIO_STORE_CATALOG_CACHE_TIME":       "10m",
		"STREMTHRU_TORZ_TORRENT_FILE_CACHE_SIZE":           "256MB",
		"STREMTHRU_TORZ_TORRENT_FILE_CACHE_TTL":            "6h",
		"STREMTHRU_TORZ_TORRENT_FILE_MAX_SIZE":             "1MB",
		"STREMTHRU_STREMIO_TORZ_INDEXER_MAX_TIMEOUT":       "10s",
		"STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_INDEXER_COUNT":  "2",
		"STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_STORE_COUNT":    "3",
		"STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_UPSTREAM_COUNT": "5",
		"STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_STORE_COUNT":    "3",
		"STREMTHRU_IP_CHECKER":                             "aws,akamai,api.ipify.org",
		"STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM":         "8",
		"STREMTHRU_NEWZ_NZB_FILE_CACHE_SIZE":               "512MB",
		"STREMTHRU_NEWZ_NZB_FILE_CACHE_TTL":                "24h",
		"STREMTHRU_NEWZ_NZB_FILE_MAX_SIZE":                 "50MB",
		"STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE":                "10GB",
		"STREMTHRU_NEWZ_STREAM_BUFFER_SIZE":                "200MB",
		"STREMTHRU_NEWZ_NZB_LINK_TYPE":                     "*:proxy",
	},
}

func getEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists && len(value) > 0 {
		return value
	}
	if val, found := defaultValueByEnv[Environment][key]; found && len(val) > 0 {
		return val
	}
	if Environment != "" {
		if val, found := defaultValueByEnv[""][key]; found && len(val) > 0 {
			return val
		}
	}
	return ""
}

func parseDuration(key string, value string, boundary ...time.Duration) (time.Duration, error) {
	if duration, err := time.ParseDuration(value); err != nil {
		return -1, fmt.Errorf("invalid %s (%s): %v", key, value, err)
	} else if len(boundary) > 0 && boundary[0] > 0 && duration < boundary[0] {
		return -1, fmt.Errorf("%s (%s) must be at least %s", key, duration, boundary[0].String())
	} else if len(boundary) > 1 && boundary[1] > 0 && duration > boundary[1] {
		return -1, fmt.Errorf("%s (%s) must be at most %s", key, duration, boundary[1].String())
	} else {
		return duration, nil
	}
}

func mustParseDuration(key string, value string, boundary ...time.Duration) time.Duration {
	duration, err := parseDuration(key, value, boundary...)
	if err != nil {
		log.Fatal(err)
	}
	return duration
}

type StoreAuthTokenMap map[string]map[string]string

func (m StoreAuthTokenMap) GetToken(user, store string) string {
	if um, ok := m[user]; ok {
		if token, ok := um[store]; ok {
			return token
		}
	}
	if user != "*" {
		return m.GetToken("*", store)
	}
	return ""
}

func (m StoreAuthTokenMap) setToken(user, store, token string) {
	if _, ok := m[user]; !ok {
		m[user] = make(map[string]string)
	}
	m[user][store] = token
}

func (m StoreAuthTokenMap) GetPreferredStore(user string) string {
	store := m.GetToken(user, "*")
	store, _, _ = strings.Cut(store, " ")
	return store
}

func (m StoreAuthTokenMap) ListStores(user string) []string {
	stores := m.GetToken(user, "*")
	return strings.Fields(stores)
}

func (m StoreAuthTokenMap) getStores(user string) string {
	if um, ok := m[user]; ok {
		if stores, ok := um["*"]; ok {
			return stores
		}
	}
	return ""
}

func (m StoreAuthTokenMap) addStore(user, store string) {
	stores := m.getStores(user)
	if stores == "" {
		stores = store
	} else if !strings.Contains(stores, store) {
		stores += " " + store
	}
	m.setToken(user, "*", stores)
}

type AuthMap struct {
	user_pass  map[string]string
	admin_pass map[string]string
	is_admin   map[string]bool

	sabnzbd_apikey map[string]string
}

func (m AuthMap) GetPassword(user string) string {
	if password, ok := m.user_pass[user]; ok {
		return password
	}
	if password, ok := m.admin_pass[user]; ok {
		return password
	}
	return ""
}

func (m AuthMap) IsAdmin(user string) bool {
	isAdmin, ok := m.is_admin[user]
	return ok && isAdmin
}

func (m AuthMap) ListUsers() map[string]string {
	users := make(map[string]string, len(m.user_pass))
	for user := range m.user_pass {
		users[user] = ""
	}
	return users
}

func (m AuthMap) GetSABnzbdUser(apikey string) string {
	for user, key := range m.sabnzbd_apikey {
		if key == apikey {
			return user
		}
	}
	return ""
}

const (
	StremioAddonSidekick string = "sidekick"
	StremioAddonStore    string = "store"
	StremioAddonWrap     string = "wrap"
)

const (
	FeatureAnime           string = "anime"
	FeatureDMMHashlist     string = "dmm_hashlist"
	FeatureIMDBTitle       string = "imdb_title"
	FeatureStremioList     string = "stremio_list"
	FeatureStremioNewz     string = "stremio_newz"
	FeatureStremioP2P      string = "stremio_p2p"
	FeatureStremioSidekick string = "stremio_sidekick"
	FeatureStremioStore    string = "stremio_store"
	FeatureStremioTorz     string = "stremio_torz"
	FeatureStremioWrap     string = "stremio_wrap"
	FeatureVault           string = "vault"
	FeatureProbeMediaInfo  string = "probe_media_info"
)

var features = []string{
	FeatureAnime,
	FeatureDMMHashlist,
	FeatureIMDBTitle,
	FeatureStremioList,
	FeatureStremioNewz,
	FeatureStremioP2P,
	FeatureStremioSidekick,
	FeatureStremioStore,
	FeatureStremioTorz,
	FeatureStremioWrap,
	FeatureVault,
	FeatureProbeMediaInfo,
}

type FeatureConfig struct {
	enabled  []string
	disabled []string
}

func (f FeatureConfig) IsDisabled(name string) bool {
	return slices.Contains(f.disabled, name)
}

func (f FeatureConfig) IsEnabled(name string) bool {
	if f.IsDisabled(name) {
		return false
	}

	if len(f.enabled) == 0 {
		return true
	}

	return slices.Contains(f.enabled, name)
}

func (f FeatureConfig) HasStremioList() bool {
	return f.IsEnabled(FeatureStremioList)
}

func (f FeatureConfig) HasStremioNewz() bool {
	return f.IsEnabled(FeatureStremioNewz)
}

func (f FeatureConfig) HasTorrentInfo() bool {
	return f.IsEnabled(FeatureStremioStore) || f.IsEnabled(FeatureStremioTorz)
}

func (f FeatureConfig) HasDMMHashlist() bool {
	return !f.IsDisabled(FeatureDMMHashlist) && f.HasTorrentInfo()
}

func (f FeatureConfig) HasIMDBTitle() bool {
	return !f.IsDisabled(FeatureIMDBTitle) && f.HasTorrentInfo()
}

func (f FeatureConfig) HasVault() bool {
	return !f.IsDisabled(FeatureVault) && VaultSecret != ""
}

type StoreContentProxyMap map[string]bool

func (scp StoreContentProxyMap) IsEnabled(name string) bool {
	if enabled, ok := scp[name]; ok {
		return enabled
	}
	if name != "*" {
		scp[name] = scp.IsEnabled("*")
	} else {
		scp[name] = true
	}
	return scp[name]
}

type ContentProxyConnectionLimitMap map[string]int

func (cpcl ContentProxyConnectionLimitMap) Get(user string) int {
	if limit, ok := cpcl[user]; ok {
		return limit
	}
	if user != "*" {
		cpcl[user] = cpcl.Get("*")
	} else {
		cpcl[user] = 0
	}
	return cpcl[user]
}

type storeContentCachedStaleTimeMapItem struct {
	cached   time.Duration
	uncached time.Duration
}

type storeContentCachedStaleTimeMap map[string]storeContentCachedStaleTimeMapItem

func (sccst storeContentCachedStaleTimeMap) GetStaleTime(isCached bool, storeName string) time.Duration {
	if staleTime, ok := sccst[storeName]; ok {
		if isCached {
			return staleTime.cached
		}
		return staleTime.uncached
	}
	if storeName != "*" {
		return sccst.GetStaleTime(isCached, "*")
	}
	return 0
}

func parseStoreContentCachedStaleTime(staleTimeConfig string) (staleTimeMap storeContentCachedStaleTimeMap, err error) {
	staleTimeMap = storeContentCachedStaleTimeMap{}
	staleTimeList := strings.FieldsFunc(staleTimeConfig, func(c rune) bool {
		return c == ','
	})

	for _, staleTimeString := range staleTimeList {
		parts := strings.SplitN(staleTimeString, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid stale time: %s", staleTimeString)
		}

		staleTime := storeContentCachedStaleTimeMapItem{}
		store, cachedStaleTime, uncachedStaleTime := parts[0], parts[1], parts[2]

		if cachedStaleDuration, err := time.ParseDuration(cachedStaleTime); err != nil {
			return nil, fmt.Errorf("invalid cached stale time (%s): %v", cachedStaleTime, err)
		} else if cachedStaleDuration < 18*time.Hour {
			return nil, fmt.Errorf("cached stale time (%s) must be at least 18h", cachedStaleTime)
		} else {
			staleTime.cached = cachedStaleDuration
		}

		if uncachedStaleDuration, err := time.ParseDuration(uncachedStaleTime); err != nil {
			return nil, fmt.Errorf("invalid uncached stale time (%s): %v", uncachedStaleTime, err)
		} else if uncachedStaleDuration < 6*time.Hour {
			return nil, fmt.Errorf("uncached stale time (%s) must be at least 6h", uncachedStaleTime)
		} else {
			staleTime.uncached = uncachedStaleDuration
		}

		staleTimeMap[store] = staleTime
	}

	if _, ok := staleTimeMap["*"]; !ok {
		staleTimeMap["*"] = storeContentCachedStaleTimeMapItem{
			cached:   24 * time.Hour,
			uncached: 8 * time.Hour,
		}
	}

	return staleTimeMap, nil
}

type configPeerFlag struct {
	Lazy        bool
	NoSpillTorz bool
}

type Config struct {
	LogLevel  llog.Level
	LogFormat string

	ListenAddr                  string
	Port                        string
	StoreAuthToken              StoreAuthTokenMap
	Auth                        AuthMap
	BuddyURL                    string
	HasBuddy                    bool
	PeerURL                     string
	PeerAuthToken               string
	PeerFlag                    configPeerFlag
	HasPeer                     bool
	PullPeerURL                 string
	RedisURI                    string
	DatabaseURI                 string
	DatabaseReplicaURIs         []string
	Feature                     FeatureConfig
	Version                     string
	LandingPage                 string
	ServerStartTime             time.Time
	StoreContentProxy           StoreContentProxyMap
	StoreContentCachedStaleTime storeContentCachedStaleTimeMap
	StoreClientUserAgent        string
	ContentProxyConnectionLimit ContentProxyConnectionLimitMap

	DataDir     string
	VaultSecret string
}

func parseUri(uri string) (parsedUrl, parsedToken string) {
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatalf("invalid uri: %s", uri)
	}
	if password, ok := u.User.Password(); ok {
		parsedToken = password
	} else {
		parsedToken = u.User.Username()
	}
	u.User = nil
	parsedUrl = strings.TrimSpace(u.String())
	return
}

var config = func() Config {
	authMap := AuthMap{
		user_pass:  map[string]string{},
		admin_pass: map[string]string{},
		is_admin:   map[string]bool{},

		sabnzbd_apikey: map[string]string{},
	}

	authCredStr := getEnv("STREMTHRU_AUTH")
	if authCredStr == "" {
		// deprecated
		authCredStr = getEnv("STREMTHRU_PROXY_AUTH")
	}
	for _, cred := range strings.FieldsFunc(authCredStr, func(c rune) bool {
		return c == ','
	}) {
		if basicAuth, err := util.ParseBasicAuth(cred); err == nil {
			authMap.user_pass[basicAuth.Username] = basicAuth.Password
		}
	}

	for _, admin := range strings.FieldsFunc(getEnv("STREMTHRU_AUTH_ADMIN"), func(c rune) bool {
		return c == ','
	}) {
		if strings.Contains(admin, ":") {
			if basicAuth, err := util.ParseBasicAuth(admin); err == nil {
				if authMap.GetPassword(basicAuth.Username) != "" {
					log.Fatalf("password already set for user: %s", basicAuth.Username)
				}
				authMap.is_admin[basicAuth.Username] = true
				authMap.admin_pass[basicAuth.Username] = basicAuth.Password
			}
		} else if password := authMap.GetPassword(admin); password != "" {
			authMap.is_admin[admin] = true
		}
	}
	if len(authMap.is_admin) == 0 {
		for username := range authMap.user_pass {
			authMap.is_admin[username] = true
		}
	}
	if len(authMap.user_pass) == 0 {
		username := "st-" + util.GenerateRandomString(7, util.CharSet.AlphaNumeric)
		password := util.GenerateRandomString(27, util.CharSet.AlphaNumericMixedCase)
		authMap.is_admin[username] = true
		authMap.admin_pass[username] = password
	}

	for _, creds := range strings.FieldsFunc(getEnv("STREMTHRU_AUTH_SABNZBD"), func(c rune) bool {
		return c == ','
	}) {
		if !strings.Contains(creds, ":") {
			log.Fatalf("invalid sabnzbd auth credential, expected format: <username:apikey>, got: <%s>", creds)
		}
		basicAuth, err := util.ParseBasicAuth(creds)
		if err != nil {
			log.Fatalf("invalid sabnzbd auth credential: <%s>, error: %v", creds, err)
		}
		authMap.sabnzbd_apikey[basicAuth.Username] = basicAuth.Password
	}

	storeAlldebridTokenList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_AUTH"), func(c rune) bool {
		return c == ','
	})
	storeAuthTokenMap := make(StoreAuthTokenMap)
	for _, userStoreToken := range storeAlldebridTokenList {
		if user, storeToken, ok := strings.Cut(userStoreToken, ":"); ok {
			if storeName, token, ok := strings.Cut(storeToken, ":"); ok {
				if !store.StoreName(storeName).IsValid() {
					log.Fatalf("invalid store name: %s", storeName)
				}
				storeAuthTokenMap.addStore(user, storeName)
				storeAuthTokenMap.setToken(user, storeName, token)
			}
		}
	}

	buddyUrl, _ := parseUri(getEnv("STREMTHRU_BUDDY_URI"))
	pullPeerUrl := ""
	if buddyUrl != "" {
		pullPeerUrl, _ = parseUri(getEnv("STREMTHRU__PULL__PEER_URI"))
	}

	defaultPeerUri := ""
	if peerUri, err := util.Base64Decode("aHR0cHM6Ly9zdHJlbXRocnUuMTMzNzcwMDEueHl6"); err == nil && buddyUrl == "" {
		defaultPeerUri = peerUri
	}
	peerUri := getEnv("STREMTHRU_PEER_URI")
	if peerUri == "" {
		peerUri = defaultPeerUri
	}
	peerUrl, peerAuthToken := "", ""
	if peerUri != "-" {
		peerUrl, peerAuthToken = parseUri(peerUri)
	}

	databaseUri := getEnv("STREMTHRU_DATABASE_URI")
	if _, err := url.Parse(databaseUri); err != nil {
		log.Fatalf("invalid database uri: %v", err)
	}
	databaseReplicaUris := []string{}
	for uri := range strings.FieldsFuncSeq(getEnv("STREMTHRU_DATABASE_REPLICA_URIS"), func(c rune) bool {
		return c == ','
	}) {
		uri = strings.TrimSpace(uri)
		if _, err := url.Parse(uri); err != nil {
			log.Fatalf("invalid database replica uri: %v", err)
		}
		databaseReplicaUris = append(databaseReplicaUris, uri)
	}

	feature := FeatureConfig{
		disabled: []string{FeatureAnime, FeatureStremioP2P},
	}
	for _, name := range strings.FieldsFunc(strings.TrimSpace(getEnv("STREMTHRU_FEATURE")), func(c rune) bool {
		return c == ','
	}) {
		switch {
		case strings.HasPrefix(name, "-"):
			name = strings.TrimPrefix(name, "-")
			if slices.Contains(feature.enabled, name) {
				log.Fatalf("feature conflict, trying to disable already enabled feature: -%s", name)
			} else {
				feature.disabled = append(feature.disabled, name)
			}
		case strings.HasPrefix(name, "+"):
			name = strings.TrimPrefix(name, "+")
			if slices.Contains(feature.disabled, name) {
				feature.disabled = slices.DeleteFunc(feature.disabled, func(feat string) bool {
					return feat == name
				})
			} else {
				log.Fatalf("feature conflict, trying to force enable a not disabled feature: +%s", name)
			}
		default:
			if slices.Contains(feature.disabled, name) {
				log.Fatalf("feature conflict, trying to enable already disabled feature: %s", name)
			} else {
				feature.enabled = append(feature.enabled, name)
			}
		}
	}

	storeContentProxyList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_CONTENT_PROXY"), func(c rune) bool {
		return c == ','
	})

	storeContentProxyMap := make(StoreContentProxyMap)
	for _, storeContentProxy := range storeContentProxyList {
		if store, enabled, ok := strings.Cut(storeContentProxy, ":"); ok {
			storeContentProxyMap[store] = enabled == "true"
		}
	}

	var logLevel llog.Level
	if err := logLevel.UnmarshalText([]byte(getEnv("STREMTHRU_LOG_LEVEL"))); err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}

	logFormat := getEnv("STREMTHRU_LOG_FORMAT")
	if logFormat != "json" && logFormat != "text" {
		log.Fatalf("Invalid log format: %s, expected: json / text", logFormat)
	}

	contentProxyConnectionMap := make(ContentProxyConnectionLimitMap)
	contentProxyConnectionList := strings.FieldsFunc(getEnv("STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT"), func(c rune) bool {
		return c == ','
	})
	for _, contentProxyConnection := range contentProxyConnectionList {
		if user, limitStr, ok := strings.Cut(contentProxyConnection, ":"); ok {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				log.Fatalf("Invalid content proxy connection limit: %v", err)
			}
			contentProxyConnectionMap[user] = max(0, limit)
		}
	}

	dataDir, err := filepath.Abs(getEnv("STREMTHRU_DATA_DIR"))
	if err != nil {
		log.Fatalf("failed to resolve data directory: %v", err)
	} else if exists, err := util.DirExists(dataDir); err != nil {
		log.Fatalf("failed to check data directory: %v", err)
	} else if !exists {
		log.Fatalf("data directory does not exist: %v", dataDir)
	}

	storeContentCachedStaleTimeMap, err := parseStoreContentCachedStaleTime(getEnv("STREMTHRU_STORE_CONTENT_CACHED_STALE_TIME"))
	if err != nil {
		log.Fatalf("failed to parse store content cached stale time: %v", err)
	}

	vaultSecret := getEnv("STREMTHRU_VAULT_SECRET")

	// @deprecated
	lazyPeer := strings.ToLower(getEnv("STREMTHRU_LAZY_PEER"))

	peerFlag := configPeerFlag{}
	for _, flag := range strings.FieldsFunc(getEnv("STREMTHRU_PEER_FLAG"), func(c rune) bool {
		return c == ','
	}) {
		switch flag {
		case "lazy":
			peerFlag.Lazy = true
		case "no_spill_torz":
			peerFlag.NoSpillTorz = true
		}
	}

	if lazyPeer == "1" || lazyPeer == "true" {
		log.Println("WARNING: STREMTHRU_LAZY_PEER is deprecated, use STREMTHRU_PEER_FLAG=lazy instead")
		peerFlag.Lazy = true
	}

	listenAddr := getEnv("STREMTHRU_LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":" + getEnv("STREMTHRU_PORT")
		if Environment == EnvDev {
			listenAddr = "127.0.0.1" + listenAddr
		}
	}

	redisUri := getEnv("STREMTHRU_REDIS_URI")
	if _, err := url.Parse(redisUri); err != nil {
		log.Fatalf("invalid redis uri: %v", err)
	}

	return Config{
		LogLevel:  logLevel,
		LogFormat: logFormat,

		ListenAddr:                  listenAddr,
		Auth:                        authMap,
		StoreAuthToken:              storeAuthTokenMap,
		BuddyURL:                    buddyUrl,
		HasBuddy:                    len(buddyUrl) > 0,
		PeerURL:                     peerUrl,
		PeerAuthToken:               peerAuthToken,
		PeerFlag:                    peerFlag,
		HasPeer:                     len(peerUrl) > 0,
		PullPeerURL:                 pullPeerUrl,
		RedisURI:                    redisUri,
		DatabaseURI:                 databaseUri,
		DatabaseReplicaURIs:         databaseReplicaUris,
		Feature:                     feature,
		Version:                     "0.100.0", // x-release-please-version
		LandingPage:                 getEnv("STREMTHRU_LANDING_PAGE"),
		ServerStartTime:             time.Now(),
		StoreContentProxy:           storeContentProxyMap,
		StoreContentCachedStaleTime: storeContentCachedStaleTimeMap,
		StoreClientUserAgent:        getEnv("STREMTHRU_STORE_CLIENT_USER_AGENT"),
		ContentProxyConnectionLimit: contentProxyConnectionMap,
		DataDir:                     dataDir,
		VaultSecret:                 vaultSecret,
	}
}()

var LogLevel = config.LogLevel
var LogFormat = config.LogFormat

var ListenAddr = config.ListenAddr
var Auth = config.Auth
var StoreAuthToken = config.StoreAuthToken
var BuddyURL = config.BuddyURL
var HasBuddy = config.HasBuddy
var PeerURL = config.PeerURL
var PeerAuthToken = config.PeerAuthToken
var PeerFlag = config.PeerFlag
var HasPeer = config.HasPeer
var PullPeerURL = config.PullPeerURL
var RedisURI = config.RedisURI
var DatabaseURI = config.DatabaseURI
var DatabaseReplicaURIs = config.DatabaseReplicaURIs
var Feature = config.Feature
var Version = config.Version
var LandingPage = config.LandingPage
var ServerStartTime = config.ServerStartTime
var StoreContentProxy = config.StoreContentProxy
var StoreContentCachedStaleTime = config.StoreContentCachedStaleTime
var StoreClientUserAgent = config.StoreClientUserAgent
var ContentProxyConnectionLimit = config.ContentProxyConnectionLimit
var InstanceId = strings.ReplaceAll(uuid.NewString(), "-", "")

var IsTrusted = func() bool {
	rootHost := util.MustDecodeBase64("c3RyZW10aHJ1LjEzMzc3MDAxLnh5eg==")
	switch BaseURL.Hostname() {
	case rootHost:
		return true
	}
	if config.PeerURL == "" || config.PeerAuthToken == "" {
		return false
	}
	u := util.MustParseURL(config.PeerURL)
	switch u.Hostname() {
	case rootHost, "localhost":
		return true
	}
	return false
}()

var DataDir = config.DataDir
var VaultSecret = config.VaultSecret

var IsPublicInstance = len(Auth.user_pass) == 0

func getRedactedURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	return u.Redacted(), nil
}

type AppState struct {
	StoreNames []string
}

func PrintConfig(state *AppState) {
	data := BuildConfigDisplay(state.StoreNames)

	l := log.New(os.Stderr, "=", 0)
	l.Println("====== StremThru =======")
	l.Printf(" Time: %v\n", data.Server.StartedAt.Format(time.RFC3339))
	l.Printf(" Version: %v\n", data.Server.Version)
	l.Printf(" Addr: %v\n", data.Server.ListenAddr)
	if data.Server.Environment != "" {
		l.Printf(" Env: %v\n", data.Server.Environment)
	}
	l.Println("========================")
	l.Println()

	l.Printf("  Log Level: %s\n", data.Server.LogLevel)
	l.Printf(" Log Format: %s\n", data.Server.LogFormat)
	l.Println()

	if !data.Tunnel.Disabled {
		l.Println(" Tunnel:")
		if data.Tunnel.Default != "" {
			defaultProxyConfig := ""
			if noProxy := getEnv("NO_PROXY"); noProxy == "*" {
				defaultProxyConfig = " (disabled)"
			}
			l.Println("   Default: " + data.Tunnel.Default + defaultProxyConfig)
			l.Println("   [Store]: " + data.Tunnel.Default)
		}

		if len(data.Tunnel.ByHost) > 0 {
			l.Println("   By Host:")
			for hostname, proxy := range data.Tunnel.ByHost {
				l.Println("     " + hostname + ": " + proxy)
			}
		}

		l.Println()
	}

	l.Println(" Machine IP: " + data.Network.MachineIP)
	if len(data.Network.TunnelIPs) > 0 {
		l.Println("  Tunnel IP: ")
		for proxyHost, tunnelIp := range data.Network.TunnelIPs {
			if tunnelIp == "" {
				tunnelIp = "(unresolved)"
			}
			l.Println("    [" + proxyHost + "]: " + tunnelIp)
		}
	}
	l.Println()

	l.Printf("   Base URL: %s\n", data.Instance.BaseURL)
	l.Println()

	if len(data.Users) > 0 {
		l.Println(" Users:")
		for _, user := range data.Users {
			l.Println("   - " + user.Name)
			l.Println("       store: " + strings.Join(user.Stores, ","))
			if user.ContentProxyConnectionLimit > 0 {
				l.Println("       content_proxy_connection_limit: " + strconv.FormatUint(uint64(user.ContentProxyConnectionLimit), 10))
			}
		}
		l.Println()
	}

	l.Println(" Stores:")
	for _, s := range data.Stores.Items {
		storeConfig := ""
		if s.Config != "" {
			storeConfig = " (" + s.Config + ")"
		}
		l.Println("   - " + s.Name + storeConfig)
	}
	l.Println()

	if len(Auth.admin_pass) == 1 {
		for username, password := range Auth.admin_pass {
			if strings.HasPrefix(username, "st-") {
				l.Println(" (Auto Generated) Admin Creds:")
				l.Println("   " + username + ":" + password)
				l.Println()
			}
		}
	}

	if data.Network.BuddyURL != "" {
		l.Println(" Buddy URI:")
		l.Println("   " + data.Network.BuddyURL)
		l.Println()
	}

	if data.Network.PeerURL != "" {
		peerFlags := ""
		if data.Network.PeerFlags != "" {
			peerFlags = " (" + data.Network.PeerFlags + ")"
		}
		l.Println(" Peer URI" + peerFlags + ":")
		l.Println("   " + data.Network.PeerURL)
		l.Println()
	}
	if data.Network.PullPeerURL != "" {
		l.Println(" (Pull) Peer URI:")
		l.Println("   " + data.Network.PullPeerURL)
		l.Println()
	}

	if !data.Redis.Disabled {
		l.Println(" Redis URI:")
		l.Println("   " + data.Redis.URI)
		l.Println()
	}

	l.Println(" Database URI:")
	l.Println("   " + data.Database.URI)
	if len(data.Database.ReplicaURIs) > 0 {
		l.Println(" Replica URIs:")
		for _, uri := range data.Database.ReplicaURIs {
			l.Println("   " + uri)
		}
	}
	l.Println()

	l.Println(" Features:")
	for _, feature := range data.Features {
		disabled := ""
		if !feature.Enabled {
			disabled = " (disabled)"
		}
		l.Println("   - " + feature.Name + disabled)
		if feature.Enabled && len(feature.Settings) > 0 {
			for key, value := range feature.Settings {
				l.Println("       " + key + ": " + value)
			}
		}
	}
	l.Println()

	l.Println(" Integrations:")
	for _, integration := range data.Integrations {
		disabled := ""
		if !integration.Enabled {
			disabled = " (disabled)"
		}
		l.Println("   - " + integration.Name + disabled)
		if len(integration.Settings) > 0 {
			for key, value := range integration.Settings {
				l.Println("       " + key + ": " + value)
			}
		}
	}
	l.Println()

	if !data.Newz.Disabled {
		l.Println(" Newz:")
		l.Println("   max conn. per stream: " + data.Newz.MaxConnectionPerStream)
		l.Println("    nzb file cache size: " + data.Newz.NZBFileCacheSize)
		l.Println("     nzb file cache ttl: " + data.Newz.NZBFileCacheTTL)
		l.Println("      nzb file max size: " + data.Newz.NZBFileMaxSize)
		l.Println("     segment cache size: " + data.Newz.SegmentCacheSize)
		l.Println("     stream buffer size: " + data.Newz.StreamBufferSize)
		l.Println()
	}

	l.Println(" Torz:")
	if data.Torz.TorrentFileCacheSize != "" {
		l.Println("   torrent file cache size: " + data.Torz.TorrentFileCacheSize)
	}
	if data.Torz.TorrentFileCacheTTL != "" {
		l.Println("    torrent file cache ttl: " + data.Torz.TorrentFileCacheTTL)
	}
	l.Println("     torrent file max size: " + data.Torz.TorrentFileMaxSize)
	l.Println()

	l.Println(" Instance ID:")
	l.Println("   " + data.Instance.ID)
	l.Println()

	l.Println(" Data Directory:")
	l.Println("   " + data.Server.DataDir)
	l.Println()

	l.Print("========================\n\n")
}
