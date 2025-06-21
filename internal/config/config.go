package config

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/google/uuid"
)

const (
	EnvDev  string = "dev"
	EnvProd string = "prod"
	EnvTest string = "test"
)

var Environment = func() string {
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
		"STREMTHRU_DATA_DIR": os.TempDir(),
	},
	"": {
		"STREMTHRU_BASE_URL":                       "http://localhost:8080",
		"STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT": "*:0",
		"STREMTHRU_DATABASE_URI":                   "sqlite://./data/stremthru.db",
		"STREMTHRU_DATA_DIR":                       "./data",
		"STREMTHRU_LANDING_PAGE":                   "{}",
		"STREMTHRU_LOG_FORMAT":                     "json",
		"STREMTHRU_LOG_LEVEL":                      "INFO",
		"STREMTHRU_PORT":                           "8080",
		"STREMTHRU_STORE_CONTENT_PROXY":            "*:true",
		"STREMTHRU_STORE_TUNNEL":                   "*:true",
		"STREMTHRU_STORE_CLIENT_USER_AGENT":        "stremthru",
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

type UserPasswordMap map[string]string

func (m UserPasswordMap) GetPassword(user string) string {
	if password, ok := m[user]; ok {
		return password
	}
	return ""
}

type AuthAdminMap map[string]bool

func (m AuthAdminMap) IsAdmin(userName string) bool {
	if isAdmin, ok := m[userName]; ok {
		return isAdmin
	}
	return false
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
	FeatureStremioP2P      string = "stremio_p2p"
	FeatureStremioSidekick string = "stremio_sidekick"
	FeatureStremioStore    string = "stremio_store"
	FeatureStremioTorz     string = "stremio_torz"
	FeatureStremioWrap     string = "stremio_wrap"
)

var features = []string{
	FeatureAnime,
	FeatureDMMHashlist,
	FeatureIMDBTitle,
	FeatureStremioList,
	FeatureStremioP2P,
	FeatureStremioSidekick,
	FeatureStremioStore,
	FeatureStremioTorz,
	FeatureStremioWrap,
}

type FeatureConfig struct {
	enabled  []string
	disabled []string
}

func (f FeatureConfig) IsEnabled(name string) bool {
	if slices.Contains(f.disabled, name) {
		return false
	}

	if len(f.enabled) == 0 {
		return true
	}

	return slices.Contains(f.enabled, name)
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

type Config struct {
	LogLevel  slog.Level
	LogFormat string

	Port                        string
	StoreAuthToken              StoreAuthTokenMap
	ProxyAuthPassword           UserPasswordMap
	AuthAdmin                   AuthAdminMap
	AdminPassword               UserPasswordMap
	BuddyURL                    string
	HasBuddy                    bool
	PeerURL                     string
	PeerAuthToken               string
	HasPeer                     bool
	LazyPeer                    bool
	PullPeerURL                 string
	RedisURI                    string
	DatabaseURI                 string
	Feature                     FeatureConfig
	Version                     string
	LandingPage                 string
	ServerStartTime             time.Time
	StoreContentProxy           StoreContentProxyMap
	StoreContentCachedStaleTime storeContentCachedStaleTimeMap
	StoreClientUserAgent        string
	ContentProxyConnectionLimit ContentProxyConnectionLimitMap
	IP                          *IPResolver

	DataDir string
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
	proxyAuthCredList := strings.FieldsFunc(getEnv("STREMTHRU_PROXY_AUTH"), func(c rune) bool {
		return c == ','
	})
	proxyAuthPasswordMap := make(UserPasswordMap)

	for _, cred := range proxyAuthCredList {
		if basicAuth, err := core.ParseBasicAuth(cred); err == nil {
			proxyAuthPasswordMap[basicAuth.Username] = basicAuth.Password
		}
	}

	authAdminMap := AuthAdminMap{}
	authAdminList := strings.FieldsFunc(getEnv("STREMTHRU_AUTH_ADMIN"), func(c rune) bool {
		return c == ','
	})
	adminPasswordMap := UserPasswordMap{}
	for _, admin := range authAdminList {
		if strings.Contains(admin, ":") {
			username, password, _ := strings.Cut(admin, ":")
			adminPasswordMap[username] = password
		} else {
			authAdminMap[admin] = true
		}
	}
	if len(authAdminMap) == 0 {
		for username := range proxyAuthPasswordMap {
			authAdminMap[username] = true
		}
	}
	if len(adminPasswordMap) == 0 {
		username := "st-" + util.GenerateRandomString(7, util.CharSet.AlphaNumeric)
		password := util.GenerateRandomString(27, util.CharSet.AlphaNumericMixedCase)
		adminPasswordMap[username] = password
	}

	storeAlldebridTokenList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_AUTH"), func(c rune) bool {
		return c == ','
	})
	storeAuthTokenMap := make(StoreAuthTokenMap)
	for _, userStoreToken := range storeAlldebridTokenList {
		if user, storeToken, ok := strings.Cut(userStoreToken, ":"); ok {
			if store, token, ok := strings.Cut(storeToken, ":"); ok {
				storeAuthTokenMap.addStore(user, store)
				storeAuthTokenMap.setToken(user, store, token)
			}
		}
	}

	buddyUrl, _ := parseUri(getEnv("STREMTHRU_BUDDY_URI"))
	pullPeerUrl := ""
	if buddyUrl != "" {
		pullPeerUrl, _ = parseUri(getEnv("STREMTHRU__PULL__PEER_URI"))
	}

	defaultPeerUri := ""
	if peerUri, err := core.Base64Decode("aHR0cHM6Ly9zdHJlbXRocnUuMTMzNzcwMDEueHl6"); err == nil && buddyUrl == "" {
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

	var logLevel slog.Level
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

	lazyPeer := strings.ToLower(getEnv("STREMTHRU_LAZY_PEER"))

	return Config{
		LogLevel:  logLevel,
		LogFormat: logFormat,

		Port:                        getEnv("STREMTHRU_PORT"),
		ProxyAuthPassword:           proxyAuthPasswordMap,
		AuthAdmin:                   authAdminMap,
		AdminPassword:               adminPasswordMap,
		StoreAuthToken:              storeAuthTokenMap,
		BuddyURL:                    buddyUrl,
		HasBuddy:                    len(buddyUrl) > 0,
		PeerURL:                     peerUrl,
		PeerAuthToken:               peerAuthToken,
		HasPeer:                     len(peerUrl) > 0,
		LazyPeer:                    lazyPeer == "1" || lazyPeer == "true",
		PullPeerURL:                 pullPeerUrl,
		RedisURI:                    getEnv("STREMTHRU_REDIS_URI"),
		DatabaseURI:                 databaseUri,
		Feature:                     feature,
		Version:                     "0.79.3", // x-release-please-version
		LandingPage:                 getEnv("STREMTHRU_LANDING_PAGE"),
		ServerStartTime:             time.Now(),
		StoreContentProxy:           storeContentProxyMap,
		StoreContentCachedStaleTime: storeContentCachedStaleTimeMap,
		StoreClientUserAgent:        getEnv("STREMTHRU_STORE_CLIENT_USER_AGENT"),
		ContentProxyConnectionLimit: contentProxyConnectionMap,
		IP:                          &IPResolver{},

		DataDir: dataDir,
	}
}()

var LogLevel = config.LogLevel
var LogFormat = config.LogFormat

var Port = config.Port
var ProxyAuthPassword = config.ProxyAuthPassword
var AuthAdmin = config.AuthAdmin
var AdminPassword = config.AdminPassword
var StoreAuthToken = config.StoreAuthToken
var BuddyURL = config.BuddyURL
var HasBuddy = config.HasBuddy
var PeerURL = config.PeerURL
var PeerAuthToken = config.PeerAuthToken
var HasPeer = config.HasPeer
var LazyPeer = config.LazyPeer
var PullPeerURL = config.PullPeerURL
var RedisURI = config.RedisURI
var DatabaseURI = config.DatabaseURI
var Feature = config.Feature
var Version = config.Version
var LandingPage = config.LandingPage
var ServerStartTime = config.ServerStartTime
var StoreContentProxy = config.StoreContentProxy
var StoreContentCachedStaleTime = config.StoreContentCachedStaleTime
var StoreClientUserAgent = config.StoreClientUserAgent
var ContentProxyConnectionLimit = config.ContentProxyConnectionLimit
var InstanceId = strings.ReplaceAll(uuid.NewString(), "-", "")
var IP = config.IP

var DataDir = config.DataDir

var IsPublicInstance = len(ProxyAuthPassword) == 0

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
	hasTunnel := Tunnel.hasProxy()
	defaultProxyHost := Tunnel.GetDefaultProxyHost()

	machineIP := IP.GetMachineIP()
	var tunnelIpByProxyHost map[string]string
	if hasTunnel {
		ipMap, err := IP.GetTunnelIPByProxyHost()
		if err != nil {
			if defaultProxyHost != "" && ipMap[defaultProxyHost] == "" {
				log.Panicf("Failed to resolve Tunnel IP Map: %v\n", err)
			} else {
				log.Printf("Failed to resolve Tunnel IP Map: %v\n\n", err)
			}
		}
		tunnelIpByProxyHost = ipMap
	}

	l := log.New(os.Stderr, "=", 0)
	l.Println("====== StremThru =======")
	l.Printf(" Time: %v\n", ServerStartTime.Format(time.RFC3339))
	l.Printf(" Version: %v\n", Version)
	l.Printf(" Port: %v\n", Port)
	l.Println("========================")
	l.Println()

	l.Printf("  Log Level: %s\n", LogLevel.String())
	l.Printf(" Log Format: %s\n", LogFormat)
	l.Println()

	if hasTunnel {
		l.Println(" Tunnel:")
		if defaultProxy := Tunnel.getProxy("*"); defaultProxy != nil && defaultProxy.Host != "" {
			defaultProxyConfig := ""
			if noProxy := getEnv("NO_PROXY"); noProxy == "*" {
				defaultProxyConfig = " (disabled)"
			}
			l.Println("   Default: " + defaultProxy.Redacted() + defaultProxyConfig)
			l.Println("   [Store]: " + defaultProxy.Redacted())
		}

		if len(Tunnel) > 1 {
			l.Println("   By Host:")
			for hostname, proxy := range Tunnel {
				if hostname == "*" {
					continue
				}

				if proxy.Host == "" {
					if defaultProxyHost != "" {
						l.Println("     " + hostname + ": (disabled)")
					}
				} else {
					l.Println("     " + hostname + ": " + proxy.Redacted())
				}
			}
		}

		l.Println()
	}

	l.Println(" Machine IP: " + machineIP)
	if hasTunnel {
		l.Println("  Tunnel IP: ")
		for proxyHost, tunnelIp := range tunnelIpByProxyHost {
			if tunnelIp == "" {
				tunnelIp = "(unresolved)"
			}
			l.Println("    [" + proxyHost + "]: " + tunnelIp)
		}
	}
	l.Println()

	l.Printf("   Base URL: %s\n", BaseURL.String())
	l.Println()

	if !IsPublicInstance {
		l.Println(" Users:")
		for user := range ProxyAuthPassword {
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
			l.Println("   - " + user)
			l.Println("       store: " + strings.Join(stores, ","))
			if cpcl := ContentProxyConnectionLimit.Get(user); cpcl > 0 {
				l.Println("       content_proxy_connection_limit: " + strconv.FormatUint(uint64(cpcl), 10))
			}
		}
		l.Println()
	}

	l.Println(" Stores:")
	for _, store := range state.StoreNames {
		storeConfig := ""
		if !IsPublicInstance && StoreContentProxy.IsEnabled(string(store)) {
			storeConfig += "content_proxy"
		}
		if hasTunnel {
			if StoreTunnel.isEnabledForAPI(string(store)) {
				if storeConfig != "" {
					storeConfig += ","
				}
				storeConfig += "tunnel:api"
				if !IsPublicInstance && StoreTunnel.GetTypeForStream(string(store)) == TUNNEL_TYPE_FORCED {
					storeConfig += "+stream"
				}
			}
		}
		if storeConfig != "" {
			storeConfig = " (" + storeConfig + ")"
		}
		l.Println("   - " + string(store) + storeConfig)
	}
	l.Println()

	if len(AdminPassword) == 1 {
		for username, password := range AdminPassword {
			if strings.HasPrefix(username, "st-") {
				l.Println(" (Auto Generated) Admin Creds:")
				l.Println("   " + username + ":" + password)
				l.Println()
			}
		}
	}

	if HasBuddy {
		l.Println(" Buddy URI:")
		l.Println("   " + BuddyURL)
		l.Println()
	}

	if HasPeer {
		u, err := url.Parse(PeerURL)
		if err != nil {
			l.Panicf(" Invalid Peer URI: %v\n", err)
		}
		u.User = url.UserPassword("", PeerAuthToken)
		lazyPeer := ""
		if LazyPeer {
			lazyPeer = " (lazy)"
		}
		l.Println(" Peer URI" + lazyPeer + ":")
		l.Println("   " + u.Redacted())
		l.Println()
	}
	if PullPeerURL != "" {
		u, err := url.Parse(PullPeerURL)
		if err != nil {
			l.Panicf(" Invalid (Pull) Peer URI: %v\n", err)
		}
		l.Println(" (Pull) Peer URI:")
		l.Println("   " + u.Redacted())
		l.Println()
	}

	if RedisURI != "" {
		uri, err := getRedactedURI(RedisURI)
		if err != nil {
			l.Panicf(" Invalid Redis URI: %v\n", err)
		}
		l.Println(" Redis URI:")
		l.Println("   " + uri)
		l.Println()
	}

	uri, err := getRedactedURI(DatabaseURI)
	if err != nil {
		l.Panicf(" Invalid Database URI: %v\n", err)
	}
	l.Println(" Database URI:")
	l.Println("   " + uri)
	l.Println()

	l.Println(" Features:")
	for _, feature := range features {
		disabled := ""
		if !Feature.IsEnabled(feature) {
			disabled = " (disabled)"
		}
		l.Println("   - " + feature + disabled)
	}
	l.Println()

	l.Println(" Integrations:")
	for _, integration := range []string{"anilist.co", "trakt.tv", "kitsu.app"} {
		switch integration {
		case "anilist.co":
			disabled := ""
			if !Feature.IsEnabled(FeatureAnime) {
				disabled = " (disabled)"
			}
			l.Println("   - " + integration + disabled)
		case "kitsu.app":
			disabled := ""
			if !Feature.IsEnabled(FeatureAnime) || !Integration.Kitsu.HasDefaultCredentials() {
				disabled = " (disabled)"
			}
			l.Println("   - " + integration + disabled)
			if disabled == "" {
				if Integration.Kitsu.ClientId != "" {
					l.Println("           client_id: " + Integration.Kitsu.ClientId[0:3] + "..." + Integration.Kitsu.ClientId[len(Integration.Trakt.ClientId)-3:])
				}
				if Integration.Kitsu.ClientSecret != "" {
					l.Println("       client_secret: " + Integration.Kitsu.ClientSecret[0:3] + "..." + Integration.Kitsu.ClientSecret[len(Integration.Trakt.ClientSecret)-3:])
				}
				l.Println("               email: " + Integration.Kitsu.Email)
				l.Println("            password: " + "*******")
			}
		case "trakt.tv":
			disabled := ""
			if !Integration.Trakt.IsEnabled() {
				disabled = " (disabled)"
			}
			l.Println("   - " + integration + disabled)
			if disabled == "" {
				l.Println("           client_id: " + Integration.Trakt.ClientId[0:3] + "..." + Integration.Trakt.ClientId[len(Integration.Trakt.ClientId)-3:])
				l.Println("       client_secret: " + Integration.Trakt.ClientSecret[0:3] + "..." + Integration.Trakt.ClientSecret[len(Integration.Trakt.ClientSecret)-3:])
			}
		}
	}
	l.Println()

	l.Println(" Instance ID:")
	l.Println("   " + InstanceId)
	l.Println()

	l.Println(" Data Directory:")
	l.Println("   " + DataDir)
	l.Println()

	l.Print("========================\n\n")
}
