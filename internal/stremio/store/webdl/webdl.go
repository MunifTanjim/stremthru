package stremio_store_webdl

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/torbox"
)

var tbClient = torbox.NewAPIClient(&torbox.APIClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("torbox")),
})

type WebDLFile struct {
	Idx  int    `json:"index"`
	Link string `json:"link,omitempty"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
	Size int64  `json:"size"`
}

type ListWebDLsParams struct {
	request.Ctx
	Limit    int // min 1, max 500, default 100
	Offset   int // default 0
	ClientIP string
}

type WebDLStatus = store.MagnetStatus

type WebDL struct {
	Id      string      `json:"id"`
	Hash    string      `json:"hash"`
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Status  WebDLStatus `json:"status"`
	AddedAt time.Time   `json:"added_at"`
	Files   []WebDLFile `json:"files"`
}

type ListWebDLsData struct {
	Items      []WebDL `json:"items"`
	TotalItems int     `json:"total_items"`
}

func ListWebDLs(params *ListWebDLsParams, storeName store.StoreName) (*ListWebDLsData, error) {
	params.Limit = max(1, min(params.Limit, 500))

	switch storeName {
	case store.StoreNameTorBox:
		rParams := &torbox.ListWebDLDownloadParams{
			Ctx:    params.Ctx,
			Limit:  params.Limit,
			Offset: params.Offset,
		}
		res, err := tbClient.ListWebDLDownload(rParams)
		if err != nil {
			return nil, err
		}

		data := ListWebDLsData{}
		for i := range res.Data {
			und := &res.Data[i]
			item := WebDL{
				Id:      strconv.Itoa(und.Id),
				Hash:    und.Hash,
				Name:    und.Name,
				Size:    und.Size,
				Status:  store.MagnetStatusUnknown,
				AddedAt: und.GetAddedAt(),
			}
			if und.DownloadState == torbox.TorrentDownloadStateDownloading {
				item.Status = store.MagnetStatusDownloading
			} else if und.DownloadFinished && und.DownloadPresent {
				item.Status = store.MagnetStatusDownloaded
			}
			for i := range und.Files {
				f := &und.Files[i]
				file := WebDLFile{
					Idx:  f.Id,
					Link: torbox.LockedFileLink("").Create(und.Id, f.Id),
					Name: f.ShortName,
					Path: "/" + f.Name,
					Size: f.Size,
				}
				item.Files = append(item.Files, file)
			}
			data.Items = append(data.Items, item)
		}

		count := len(data.Items)
		// torbox returns 1 extra item
		if count > params.Limit {
			data.Items = data.Items[0:params.Limit]
			count = params.Limit
		}
		data.TotalItems = params.Offset + count
		if count == params.Limit {
			data.TotalItems += 1
		}

		return &data, nil
	default:
		return &ListWebDLsData{}, nil
	}
}

type GetWebDLParams struct {
	request.Ctx
	Id          string
	ClientIP    string
	BypassCache bool
}

type GetWebDLData = WebDL

func GetWebDL(params *GetWebDLParams, storeName store.StoreName) (*WebDL, error) {
	switch storeName {
	case store.StoreNameTorBox:
		id, err := strconv.Atoi(params.Id)
		if err != nil {
			return nil, err
		}
		rParams := &torbox.GetWebDLDownloadParams{
			Ctx:         params.Ctx,
			Id:          id,
			BypassCache: params.BypassCache,
		}
		res, err := tbClient.GetWebDLDownload(rParams)
		if err != nil {
			return nil, err
		}
		und := &res.Data
		item := WebDL{
			Id:      strconv.Itoa(und.Id),
			Hash:    und.Hash,
			Name:    und.Name,
			Size:    und.Size,
			Status:  store.MagnetStatusUnknown,
			AddedAt: und.GetAddedAt(),
		}
		if und.DownloadState == torbox.TorrentDownloadStateDownloading {
			item.Status = store.MagnetStatusDownloading
		}
		if und.DownloadFinished && und.DownloadPresent {
			item.Status = store.MagnetStatusDownloaded
		}
		for i := range und.Files {
			f := &und.Files[i]
			file := WebDLFile{
				Idx:  f.Id,
				Link: torbox.LockedFileLink("").Create(und.Id, f.Id),
				Name: f.ShortName,
				Path: "/" + f.Name,
				Size: f.Size,
			}
			item.Files = append(item.Files, file)
		}
		return &item, nil
	default:
		return nil, errors.New("unsupported")
	}
}

type GenerateLinkData struct {
	Link string `json:"link"`
}

type GenerateLinkParams struct {
	request.Ctx
	Link     string
	CLientIP string
}

func GenerateLink(params *GenerateLinkParams, storeName store.StoreName) (*GenerateLinkData, error) {
	switch storeName {
	case store.StoreNameTorBox:
		id, fileId, err := torbox.LockedFileLink(params.Link).Parse()
		if err != nil {
			error := core.NewAPIError("invalid link")
			error.StatusCode = http.StatusBadRequest
			error.Cause = err
			return nil, error
		}
		rParams := &torbox.RequestWebDLDownloadLinkParams{
			Ctx:     params.Ctx,
			WebDLId: id,
			FileId:  fileId,
			UserIP:  params.CLientIP,
		}
		res, err := tbClient.RequestWebDLDownloadLink(rParams)
		if err != nil {
			return nil, err
		}
		data := GenerateLinkData{
			Link: res.Data.Link,
		}
		return &data, nil
	default:
		return nil, errors.New("unsupported")
	}
}
