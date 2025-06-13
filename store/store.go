package store

import (
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/request"
)

type Ctx = request.Ctx

type StoreName string

const (
	StoreNameAlldebrid  StoreName = "alldebrid"
	StoreNameDebridLink StoreName = "debridlink"
	StoreNameEasyDebrid StoreName = "easydebrid"
	StoreNameOffcloud   StoreName = "offcloud"
	StoreNamePikPak     StoreName = "pikpak"
	StoreNamePremiumize StoreName = "premiumize"
	StoreNameRealDebrid StoreName = "realdebrid"
	StoreNameTorBox     StoreName = "torbox"
)

type StoreCode string

const (
	StoreCodeAllDebrid  StoreCode = "ad"
	StoreCodeDebridLink StoreCode = "dl"
	StoreCodeEasyDebrid StoreCode = "ed"
	StoreCodeOffcloud   StoreCode = "oc"
	StoreCodePikPak     StoreCode = "pp"
	StoreCodePremiumize StoreCode = "pm"
	StoreCodeRealDebrid StoreCode = "rd"
	StoreCodeTorBox     StoreCode = "tb"
)

var storeCodeByName = map[StoreName]StoreCode{
	StoreNameAlldebrid:  StoreCodeAllDebrid,
	StoreNameDebridLink: StoreCodeDebridLink,
	StoreNameEasyDebrid: StoreCodeEasyDebrid,
	StoreNameOffcloud:   StoreCodeOffcloud,
	StoreNamePikPak:     StoreCodePikPak,
	StoreNamePremiumize: StoreCodePremiumize,
	StoreNameRealDebrid: StoreCodeRealDebrid,
	StoreNameTorBox:     StoreCodeTorBox,
}

var storeNameByCode = map[StoreCode]StoreName{
	StoreCodeAllDebrid:  StoreNameAlldebrid,
	StoreCodeDebridLink: StoreNameDebridLink,
	StoreCodeEasyDebrid: StoreNameEasyDebrid,
	StoreCodeOffcloud:   StoreNameOffcloud,
	StoreCodePikPak:     StoreNamePikPak,
	StoreCodePremiumize: StoreNamePremiumize,
	StoreCodeRealDebrid: StoreNameRealDebrid,
	StoreCodeTorBox:     StoreNameTorBox,
}

func (sn StoreName) Code() StoreCode {
	return storeCodeByName[sn]
}

func (sn StoreName) Validate() (StoreName, *core.StoreError) {
	if _, ok := storeCodeByName[sn]; !ok {
		return sn, ErrorInvalidStoreName(string(sn))
	}
	return sn, nil
}

func (sc StoreCode) Name() StoreName {
	return storeNameByCode[sc]
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
	Size int64  `json:"size"`
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
	Magnets          []string
	ClientIP         string
	SId              string
	LocalOnly        bool
	IsTrustedRequest bool
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
	Id      string       `json:"id"`
	Hash    string       `json:"hash"`
	Magnet  string       `json:"magnet"`
	Name    string       `json:"name"`
	Size    int64        `json:"size"`
	Status  MagnetStatus `json:"status"`
	Files   []MagnetFile `json:"files"`
	AddedAt time.Time    `json:"added_at"`
}

type AddMagnetParams struct {
	Ctx
	Magnet   string
	ClientIP string
}

type GetMagnetData struct {
	Id      string       `json:"id"`
	Name    string       `json:"name"`
	Hash    string       `json:"hash"`
	Size    int64        `json:"size"`
	Status  MagnetStatus `json:"status"`
	Files   []MagnetFile `json:"files"`
	AddedAt time.Time    `json:"added_at"`
}

type GetMagnetParams struct {
	Ctx
	Id       string
	ClientIP string
}

type ListMagnetsDataItem struct {
	Id      string       `json:"id"`
	Hash    string       `json:"hash"`
	Name    string       `json:"name"`
	Size    int64        `json:"size"`
	Status  MagnetStatus `json:"status"`
	AddedAt time.Time    `json:"added_at"`
}

type ListMagnetsData struct {
	Items      []ListMagnetsDataItem `json:"items"`
	TotalItems int                   `json:"total_items"`
}

type ListMagnetsParams struct {
	Ctx
	Limit    int // min 1, max 500, default 100
	Offset   int // default 0
	ClientIP string
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
	Link     string
	ClientIP string
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
