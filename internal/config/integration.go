package config

import (
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/util"
)

var BaseURL = func() *url.URL {
	baseUrl, err := url.Parse(getEnv("STREMTHRU_BASE_URL"))
	if err != nil {
		log.Panicf("Invalid Base URL: %v\n", err)
	}
	return baseUrl
}()

type integrationConfigAniList struct {
	ListStaleTime time.Duration
}

type integrationConfigBitmagnet struct {
	BaseURL     *url.URL
	DatabaseURI string
}

func (c integrationConfigBitmagnet) IsEnabled() bool {
	return c.BaseURL != nil && c.DatabaseURI != ""
}

type integrationConfigLettterboxd struct {
	ClientId      string
	ClientSecret  string
	UserAgent     string
	ListStaleTime time.Duration
}

func (c integrationConfigLettterboxd) IsEnabled() bool {
	return c.ClientId != "" && c.ClientSecret != ""
}

func (c integrationConfigLettterboxd) IsPiggybacked() bool {
	return !c.IsEnabled() && HasPeer
}

type integrationConfigMDBList struct {
	ListStaleTime time.Duration
}

type integrationConfigTrakt struct {
	ClientId      string
	ClientSecret  string
	ListStaleTime time.Duration
}

func (c integrationConfigTrakt) IsEnabled() bool {
	return c.ClientId != "" && c.ClientSecret != ""
}

type integrationConfigKitsu struct {
	ClientId     string
	ClientSecret string
	Email        string
	Password     string
}

func (c integrationConfigKitsu) HasDefaultCredentials() bool {
	return c.Email != "" && c.Password != ""
}

type integrationConfigGitHub struct {
	User  string
	Token string
}

func (c integrationConfigGitHub) HasDefaultCredentials() bool {
	return c.User != "" && c.Token != ""
}

type integrationConfigTMDB struct {
	AccessToken   string
	ListStaleTime time.Duration
}

func (c integrationConfigTMDB) IsEnabled() bool {
	return c.AccessToken != ""
}

type integrationConfigTVDB struct {
	APIKey             string
	ListStaleTime      time.Duration
	SystemOAuthTokenId string
}

func (c integrationConfigTVDB) IsEnabled() bool {
	return c.APIKey != ""
}

type IntegrationConfig struct {
	AniList    integrationConfigAniList
	Bitmagnet  integrationConfigBitmagnet
	GitHub     integrationConfigGitHub
	Letterboxd integrationConfigLettterboxd
	MDBList    integrationConfigMDBList
	Trakt      integrationConfigTrakt
	Kitsu      integrationConfigKitsu
	TMDB       integrationConfigTMDB
	TVDB       integrationConfigTVDB
}

func parseIntegration() IntegrationConfig {
	bitmagnet := integrationConfigBitmagnet{
		DatabaseURI: getEnv("STREMTHRU_INTEGRATION_BITMAGNET_DATABASE_URI"),
	}

	if bitmagnetBaseUrl := getEnv("STREMTHRU_INTEGRATION_BITMAGNET_BASE_URL"); bitmagnetBaseUrl != "" {
		bitmagnet.BaseURL = util.MustParseURL(bitmagnetBaseUrl)
	}

	integration := IntegrationConfig{
		AniList: integrationConfigAniList{
			ListStaleTime: mustParseDuration("anilist list stale time", getEnv("STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME"), 15*time.Minute),
		},
		Bitmagnet: bitmagnet,
		GitHub: integrationConfigGitHub{
			User:  getEnv("STREMTHRU_INTEGRATION_GITHUB_USER"),
			Token: getEnv("STREMTHRU_INTEGRATION_GITHUB_TOKEN"),
		},
		Letterboxd: integrationConfigLettterboxd{
			ClientId:      getEnv("STREMTHRU_INTEGRATION_LETTERBOXD_CLIENT_ID"),
			ClientSecret:  getEnv("STREMTHRU_INTEGRATION_LETTERBOXD_CLIENT_SECRET"),
			UserAgent:     getEnv("STREMTHRU_INTEGRATION_LETTERBOXD_USER_AGENT"),
			ListStaleTime: mustParseDuration("letterboxd list stale time", getEnv("STREMTHRU_INTEGRATION_LETTERBOXD_LIST_STALE_TIME"), 2*24*time.Hour),
		},
		MDBList: integrationConfigMDBList{
			ListStaleTime: mustParseDuration("mdblist list stale time", getEnv("STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME"), 15*time.Minute),
		},
		Trakt: integrationConfigTrakt{
			ClientId:      getEnv("STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID"),
			ClientSecret:  getEnv("STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET"),
			ListStaleTime: mustParseDuration("trakt list stale time", getEnv("STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME"), 15*time.Minute),
		},
		Kitsu: integrationConfigKitsu{
			ClientId:     getEnv("STREMTHRU_INTEGRATION_KITSU_CLIENT_ID"),
			ClientSecret: getEnv("STREMTHRU_INTEGRATION_KITSU_CLIENT_SECRET"),
			Email:        getEnv("STREMTHRU_INTEGRATION_KITSU_EMAIL"),
			Password:     getEnv("STREMTHRU_INTEGRATION_KITSU_PASSWORD"),
		},
		TMDB: integrationConfigTMDB{
			AccessToken:   getEnv("STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN"),
			ListStaleTime: mustParseDuration("tmdb list stale time", getEnv("STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME"), 15*time.Minute),
		},
		TVDB: integrationConfigTVDB{
			APIKey:             getEnv("STREMTHRU_INTEGRATION_TVDB_API_KEY"),
			ListStaleTime:      mustParseDuration("tvdb list stale time", getEnv("STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME"), 15*time.Minute),
			SystemOAuthTokenId: "system:tvdb",
		},
	}
	if integration.Kitsu.Email != "" && !strings.Contains(integration.Kitsu.Email, "@") {
		log.Panicf("Invalid Kitsu Email: %s\n", integration.Kitsu.Email)
	}
	return integration
}

var Integration = parseIntegration()
