package debridlink

import (
	"net/url"
	"strconv"
	"strings"
)

type SeedboxTorrentFile struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	DownloadUrl     string `json:"downloadUrl"`
	Downloaded      bool   `json:"downloaded"`
	Size            int    `json:"size"`
	DownloadPercent int    `json:"downloadPercent"`
}

type SeedboxTorrentTracker struct {
	Announce string `json:"announce"`
}

type SeedboxTorrent struct {
	Id              string                  `json:"id"`
	Name            string                  `json:"name"`
	Error           int                     `json:"error"`
	ErrorString     string                  `json:"errorString"`
	HashString      string                  `json:"hashString"`
	UploadRatio     float64                 `json:"uploadRatio"`
	ServerId        string                  `json:"serverId"`
	Wait            bool                    `json:"wait"`
	Downloaded      bool                    `json:"downloaded"`
	PeersConnected  int                     `json:"peersConnected"`
	Status          int                     `json:"status"`
	ActProgress     bool                    `json:"act_progress"`
	TotalSize       int                     `json:"totalSize"`
	Files           []SeedboxTorrentFile    `json:"files"`
	Trackers        []SeedboxTorrentTracker `json:"trackers"`
	IsZip           bool                    `json:"isZip"`
	Created         int                     `json:"created"`
	DownloadPercent int                     `json:"downloadPercent"`
	DownloadSpeed   int                     `json:"downloadSpeed"`
	UploadSpeed     int                     `json:"uploadSpeed"`
}

type SeedboxTorrentStructureType string

const (
	SeedboxTorrentStructureTypeList = "list"
	SeedboxTorrentStructureTypeTree = "tree"
)

const LIST_SEEDBOX_TORRENTS_PER_PAGE_MIN = 20
const LIST_SEEDBOX_TORRENTS_PER_PAGE_MAX = 50

type ListSeedboxTorrentsParams struct {
	Ctx
	Ids           []string
	StructureType SeedboxTorrentStructureType
	Page          int // start at 0
	PerPage       int // min 20, max 50
}

type ListSeedboxTorrentsData struct {
	Value      []SeedboxTorrent
	Pagination ResponsePagination
}

func (c APIClient) ListSeedboxTorrents(params *ListSeedboxTorrentsParams) (APIResponse[ListSeedboxTorrentsData], error) {
	form := &url.Values{}
	if len(params.Ids) > 0 {
		form.Add("ids", strings.Join(params.Ids, ","))
	}
	if params.Page != 0 {
		form.Add("page", strconv.Itoa(params.Page))
	}
	if params.PerPage != 0 {
		form.Add("perPage", strconv.Itoa(params.PerPage))
	}
	params.Form = form

	response := &PaginatedResponse[SeedboxTorrent]{}
	res, err := c.Request("GET", "/v2/seedbox/list", params, response)
	return newAPIResponse(res, ListSeedboxTorrentsData{
		Value:      response.Value,
		Pagination: response.Pagination,
	}), err
}

type AddSeedboxTorrentData = SeedboxTorrent

type AddSeedboxTorrentBody struct {
	Url           string                      `json:"rl"`    // torrent url, magnet or hash
	Wait          bool                        `json:"wait"`  // wait before starting the torrent to select files. default: false
	Async         bool                        `json:"async"` // If true, won't wait metadata before returning result, recommended. default: false
	StructureType SeedboxTorrentStructureType `json:"structureType,omitempty"`
}

type AddSeedboxTorrentParams struct {
	Ctx
	Url           string                      `json:"url"`   // torrent url, magnet or hash
	Wait          bool                        `json:"wait"`  // wait before starting the torrent to select files. default: false
	Async         bool                        `json:"async"` // If true, won't wait metadata before returning result, recommended. default: false
	StructureType SeedboxTorrentStructureType `json:"structureType,omitempty"`
}

func (c APIClient) AddSeedboxTorrent(params *AddSeedboxTorrentParams) (APIResponse[AddSeedboxTorrentData], error) {
	params.JSON = params
	response := &Response[AddSeedboxTorrentData]{}
	res, err := c.Request("POST", "/v2/seedbox/add", params, response)
	return newAPIResponse(res, response.Value), err
}

type RemoveSeedboxTorrentData = []string

type RemoveSeedboxTorrentParams struct {
	Ctx
	Ids []string
}

func (c APIClient) RemoveSeedboxTorrents(params *RemoveSeedboxTorrentParams) (APIResponse[RemoveSeedboxTorrentData], error) {
	response := &Response[RemoveSeedboxTorrentData]{}
	res, err := c.Request("DELETE", "/v2/seedbox/"+strings.Join(params.Ids, ",")+"/remove", params, response)
	return newAPIResponse(res, response.Value), err
}

type CheckSeedboxTorrentsCachedTorrentFile struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type CheckSeedboxTorrentsCachedTorrent struct {
	Name       string                                  `json:"name"`
	HashString string                                  `json:"hashString"`
	Files      []CheckSeedboxTorrentsCachedTorrentFile `json:"files"`
}

type CheckSeedboxTorrentsCachedData = map[string]CheckSeedboxTorrentsCachedTorrent

type CheckSeedboxTorrentsCachedParams struct {
	Ctx
	Urls []string // torrent url, magnet or hash
}

func (c APIClient) CheckSeedboxTorrentsCached(params *CheckSeedboxTorrentsCachedParams) (APIResponse[CheckSeedboxTorrentsCachedData], error) {
	form := &url.Values{}
	form.Add("url", strings.Join(params.Urls, ","))
	params.Form = form

	response := &Response[CheckSeedboxTorrentsCachedData]{}
	res, err := c.Request("GET", "/v2/seedbox/cached", params, response)
	return newAPIResponse(res, response.Value), err
}
