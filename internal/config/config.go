package config

import (
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
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
	names := []string{}
	if um, ok := m[user]; ok {
		for name := range um {
			if name != "*" {
				names = append(names, name)
			}
		}
	}
	return names
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

type Config struct {
	Port              string
	StoreAuthToken    StoreAuthTokenMap
	ProxyAuthPassword ProxyAuthPasswordMap
	BuddyURL          string
	HasBuddy          bool
	PeerURL           string
	PeerAuthToken     string
	HasPeer           bool
	RedisURI          string
	DatabaseURI       string
	StremioAddon      StremioAddonConfig
	Version           string
	LandingPage       string
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
		enabled: strings.FieldsFunc(strings.TrimSpace(getEnv("STREMTHRU_STREMIO_ADDON", "")), func(c rune) bool {
			return c == ','
		}),
	}

	return Config{
		Port:              getEnv("STREMTHRU_PORT", "8080"),
		ProxyAuthPassword: proxyAuthPasswordMap,
		StoreAuthToken:    storeAuthTokenMap,
		BuddyURL:          buddyUrl,
		HasBuddy:          len(buddyUrl) > 0,
		PeerURL:           peerUrl,
		PeerAuthToken:     peerAuthToken,
		HasPeer:           len(peerUrl) > 0,
		RedisURI:          getEnv("STREMTHRU_REDIS_URI", ""),
		DatabaseURI:       databaseUri,
		StremioAddon:      stremioAddon,
		Version:           "0.14.0", // x-release-please-version
		LandingPage:       getEnv("STREMTHRU_LANDING_PAGE", "{}"),
	}
}()

var Port = config.Port
var ProxyAuthPassword = config.ProxyAuthPassword
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
