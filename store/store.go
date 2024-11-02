package store

import (
	"context"
	"net/url"

	"github.com/MunifTanjim/stremthru/core"
)

type Ctx struct {
	APIKey  string
	Context context.Context
	Form    *url.Values
}

type StoreName string

const (
	StoreNameAlldebrid  StoreName = "alldebrid"
	StoreNamePremiumize StoreName = "premiumize"
)

func (sn StoreName) Validate() (StoreName, *core.StoreError) {
	if sn == StoreNameAlldebrid || sn == StoreNamePremiumize {
		return sn, nil
	}
	return sn, ErrorInvalidStoreName(string(sn))
}

type UserSubscriptionStatus string

const (
	UserSubscriptionStatusPremium UserSubscriptionStatus = "premium"
	UserSubscriptionStatusTrial   UserSubscriptionStatus = "trial"
	UserSubscriptionStatusExpired UserSubscriptionStatus = "expired"
)

type User struct {
	Id                 string                 `json:"id"`
	Email              string                 `json:"email"`
	SubscriptionStatus UserSubscriptionStatus `json:"subscription_status"`
}

type GetUserParams struct {
	Ctx
}

type MagnetFileType string

const (
	MagnetFileTypeFile   = "file"
	MagnetFileTypeFolder = "folder"
)

type MagnetFile struct {
	Idx  int    `json:"index"`
	Link string `json:"link,omitempty"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
	Size int    `json:"size"`
}

type MagnetStatus string

const (
	MagnetStatusCached      MagnetStatus = "cached" // cached in store, ready to download instantly
	MagnetStatusQueued      MagnetStatus = "queued"
	MagnetStatusDownloading MagnetStatus = "downloading"
	MagnetStatusProcessing  MagnetStatus = "processing" // compressing / moving
	MagnetStatusDownloaded  MagnetStatus = "downloaded"
	MagnetStatusUploading   MagnetStatus = "uploading"
	MagnetStatusFailed      MagnetStatus = "failed"
	MagnetStatusInvalid     MagnetStatus = "invalid"
	MagnetStatusUnknown     MagnetStatus = "unknown"
)

type CheckMagnetParams struct {
	Ctx
	Magnets []string
}

type CheckMagnetDataItem struct {
	Hash   string       `json:"hash"`
	Magnet string       `json:"magnet"`
	Status MagnetStatus `json:"status"`
	Files  []MagnetFile `json:"files"`
}

type CheckMagnetData struct {
	Items []CheckMagnetDataItem `json:"items"`
}

type AddMagnetData struct {
	Id     string       `json:"id"`
	Hash   string       `json:"hash"`
	Magnet string       `json:"magnet"`
	Name   string       `json:"name"`
	Status MagnetStatus `json:"status"`
	Files  []MagnetFile `json:"files"`
}

type AddMagnetParams struct {
	Ctx
	Magnet string
}

type GetMagnetData struct {
	Id     string       `json:"id"`
	Name   string       `json:"name"`
	Status MagnetStatus `json:"status"`
	Files  []MagnetFile `json:"files"`
}

type GetMagnetParams struct {
	Ctx
	Id string
}

type ListMagnetsDataItem struct {
	Id     string       `json:"id"`
	Name   string       `json:"name"`
	Status MagnetStatus `json:"status"`
}

type ListMagnetsData struct {
	Items []ListMagnetsDataItem `json:"items"`
}

type ListMagnetsParams struct {
	Ctx
}

type RemoveMagnetData struct {
	Id string `json:"id"`
}

type RemoveMagnetParams struct {
	Ctx
	Id string
}

type GenerateLinkData struct {
	Link string `json:"link"`
}

type GenerateLinkParams struct {
	Ctx
	Link string
}

type Store interface {
	GetName() StoreName
	GetUser(params *GetUserParams) (*User, error)
	CheckMagnet(params *CheckMagnetParams) (*CheckMagnetData, error)
	AddMagnet(params *AddMagnetParams) (*AddMagnetData, error)
	GetMagnet(params *GetMagnetParams) (*GetMagnetData, error)
	ListMagnets(params *ListMagnetsParams) (*ListMagnetsData, error)
	RemoveMagnet(params *RemoveMagnetParams) (*RemoveMagnetData, error)
	GenerateLink(params *GenerateLinkParams) (*GenerateLinkData, error)
}
