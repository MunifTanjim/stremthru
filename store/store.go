package store

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
)

type RequestContext interface {
	GetAPIKey(fallbackAPIKey string) string
	GetContext() context.Context
	PrepareHeader(header *http.Header)
	PrepareBody(method string, query *url.Values) (body io.Reader, contentType string, err error)
	NewRequest(baseURL *url.URL, method, path string, header func(header *http.Header, params RequestContext), query func(query *url.Values, params RequestContext)) (req *http.Request, err error)
}

type Ctx struct {
	APIKey  string          `json:"-"`
	Context context.Context `json:"-"`
	Form    *url.Values     `json:"-"`
	JSON    any             `json:"-"`
	Headers *http.Header    `json:"-"`
}

func (ctx Ctx) GetAPIKey(fallbackAPIKey string) string {
	if len(ctx.APIKey) > 0 {
		return ctx.APIKey
	}
	return fallbackAPIKey
}

func (ctx Ctx) GetContext() context.Context {
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	return ctx.Context
}

func (ctx Ctx) PrepareBody(method string, query *url.Values) (body io.Reader, contentType string, err error) {
	if ctx.JSON != nil {
		jsonBytes, err := json.Marshal(ctx.JSON)
		if err != nil {
			return nil, "", err
		}
		body = bytes.NewBuffer(jsonBytes)
		contentType = "application/json"
	}
	if ctx.Form != nil {
		if method == http.MethodHead || method == http.MethodGet || ctx.JSON != nil {
			for key, values := range *ctx.Form {
				for _, value := range values {
					query.Add(key, value)
				}
			}
		} else {
			body = strings.NewReader(ctx.Form.Encode())
			contentType = "application/x-www-form-urlencoded"
		}
	}
	return body, contentType, nil
}

func (ctx Ctx) PrepareHeader(header *http.Header) {
	if ctx.Headers == nil {
		return
	}

	for key, values := range *ctx.Headers {
		for _, value := range values {
			header.Add(key, value)
		}
	}
}

func (ctx Ctx) NewRequest(baseURL *url.URL, method, path string, header func(header *http.Header, params RequestContext), query func(query *url.Values, params RequestContext)) (req *http.Request, err error) {
	url := baseURL.JoinPath(path)

	q := url.Query()
	query(&q, ctx)

	body, contentType, err := ctx.PrepareBody(method, &q)
	if err != nil {
		return nil, err
	}

	url.RawQuery = q.Encode()

	req, err = http.NewRequestWithContext(ctx.GetContext(), method, url.String(), body)
	if err != nil {
		return nil, err
	}

	header(&req.Header, ctx)
	ctx.PrepareHeader(&req.Header)

	if len(contentType) > 0 {
		req.Header.Add("Content-Type", contentType)
	}

	return req, nil
}

type StoreName string

const (
	StoreNameAlldebrid  StoreName = "alldebrid"
	StoreNameDebridLink StoreName = "debridlink"
	StoreNamePremiumize StoreName = "premiumize"
	StoreNameRealDebrid StoreName = "realdebrid"
	StoreNameTorBox     StoreName = "torbox"
)

type StoreCode string

const (
	StoreCodeAllDebrid  StoreCode = "ad"
	StoreCodeDebridLink StoreCode = "dl"
	StoreCodePremiumize StoreCode = "pm"
	StoreCodeRealDebrid StoreCode = "rd"
	StoreCodeTorBox     StoreCode = "tb"
)

var storeCodeByName = map[StoreName]StoreCode{
	StoreNameAlldebrid:  StoreCodeAllDebrid,
	StoreNameDebridLink: StoreCodeDebridLink,
	StoreNamePremiumize: StoreCodePremiumize,
	StoreNameRealDebrid: StoreCodeRealDebrid,
	StoreNameTorBox:     StoreCodeTorBox,
}

func (sn StoreName) Code() StoreCode {
	return storeCodeByName[sn]
}

func (sn StoreName) Validate() (StoreName, *core.StoreError) {
	if sn == StoreNameAlldebrid || sn == StoreNameDebridLink || sn == StoreNamePremiumize || sn == StoreNameRealDebrid || sn == StoreNameTorBox {
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
	Magnet   string
	ClientIP string
}

type GetMagnetData struct {
	Id     string       `json:"id"`
	Name   string       `json:"name"`
	Hash   string       `json:"hash"`
	Status MagnetStatus `json:"status"`
	Files  []MagnetFile `json:"files"`
}

type GetMagnetParams struct {
	Ctx
	Id string
}

type ListMagnetsDataItem struct {
	Id     string       `json:"id"`
	Hash   string       `json:"hash"`
	Name   string       `json:"name"`
	Status MagnetStatus `json:"status"`
}

type ListMagnetsData struct {
	Items      []ListMagnetsDataItem `json:"items"`
	TotalItems int                   `json:"total_items"`
}

type ListMagnetsParams struct {
	Ctx
	Limit  int // min 1, max 500, default 100
	Offset int // default 0
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
