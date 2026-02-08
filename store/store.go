package store

import (
	"errors"
	"mime/multipart"
	"time"

	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
	"github.com/anacrolix/torrent/metainfo"
)

type StoreName string

const (
	StoreNameAlldebrid  StoreName = "alldebrid"
	StoreNameDebrider   StoreName = "debrider"
	StoreNameDebridLink StoreName = "debridlink"
	StoreNameEasyDebrid StoreName = "easydebrid"
	StoreNameOffcloud   StoreName = "offcloud"
	StoreNamePikPak     StoreName = "pikpak"
	StoreNamePremiumize StoreName = "premiumize"
	StoreNameRealDebrid StoreName = "realdebrid"
	StoreNameStremThru   StoreName = "stremthru"
	StoreNameTorBox      StoreName = "torbox"
	StoreNameQBittorrent StoreName = "qbittorrent"
)

type StoreCode string

const (
	StoreCodeAllDebrid  StoreCode = "ad"
	StoreCodeDebrider   StoreCode = "dr"
	StoreCodeDebridLink StoreCode = "dl"
	StoreCodeEasyDebrid StoreCode = "ed"
	StoreCodeOffcloud   StoreCode = "oc"
	StoreCodePikPak     StoreCode = "pp"
	StoreCodePremiumize StoreCode = "pm"
	StoreCodeRealDebrid StoreCode = "rd"
	StoreCodeStremThru   StoreCode = "st"
	StoreCodeTorBox      StoreCode = "tb"
	StoreCodeQBittorrent StoreCode = "qb"
)

var storeCodeByName = map[StoreName]StoreCode{
	StoreNameAlldebrid:  StoreCodeAllDebrid,
	StoreNameDebrider:   StoreCodeDebrider,
	StoreNameDebridLink: StoreCodeDebridLink,
	StoreNameEasyDebrid: StoreCodeEasyDebrid,
	StoreNameOffcloud:   StoreCodeOffcloud,
	StoreNamePikPak:     StoreCodePikPak,
	StoreNamePremiumize: StoreCodePremiumize,
	StoreNameRealDebrid: StoreCodeRealDebrid,
	StoreNameStremThru:   StoreCodeStremThru,
	StoreNameTorBox:      StoreCodeTorBox,
	StoreNameQBittorrent: StoreCodeQBittorrent,
}

var storeNameByCode = map[StoreCode]StoreName{
	StoreCodeAllDebrid:  StoreNameAlldebrid,
	StoreCodeDebrider:   StoreNameDebrider,
	StoreCodeDebridLink: StoreNameDebridLink,
	StoreCodeEasyDebrid: StoreNameEasyDebrid,
	StoreCodeOffcloud:   StoreNameOffcloud,
	StoreCodePikPak:     StoreNamePikPak,
	StoreCodePremiumize: StoreNamePremiumize,
	StoreCodeRealDebrid: StoreNameRealDebrid,
	StoreCodeStremThru:   StoreNameStremThru,
	StoreCodeTorBox:      StoreNameTorBox,
	StoreCodeQBittorrent: StoreNameQBittorrent,
}

func (sn StoreName) Code() StoreCode {
	return storeCodeByName[sn]
}

func (sn StoreName) IsValid() bool {
	_, ok := storeCodeByName[sn]
	return ok
}

var ErrInvalidName = errors.New("invalid store name")

func (sn StoreName) Validate() (StoreName, error) {
	if !sn.IsValid() {
		return sn, ErrInvalidName
	}
	return sn, nil
}

func (sc StoreCode) Name() StoreName {
	return storeNameByCode[sc]
}

func (sc StoreCode) IsValid() bool {
	_, ok := storeNameByCode[sc]
	return ok
}

type Ctx = request.Ctx

var (
	_ File = (*MagnetFile)(nil)
	_ File = (*NewzFile)(nil)
)

type File interface {
	GetIdx() int
	GetPath() string
	GetName() string
	GetSize() int64
	GetLink() string
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
	HasUsenet          bool                   `json:"has_usenet"`
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
	Idx       int                   `json:"index"`
	Link      string                `json:"link,omitempty"`
	Path      string                `json:"path"`
	Name      string                `json:"name"`
	Size      int64                 `json:"size"`
	VideoHash string                `json:"video_hash,omitempty"`
	MediaInfo *media_info.MediaInfo `json:"media_info,omitempty"`
	Source    string                `json:"source,omitempty"`
}

func (f *MagnetFile) GetIdx() int {
	return f.Idx
}

func (f *MagnetFile) GetPath() string {
	return f.Path
}

func (f *MagnetFile) GetName() string {
	return f.Name
}

func (f *MagnetFile) GetSize() int64 {
	return f.Size
}

func (f *MagnetFile) GetLink() string {
	return f.Link
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
	Name   string       `json:"-"`
	Size   int64        `json:"-"`
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
	Private bool         `json:"private,omitempty"`
	AddedAt time.Time    `json:"added_at"`
}

type AddMagnetParams struct {
	Ctx
	Magnet          string
	Torrent         *multipart.FileHeader
	ClientIP        string
	torrentMetaInfo *metainfo.MetaInfo
	torrentInfo     *metainfo.Info
}

func (p *AddMagnetParams) GetTorrentMeta() (*metainfo.MetaInfo, *metainfo.Info, error) {
	if p.Torrent == nil {
		return nil, nil, nil
	}
	if p.torrentMetaInfo != nil && p.torrentInfo != nil {
		return p.torrentMetaInfo, p.torrentInfo, nil
	}
	f, err := p.Torrent.Open()
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	mi, err := metainfo.Load(f)
	if err != nil {
		return nil, nil, err
	}
	p.torrentMetaInfo = mi
	mii, err := mi.UnmarshalInfo()
	if err != nil {
		return nil, nil, err
	}
	if !mii.HasV1() {
		return nil, nil, errors.New("unsupported torrent file")
	}
	p.torrentInfo = &mii
	return p.torrentMetaInfo, p.torrentInfo, nil
}

type GetMagnetData struct {
	Id      string       `json:"id"`
	Name    string       `json:"name"`
	Hash    string       `json:"hash"`
	Size    int64        `json:"size"`
	Status  MagnetStatus `json:"status"`
	Files   []MagnetFile `json:"files"`
	Private bool         `json:"private,omitempty"`
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
	Private bool         `json:"private,omitempty"`
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

type NewzStatus string

const (
	NewzStatusCached      NewzStatus = "cached"
	NewzStatusQueued      NewzStatus = "queued"
	NewzStatusDownloading NewzStatus = "downloading"
	NewzStatusProcessing  NewzStatus = "processing"
	NewzStatusDownloaded  NewzStatus = "downloaded"
	NewzStatusFailed      NewzStatus = "failed"
	NewzStatusInvalid     NewzStatus = "invalid"
	NewzStatusUnknown     NewzStatus = "unknown"
)

type CheckNewzParams struct {
	Ctx
	Hashes []string
}

type CheckNewzDataItem struct {
	Hash   string     `json:"hash"`
	Status NewzStatus `json:"status"`
	Files  []NewzFile `json:"files"`
}

type CheckNewzData struct {
	Items []CheckNewzDataItem `json:"items"`
}

type AddNewzParams struct {
	Ctx
	File     *multipart.FileHeader
	Link     string
	ClientIP string
}

type AddNewzData struct {
	Id     string     `json:"id"`
	Hash   string     `json:"hash"`
	Status NewzStatus `json:"status"`
}

type GetNewzParams struct {
	Ctx
	Id       string
	ClientIP string
}

type NewzFile struct {
	Idx       int    `json:"index"`
	Link      string `json:"link,omitempty"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	VideoHash string `json:"video_hash,omitempty"`
}

func (f *NewzFile) GetIdx() int {
	return f.Idx
}

func (f *NewzFile) GetPath() string {
	return f.Path
}

func (f *NewzFile) GetName() string {
	return f.Name
}

func (f *NewzFile) GetSize() int64 {
	return f.Size
}

func (f *NewzFile) GetLink() string {
	return f.Link
}

type GetNewzData struct {
	Id      string     `json:"id"`
	Hash    string     `json:"hash"`
	Name    string     `json:"name"`
	Size    int64      `json:"size"`
	Status  NewzStatus `json:"status"`
	Files   []NewzFile `json:"files"`
	AddedAt time.Time  `json:"added_at"`
}

type ListNewzParams struct {
	Ctx
	Limit    int // min 1, max 500, default 100
	Offset   int // default 0
	ClientIP string
}

type ListNewzDataItem struct {
	Id      string     `json:"id"`
	Hash    string     `json:"hash"`
	Name    string     `json:"name"`
	Size    int64      `json:"size"`
	Status  NewzStatus `json:"status"`
	AddedAt time.Time  `json:"added_at"`
}

type ListNewzData struct {
	Items      []ListNewzDataItem `json:"items"`
	TotalItems int                `json:"total_items"`
}

type RemoveNewzParams struct {
	Ctx
	Id string
}

type RemoveNewzData struct {
	Id string `json:"id"`
}

type GenerateNewzLinkParams struct {
	Ctx
	Link     string
	ClientIP string
}

type GenerateNewzLinkData struct {
	Link string `json:"link"`
}

type NewzStore interface {
	GetName() StoreName
	GetUser(params *GetUserParams) (*User, error)
	CheckNewz(params *CheckNewzParams) (*CheckNewzData, error)
	AddNewz(params *AddNewzParams) (*AddNewzData, error)
	GetNewz(params *GetNewzParams) (*GetNewzData, error)
	ListNewz(params *ListNewzParams) (*ListNewzData, error)
	RemoveNewz(params *RemoveNewzParams) (*RemoveNewzData, error)
	GenerateNewzLink(params *GenerateNewzLinkParams) (*GenerateNewzLinkData, error)
}
