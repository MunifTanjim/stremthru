package config

import (
	"log"
	"net/url"
	"strings"
	"time"
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

type IntegrationConfig struct {
	AniList integrationConfigAniList
	GitHub  integrationConfigGitHub
	MDBList integrationConfigMDBList
	Trakt   integrationConfigTrakt
	Kitsu   integrationConfigKitsu
}

func parseIntegration() IntegrationConfig {
	integration := IntegrationConfig{
		AniList: integrationConfigAniList{
			ListStaleTime: mustParseDuration("anilist list stale time", getEnv("STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME"), 15*time.Minute),
		},
		GitHub: integrationConfigGitHub{
			User:  getEnv("STREMTHRU_INTEGRATION_GITHUB_USER"),
			Token: getEnv("STREMTHRU_INTEGRATION_GITHUB_TOKEN"),
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
	}
	if integration.Kitsu.Email != "" && !strings.Contains(integration.Kitsu.Email, "@") {
		log.Panicf("Invalid Kitsu Email: %s\n", integration.Kitsu.Email)
	}
	return integration
}

var Integration = parseIntegration()
