package config

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
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
	if store == "" {
		store = m.GetToken("*", "*")
	}
	return store
}

func (m StoreAuthTokenMap) ListStores(user string) []string {
	stores := []string{}
	if um, ok := m[user]; ok {
		for store := range um {
			if store != "*" {
				stores = append(stores, store)
			}
		}
	}
	return stores
}

func (m StoreAuthTokenMap) setPreferredStore(user, store string) {
	if m.GetPreferredStore(user) == "" {
		m.setToken(user, "*", store)
	}
}

type ProxyAuthPasswordMap map[string]string

func (m ProxyAuthPasswordMap) GetPassword(userName string) string {
	if token, ok := m[userName]; ok {
		return token
	}
	return ""
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

	for _, addon := range sa.enabled {
		if addon == name {
			return true
		}
	}
	return false
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

type StoreTunnelConfig struct {
	api    bool
	stream bool
}

type StoreTunnelConfigMap map[string]StoreTunnelConfig

func (stc StoreTunnelConfigMap) IsEnabledForAPI(name string) bool {
	if c, ok := stc[name]; ok {
		return c.api
	}
	if name != "*" {
		return stc.IsEnabledForAPI("*")
	}
	return true
}

func (stc StoreTunnelConfigMap) IsEnabledForStream(name string) bool {
	if c, ok := stc[name]; ok {
		return c.stream
	}
	if name != "*" {
		return stc.IsEnabledForStream("*")
	}
	return true
}

func getIp(client *http.Client) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://checkip.amazonaws.com", nil)
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

type IPResolver struct {
	machineIP string
}

func (ipr *IPResolver) GetMachineIP() string {
	if ipr.machineIP == "" {
		client := request.GetHTTPClient(false)
		ip, err := getIp(client)
		if err != nil {
			log.Panicf("Failed to detect Machine IP: %v\n", err)
		}
		ipr.machineIP = ip
	}
	return ipr.machineIP
}

func (ipr *IPResolver) GetTunnelIP() (string, error) {
	ip, err := getIp(request.DefaultHTTPClient)
	if err != nil {
		return "", err
	}
	return ip, nil
}

type Config struct {
	Port               string
	StoreAuthToken     StoreAuthTokenMap
	ProxyAuthPassword  ProxyAuthPasswordMap
	ProxyStreamEnabled bool
	BuddyURL           string
	HasBuddy           bool
	PeerURL            string
	PeerAuthToken      string
	HasPeer            bool
	RedisURI           string
	DatabaseURI        string
	StremioAddon       StremioAddonConfig
	Version            string
	LandingPage        string
	ServerStartTime    time.Time
	StoreContentProxy  StoreContentProxyMap
	StoreTunnel        StoreTunnelConfigMap
	IP                 *IPResolver
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
	if value := getEnv("STREMTHRU_HTTP_PROXY", ""); len(value) > 0 {
		if err := os.Setenv("HTTP_PROXY", value); err != nil {
			log.Fatal("failed to set http proxy")
		}
	}

	if value := getEnv("STREMTHRU_HTTPS_PROXY", ""); len(value) > 0 {
		if err := os.Setenv("HTTPS_PROXY", value); err != nil {
			log.Fatal("failed to set https proxy")
		}
	}

	proxyAuthCredList := strings.FieldsFunc(getEnv("STREMTHRU_PROXY_AUTH", ""), func(c rune) bool {
		return c == ','
	})
	proxyAuthPasswordMap := make(ProxyAuthPasswordMap)
	for _, cred := range proxyAuthCredList {
		if basicAuth, err := core.ParseBasicAuth(cred); err == nil {
			proxyAuthPasswordMap[basicAuth.Username] = basicAuth.Password
		}
	}

	storeAlldebridTokenList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_AUTH", ""), func(c rune) bool {
		return c == ','
	})
	storeAuthTokenMap := make(StoreAuthTokenMap)
	for _, userStoreToken := range storeAlldebridTokenList {
		if user, storeToken, ok := strings.Cut(userStoreToken, ":"); ok {
			if store, token, ok := strings.Cut(storeToken, ":"); ok {
				storeAuthTokenMap.setPreferredStore(user, store)
				storeAuthTokenMap.setToken(user, store, token)
			}
		}
	}

	buddyUrl, _ := parseUri(getEnv("STREMTHRU_BUDDY_URI", ""))
	peerUrl, peerAuthToken := parseUri(getEnv("STREMTHRU_PEER_URI", ""))

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

	storeTunnelList := strings.FieldsFunc(getEnv("STREMTHRU_STORE_TUNNEL", "*:true"), func(c rune) bool {
		return c == ','
	})

	storeTunnelMap := make(StoreTunnelConfigMap)
	for _, storeTunnel := range storeTunnelList {
		if store, tunnel, ok := strings.Cut(storeTunnel, ":"); ok {
			storeTunnelMap[store] = StoreTunnelConfig{
				api:    tunnel == "true" || tunnel == "api",
				stream: tunnel == "true",
			}
		}
	}

	return Config{
		Port:               getEnv("STREMTHRU_PORT", "8080"),
		ProxyAuthPassword:  proxyAuthPasswordMap,
		ProxyStreamEnabled: len(proxyAuthPasswordMap) > 0,
		StoreAuthToken:     storeAuthTokenMap,
		BuddyURL:           buddyUrl,
		HasBuddy:           len(buddyUrl) > 0,
		PeerURL:            peerUrl,
		PeerAuthToken:      peerAuthToken,
		HasPeer:            len(peerUrl) > 0,
		RedisURI:           getEnv("STREMTHRU_REDIS_URI", ""),
		DatabaseURI:        databaseUri,
		StremioAddon:       stremioAddon,
		Version:            "0.38.0", // x-release-please-version
		LandingPage:        getEnv("STREMTHRU_LANDING_PAGE", "{}"),
		ServerStartTime:    time.Now(),
		StoreContentProxy:  storeContentProxyMap,
		StoreTunnel:        storeTunnelMap,
		IP:                 &IPResolver{},
	}
}()

var Port = config.Port
var ProxyAuthPassword = config.ProxyAuthPassword
var ProxyStreamEnabled = config.ProxyStreamEnabled
var StoreAuthToken = config.StoreAuthToken
var BuddyURL = config.BuddyURL
var HasBuddy = config.HasBuddy
var PeerURL = config.PeerURL
var PeerAuthToken = config.PeerAuthToken
var HasPeer = config.HasPeer
var RedisURI = config.RedisURI
var DatabaseURI = config.DatabaseURI
var StremioAddon = config.StremioAddon
var Version = config.Version
var LandingPage = config.LandingPage
var ServerStartTime = config.ServerStartTime
var StoreContentProxy = config.StoreContentProxy
var StoreTunnel = config.StoreTunnel
var IP = config.IP

func getRedactedURI(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	return u.Redacted(), nil
}

func PrintConfig() {
	httpProxy, httpsProxy := getEnv("HTTP_PROXY", ""), getEnv("HTTPS_PROXY", "")
	hasTunnel := httpProxy != "" || httpsProxy != ""

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

	if hasTunnel {
		l.Println(" Tunnel:")
		if httpProxy != "" {
			l.Println("    HTTP: " + httpProxy)
		}
		if httpsProxy != "" {
			l.Println("   HTTPS: " + httpsProxy)
		}
		l.Println()
	}

	l.Println(" Machine IP: " + machineIP)
	if hasTunnel {
		l.Println("  Tunnel IP: " + tunnelIP)
	}
	l.Println()

	usersCount := len(ProxyAuthPassword)
	if usersCount > 0 {
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
			storeConfig := " (store:" + strings.Join(stores, ",") + ")"
			l.Println("   - " + user + storeConfig)
		}
		l.Println()
	}

	l.Println(" Stores:")
	for _, store := range []store.StoreName{
		store.StoreNameAlldebrid,
		store.StoreNameDebridLink,
		store.StoreNameEasyDebrid,
		store.StoreNameOffcloud,
		store.StoreNamePikPak,
		store.StoreNamePremiumize,
		store.StoreNameRealDebrid,
		store.StoreNameTorBox,
	} {
		storeConfig := ""
		if usersCount > 0 && StoreContentProxy.IsEnabled(string(store)) {
			storeConfig += "content_proxy"
		}
		if hasTunnel {
			if StoreTunnel.IsEnabledForAPI(string(store)) {
				if storeConfig != "" {
					storeConfig += ","
				}
				storeConfig += "tunnel:api"
				if usersCount > 0 && StoreTunnel.IsEnabledForStream(string(store)) {
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

	if RedisURI != "" {
		uri, err := getRedactedURI(RedisURI)
		if err != nil {
			l.Panicf(" Invalid Redis URI: %v\n", err)
		}
		l.Println(" Redis URI:")
		l.Println("  " + uri)
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

	l.Println("========================\n")
}
