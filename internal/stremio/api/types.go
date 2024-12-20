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

type AddonFlags struct {
	Official  bool `json:"official"`
	Protected bool `json:"protected"`
}

type Addon struct {
	TransportUrl  string           `json:"transportUrl"`
	TransportName string           `json:"transportName"`
	Manifest      stremio.Manifest `json:"manifest"`
	Flags         AddonFlags       `json:"flags"`
}

type GetAddonsData struct {
	Addons       []Addon   `json:"addons"`
	LastModified time.Time `json:"lastModified"`
}

type SetAddonsData struct {
	Success bool `json:"success"`
}
