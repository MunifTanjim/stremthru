package config

import (
	"log"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/google/uuid"
)

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && len(value) > 0 {
		return value
	}
	return defaultValue
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

var stremioAddons = []string{StremioAddonSidekick, StremioAddonStore, StremioAddonWrap}

type StremioAddonConfig struct {
	enabled []string
}

func (sa StremioAddonConfig) IsEnabled(name string) bool {
	if len(sa.enabled) == 0 {
		return true
	}

	return slices.Contains(sa.enabled, name)
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
	PullPeerURL                 string
	RedisURI                    string
	DatabaseURI                 string
	StremioAddon                StremioAddonConfig
	Version                     string
	LandingPage                 string
	ServerStartTime             time.Time
	StoreContentProxy           StoreContentProxyMap
	ContentProxyConnectionLimit ContentProxyConnectionLimitMap
	IP                          *IPResolver
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
	proxyAuthCredList := strings.FieldsFunc(getEnv("STREMTHRU_PROXY_AUTH", ""), func(c rune) bool {
		return c == ','
	})
	proxyAuthPasswordMap := make(UserPasswordMap)

	for _, cred := range proxyAuthCredList {
		if basicAuth, err := core.ParseBasicAuth(cred); err == nil {
			proxyAuthPasswordMap[basicAuth.Username] = basicAuth.Password
		}
	}

	authAdminMap := AuthAdminMap{}
	authAdminList := strings.FieldsFunc(getEnv("STREMTHRU_AUTH_ADMIN", ""), func(c rune) bool {
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

	storeAlldebridTokenList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_AUTH", ""), func(c rune) bool {
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

	buddyUrl, _ := parseUri(getEnv("STREMTHRU_BUDDY_URI", ""))
	pullPeerUrl := ""
	if buddyUrl != "" {
		pullPeerUrl, _ = parseUri(getEnv("STREMTHRU__PULL__PEER_URI", ""))
	}

	defaultPeerUri := ""
	if peerUri, err := core.Base64Decode("aHR0cHM6Ly9zdHJlbXRocnUuMTMzNzcwMDEueHl6"); err == nil && buddyUrl == "" {
		defaultPeerUri = peerUri
	}
	peerUrl, peerAuthToken := parseUri(getEnv("STREMTHRU_PEER_URI", defaultPeerUri))

	databaseUri := getEnv("STREMTHRU_DATABASE_URI", "sqlite://./data/stremthru.db")

	stremioAddon := StremioAddonConfig{
		enabled: strings.FieldsFunc(strings.TrimSpace(getEnv("STREMTHRU_STREMIO_ADDON", strings.Join(stremioAddons, ","))), func(c rune) bool {
			return c == ','
		}),
	}

	storeContentProxyList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_CONTENT_PROXY", "*:true"), func(c rune) bool {
		return c == ','
	})

	storeContentProxyMap := make(StoreContentProxyMap)
	for _, storeContentProxy := range storeContentProxyList {
		if store, enabled, ok := strings.Cut(storeContentProxy, ":"); ok {
			storeContentProxyMap[store] = enabled == "true"
		}
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(getEnv("STREMTHRU_LOG_LEVEL", "INFO"))); err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}

	logFormat := getEnv("STREMTHRU_LOG_FORMAT", "json")
	if logFormat != "json" && logFormat != "text" {
		log.Fatalf("Invalid log format: %s, expected: json / text", logFormat)
	}

	contentProxyConnectionMap := make(ContentProxyConnectionLimitMap)
	contentProxyConnectionList := strings.FieldsFunc(getEnv("STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT", "*:0"), func(c rune) bool {
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

	return Config{
		LogLevel:  logLevel,
		LogFormat: logFormat,

		Port:                        getEnv("STREMTHRU_PORT", "8080"),
		ProxyAuthPassword:           proxyAuthPasswordMap,
		AuthAdmin:                   authAdminMap,
		AdminPassword:               adminPasswordMap,
		StoreAuthToken:              storeAuthTokenMap,
		BuddyURL:                    buddyUrl,
		HasBuddy:                    len(buddyUrl) > 0,
		PeerURL:                     peerUrl,
		PeerAuthToken:               peerAuthToken,
		HasPeer:                     len(peerUrl) > 0,
		PullPeerURL:                 pullPeerUrl,
		RedisURI:                    getEnv("STREMTHRU_REDIS_URI", ""),
		DatabaseURI:                 databaseUri,
		StremioAddon:                stremioAddon,
		Version:                     "0.66.2", // x-release-please-version
		LandingPage:                 getEnv("STREMTHRU_LANDING_PAGE", "{}"),
		ServerStartTime:             time.Now(),
		StoreContentProxy:           storeContentProxyMap,
		ContentProxyConnectionLimit: contentProxyConnectionMap,
		IP:                          &IPResolver{},
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
var PullPeerURL = config.PullPeerURL
var RedisURI = config.RedisURI
var DatabaseURI = config.DatabaseURI
var StremioAddon = config.StremioAddon
var Version = config.Version
var LandingPage = config.LandingPage
var ServerStartTime = config.ServerStartTime
var StoreContentProxy = config.StoreContentProxy
var ContentProxyConnectionLimit = config.ContentProxyConnectionLimit
var InstanceId = strings.ReplaceAll(uuid.NewString(), "-", "")
var IP = config.IP

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
	hasTunnel := false
	if proxy := Tunnel.getProxy("*"); proxy != nil && proxy.Host != "" {
		hasTunnel = true
	}

	machineIP := IP.GetMachineIP()
	tunnelIP := ""
	if hasTunnel {
		ip, err := IP.GetTunnelIP()
		if err != nil {
			log.Panicf("Failed to resolve Tunnel IP: %v\n", err)
		}
		tunnelIP = ip
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
		if defaultProxy := Tunnel.getProxy("*"); defaultProxy != nil {
			defaultProxyConfig := ""
			if noProxy := getEnv("NO_PROXY", ""); noProxy == "*" {
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
					l.Println("     " + hostname + ": (disabled)")
				} else {
					l.Println("     " + hostname + ": " + proxy.Redacted())
				}
			}
		}

		l.Println()
	}

	l.Println(" Machine IP: " + machineIP)
	if hasTunnel {
		l.Println("  Tunnel IP: " + tunnelIP)
	}
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
		l.Println(" Peer URI:")
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

	if len(StremioAddon.enabled) > 0 {
		l.Println(" Stremio Addons:")
		for _, addon := range StremioAddon.enabled {
			l.Println("   - " + addon)
		}
		l.Println()
	}

	l.Println(" Instance ID:")
	l.Println("   " + InstanceId)
	l.Println()

	l.Print("========================\n\n")
}
