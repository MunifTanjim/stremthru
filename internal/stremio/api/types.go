package stremio_api

import (
	"time"

	"github.com/MunifTanjim/stremthru/stremio"
)

type UserGDPRConsent struct {
	Marketing bool   `json:"marketing"`
	Privacy   bool   `json:"privacy"`
	TOS       bool   `json:"tos"`
	From      string `json:"from"`
}

type User struct {
	Id             string          `json:"_id"`
	Email          string          `json:"email"`
	FBId           string          `json:"fbId"`
	Fullname       string          `json:"fullname"`
	Avatar         string          `json:"avatar"`
	Anonymous      bool            `json:"anonymous"`
	GDPRConsent    UserGDPRConsent `json:"gdpr_consent"`
	Taste          interface{}     `json:"taste"`
	Lang           string          `json:"lang"`
	DateRegistered time.Time       `json:"dateRegistered"`
	LastModified   time.Time       `json:"lastModified"`
	Trakt          interface{}     `json:"trakt"`
	StremioAddons  string          `json:"stremio_addons"`
	PremiumExpire  time.Time       `json:"premium_expire"`
}

type LoginData struct {
	AuthKey string `json:"authKey"`
	User    User   `json:"user"`
}

type GetAddonsData struct {
	Addons       []stremio.Addon `json:"addons"`
	LastModified time.Time       `json:"lastModified"`
}

type SetAddonsData struct {
	Success bool `json:"success"`
}

type LibraryItemBehaviorHints struct {
	DefaultVideoId     string `json:"defaultVideoId"`
	FeaturedVideoId    string `json:"featuredVideoId,omitempty"`
	HasScheduledVideos bool   `json:"hasScheduledVideos"`
}

type LibraryItemState struct {
	LastWatched        time.Time `json:"lastWatched"`
	TimeWatched        int       `json:"timeWatched"`
	TimeOffset         int       `json:"timeOffset"`
	OverallTimeWatched int       `json:"overallTimeWatched"`
	TimesWatched       int       `json:"timesWatched"`
	FlaggedWatched     int       `json:"flaggedWatched"`
	Duration           int       `json:"duration"`
	VideoId            string    `json:"video_id"`
	Watched            string    `json:"watched"`
	NoNotif            bool      `json:"noNotif"`
	Season             int       `json:"season,omitempty"`
	Episode            int       `json:"episode,omitempty"`
}

type LibraryItem struct {
	Id          string                  `json:"_id"`
	Removed     bool                    `json:"removed"`
	Temp        bool                    `json:"temp"`
	CTime       time.Time               `json:"_ctime"`
	MTime       time.Time               `json:"_mtime"`
	State       LibraryItemState        `json:"state"`
	Name        string                  `json:"name"`
	Type        string                  `json:"type"`
	Poster      string                  `json:"poster"`
	PosterShape stremio.MetaPosterShape `json:"posterShape,omitempty"`
	Background  string                  `json:"background,omitempty"`
	Logo        string                  `json:"logo,omitempty"`
	Year        string                  `json:"year,omitempty"`

	BehaviorHints *LibraryItemBehaviorHints `json:"behaviorHints,omitempty"`
}

type GetAllLibraryItemsData []LibraryItem

type UpdateLibraryItemsData struct {
	Success bool `json:"success"`
}
