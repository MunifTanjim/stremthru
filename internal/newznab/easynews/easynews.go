package newznab_easynews

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	nzb_info "github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

var (
	_ newznab_client.Indexer = (*Indexer)(nil)
)

type Indexer struct {
	user     string
	pass     string
	skipAuth bool
}

func NewIndexer(apiKey string) *Indexer {
	ba, _ := util.ParseBasicAuth(apiKey)
	return &Indexer{
		user: ba.Username,
		pass: ba.Password,
	}
}

var getCaps = sync.OnceValue(func() *znab.Caps {
	return &znab.Caps{
		Server: &znab.CapsServer{
			Title: "Easynews",
		},
		Searching: &znab.CapsSearching{
			Search: &znab.CapsSearchingItem{
				Available: true,
				SupportedParams: znab.CapsSearchingItemSupportedParams{
					znab.SearchParamQ,
				},
			},
			TVSearch: &znab.CapsSearchingItem{
				Available: true,
				SupportedParams: znab.CapsSearchingItemSupportedParams{
					znab.SearchParamQ,
					znab.SearchParamSeason,
					znab.SearchParamEp,
				},
			},
			MovieSearch: &znab.CapsSearchingItem{
				Available: true,
				SupportedParams: znab.CapsSearchingItemSupportedParams{
					znab.SearchParamQ,
				},
			},
		},
		Categories: []znab.CapsCategory{},
	}
})

func (idxr *Indexer) GetId() string {
	return "easynews"
}

func (idxr *Indexer) Capabilities() znab.Caps {
	return *getCaps()
}

func (i *Indexer) isValidAPIKey() bool {
	if i.skipAuth {
		return true
	}
	password := config.Auth.GetPassword(i.user)
	return password != "" && password == i.pass
}

func (idxr *Indexer) NewSearchQuery(fn func(caps *znab.Caps) newznab_client.Function) (*newznab_client.Query, error) {
	caps := getCaps()
	return newznab_client.NewQuery(caps).SetT(fn(caps)), nil
}

type searchDataItem struct {
	ACodec         string   `json:"acodec"`          // AAC
	Alangs         []string `json:"alangs"`          // ["ita"]
	AudioTracks    []string `json:"audio_tracks"`    // ["ita"]
	BPS            int      `json:"bps"`             // 1409501
	Colid          string   `json:"colid"`           //
	Disposition    string   `json:"disposition"`     // 1 / U
	Expires        string   `json:"expires"`         //
	Extension      string   `json:"extension"`       // ['11']~ .mp4
	FPS            float32  `json:"fps"`             // 23.98
	FallbackURL    string   `json:"fallbackURL"`     // //
	Fn             string   `json:"fn"`              // ['10']~
	Fullres        string   `json:"fullres"`         // 720 x 380
	Groups         string   `json:"groups"`          // alt.binaries.friends alt.binaries.boneless
	Hash           string   `json:"hash"`            // ['0']~
	Height         string   `json:"height"`          // 380
	Hz             int      `json:"hz"`              // 48000
	ID             string   `json:"id"`              //
	Master         string   `json:"master"`          // 32
	Meta           string   `json:"meta"`            //
	Mid            string   `json:"mid"`             //
	Nfo            string   `json:"nfo"`             //
	OldSetid       string   `json:"oldsetid"`        //
	OriginNSP      string   `json:"origin_nsp"`      // usenet.farm / tweaknews.nl
	Parset         string   `json:"parset"`          //
	Passwd         bool     `json:"passwd"`          // false
	Password       bool     `json:"password"`        // false
	Poster         string   `json:"poster"`          // FRIENDS <friends@group.local>
	PrimaryURL     string   `json:"primaryURL"`      // //
	RawSize        int64    `json:"rawSize"`         // 1367868353
	Reposts        int      `json:"reposts"`         // 2
	Runtime        int      `json:"runtime"`         // 7763
	SB             int      `json:"sb"`              // 1
	SC             bool     `json:"sc"`              // true
	Setid          string   `json:"setid"`           //
	Sig            string   `json:"sig"`             //
	Size           int64    `json:"size"`            // 1367868353
	Slangs         []string `json:"slangs"`          // ["ita"]
	Subject        string   `json:"subject"`         //
	SubtitleTracks []string `json:"subtitle_tracks"` // ["ita"]
	TS             int64    `json:"ts"`              // 1758741837
	Theight        int      `json:"theight"`         // 48
	Timestamp      int64    `json:"timestamp"`       // 1758741837
	Twidth         int      `json:"twidth"`          // 90
	Type           string   `json:"type"`            // VIDEO
	Uniq           string   `json:"uniq"`            //
	VCodec         string   `json:"vcodec"`          // H264
	Virus          bool     `json:"virus"`           // false
	Volume         bool     `json:"volume"`          // false
	Width          string   `json:"width"`           // 720
	XRes           int      `json:"xres"`            // 720
	YRes           int      `json:"yres"`            // 380

	title string `json:"-"`
}

func (item *searchDataItem) getDownloadURL(base, farm, port, sid string, num int) string {
	dlFarm := ""
	if farm != "" {
		dlFarm = "/" + farm + "/" + port
	}

	ext := strings.ReplaceAll(item.Expires, " ", "_")
	newHash := item.Hash + item.ID
	hashExt := url.PathEscape(newHash + ext)
	filenameExt := url.PathEscape(item.Fn + ext)

	dlURL := base + dlFarm + "/" + hashExt + "/" + filenameExt + "?"
	if sid != "" {
		dlURL += "sid=" + sid + ":" + util.IntToString(num) + "&"
	}
	if item.Sig != "" {
		dlURL += "sig=" + item.Sig
	}
	return dlURL
}

type searchData struct {
	BaseURL  string `json:"baseURL"`  // https://members.easynews.com
	DlFarm   string `json:"dlFarm"`   // auto
	DlPort   string `json:"dlPort"`   // 443
	DownURL  string `json:"downURL"`  // https://members.easynews.com/dl
	New      int    `json:"New"`      // 0
	NumPages int    `json:"numPages"` // 7
	Page     int    `json:"page"`     // 1
	PerPage  string `json:"perPage"`  // 100
	Results  int    `json:"results"`  //
	Returned int    `json:"returned"` // 27
	Sid      string `json:"sid"`      //
	Stemmed  string `json:"stemmed"`  // false
	ThumbURL string `json:"thumbURL"` // https://th.easynews.com/thumbnails-

	Data   []searchDataItem `json:"data"`
	Fields map[string]string
	Groups []map[string]int
}

var httpClient = func() *http.Client {
	client := config.GetHTTPClient(config.TUNNEL_TYPE_AUTO)
	client.Timeout = 30 * time.Second
	return client
}()

type ResultType int

const (
	ResultTypeVideo ResultType = iota
	ResultTypeSinglePartArchive
	ResultTypeMultiPartArchive
	ResultTypeNZB
)

var rarPartNumberRegex = regexp.MustCompile(`(?i)part(\d+)\.rar$`)
var sevenzipPartNumberRegex = regexp.MustCompile(`(?i)7z\.(\d+)$`)

func (idxr *Indexer) newSearchParams() url.Values {
	params := url.Values{}
	params.Set("nostem", "1")
	params.Set("safeO", "0")
	params.Set("sb", "1")
	params.Set("sc", "1")
	params.Set("st", "adv")
	params.Set("u", "1")

	params.Set("pby", "500")
	params.Set("pno", "1")

	params.Set("s1", "nrfile")
	params.Set("s1d", "-")
	params.Set("s2", "otime")
	params.Set("s2d", "+")
	params.Set("s3", "dsize")
	params.Set("s3d", "-")

	return params
}

func (idxr *Indexer) getAllArchiveVolumeFiles(filename string) ([]searchDataItem, error) {
	params := idxr.newSearchParams()

	params.Set("fty[]", "ARCHIVE")

	params.Set("s1", "nrfile")
	params.Set("s1d", "+")
	params.Set("s2", "relevance")
	params.Set("s2d", "+")
	params.Set("s3", "relevance")
	params.Set("s3d", "+")

	if rarPartNumberRegex.MatchString(filename) {
		println(filename)
		println(rarPartNumberRegex.ReplaceAllString(filename, ""))
		params.Set("fil", strings.TrimSpace(rarPartNumberRegex.ReplaceAllString(filename, "")))
		params.Set("fex", "rar")
	} else if sevenzipPartNumberRegex.MatchString(filename) {
		params.Set("fil", strings.TrimSpace(sevenzipPartNumberRegex.ReplaceAllString(filename, "7z")))
	} else {
		return nil, fmt.Errorf("filename does not match expected RAR or 7z part patterns")
	}

	result, err := idxr.search(params)

	return result.Data, err
}

func (idxr *Indexer) prepareSearchParams(searchQuery string, resultType ResultType) url.Values {
	params := idxr.newSearchParams()

	params.Set("gps", searchQuery)

	switch resultType {
	case ResultTypeSinglePartArchive:
		params.Set("fty[]", "ARCHIVE")
		params.Set("fil", "! part|nzb")
		params.Set("fex", "rar 7z")
	case ResultTypeMultiPartArchive:
		params.Set("fty[]", "ARCHIVE")
		params.Set("fil", "part01|part001|7z")
		params.Set("fex", "rar 001")
	case ResultTypeVideo:
		params.Set("fil", "! sample")
		params.Set("fty[]", "VIDEO")

		params.Set("s3", "druntime")
		params.Set("s3d", "-")
	case ResultTypeNZB:
		params.Set("fty[]", "ARCHIVE")
		params.Add("fty[]", "DOCUMENT")
		params.Set("fil", "! part")
		params.Set("fex", "nzb rar 7z")
		params.Set("sbj", "nzb")
	}

	return params
}

func (idxr *Indexer) search(params url.Values) (*searchData, error) {
	reqURL := "https://members.easynews.com/2.0/search/solr-search/?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(config.Integration.Easynews.User, config.Integration.Easynews.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("easynews search failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result searchData
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type nzbContainer interface {
	getNZBReqKey(idx int) string
	getNZBReqVal() string
}

func getNZBReqKey(idx int, sig string) string {
	return util.IntToString(idx) + "&sig=" + sig
}

func getNZBReqVal(hash, fname, ext string) string {
	b64Filename := util.Base64Encode(fname)
	b64Ext := util.Base64Encode(ext)
	return hash + "|" + strings.ReplaceAll(b64Filename, "=", "") + ":" + strings.ReplaceAll(b64Ext, "=", "")
}

func (item *searchDataItem) getNZBReqKey(idx int) string {
	return getNZBReqKey(idx, item.Sig)
}

func (item *searchDataItem) getNZBReqVal() string {
	return getNZBReqVal(item.Hash, item.Fn, item.Extension)
}

func (item *NZBIDData) getNZBReqKey(idx int) string {
	return getNZBReqKey(idx, item.Sig)
}

func (item *NZBIDData) getNZBReqVal() string {
	return getNZBReqVal(item.Hash, item.Fn, item.Extension)
}

func createNZB(name string, items []nzbContainer) (*nzb_info.NZBFile, error) {
	form := url.Values{}
	form.Set("autoNZB", "1")
	for i, item := range items {
		form.Add(item.getNZBReqKey(i), item.getNZBReqVal())
	}

	reqURL := "https://members.easynews.com/2.0/api/dl-nzb"
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.Integration.Easynews.User, config.Integration.Easynews.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("easynews create nzb failed: status %d", resp.StatusCode)
	}

	blob, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(name, ".nzb") {
		name += ".nzb"
	}

	return &nzb_info.NZBFile{
		Blob: blob,
		Name: name,
		Link: reqURL,
		Mod:  time.Now(),
	}, nil
}

func (idxr *Indexer) searchItems(searchQuery string, resultType ResultType) (*searchData, error) {
	params := idxr.prepareSearchParams(searchQuery, resultType)
	return idxr.search(params)
}

func buildSearchQuery(q string, season int, ep int) string {
	query := q
	if season > 0 {
		query += fmt.Sprintf(" S%02d", season)
		if ep > 0 {
			query += fmt.Sprintf("E%02d", ep)
		}
	}
	return query
}

func buildDownloadLink(item *searchDataItem) string {
	hash := item.Hash
	filename := item.Fn
	ext := item.Extension
	sig := item.Sig

	params := url.Values{}
	params.Set("hash", hash)
	params.Set("filename", filename)
	params.Set("ext", ext)
	if sig != "" {
		params.Set("sig", sig)
	}

	u := url.URL{
		Scheme:   "https",
		User:     url.UserPassword(config.Integration.Easynews.User, config.Integration.Easynews.Password),
		Host:     "members.easynews.com",
		Path:     "/2.0/api/dl-nzb",
		RawQuery: params.Encode(),
	}
	return u.String()
}

type NZBIDData struct {
	Type      ResultType
	Hash      string
	Fn        string
	Extension string
	Sig       string
}

func (d *NZBIDData) String() string {
	var buf bytes.Buffer
	buf.WriteString(d.Hash)
	buf.WriteString("::")
	buf.WriteString(d.Fn)
	buf.WriteString("::")
	buf.WriteString(d.Extension)
	buf.WriteString("::")
	buf.WriteString(d.Sig)
	return "easynews:" + strconv.Itoa(int(d.Type)) + ":" + util.Base64EncodeByte(buf.Bytes())
}

func ParseEasynewsNZBID(nzbID string) (*NZBIDData, error) {
	encoded, ok := strings.CutPrefix(nzbID, "easynews:")
	if !ok {
		return nil, fmt.Errorf("invalid nzb id prefix")
	}
	rTypeStr, encoded, ok := strings.Cut(encoded, ":")
	if !ok {
		return nil, fmt.Errorf("invalid nzb id type")
	}
	rType := ResultType(util.SafeParseInt(rTypeStr, -1))
	if rType == -1 {
		return nil, fmt.Errorf("invalid nzb id type")
	}
	decoded, err := util.Base64Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid nzb id encoding: %w", err)
	}
	parts := strings.Split(decoded, "::")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid nzb id format")
	}
	nzbIDData := &NZBIDData{
		Type:      rType,
		Hash:      parts[0],
		Fn:        parts[1],
		Extension: parts[2],
		Sig:       parts[3],
	}
	return nzbIDData, nil
}

func (item *searchDataItem) getTitle() string {
	if item.title == "" {
		title := item.Fn
		if title == "" {
			fname, _ := nzb.ParseSubject(item.Subject)
			title = fname
		}
		if item.Extension != "" && !strings.HasSuffix(title, item.Extension) {
			title += item.Extension
		}
		item.title = title
	}
	return item.title
}

func (idxr *Indexer) convertSearchItemToNewz(rType ResultType, item *searchDataItem) newznab_client.Newz {
	title := item.getTitle()
	date := time.Unix(item.Timestamp, 0)

	group := ""
	if groups := strings.Fields(item.Groups); len(groups) > 0 {
		group = groups[0]
	}

	nzbIDData := NZBIDData{
		Type:      rType,
		Hash:      item.Hash,
		Fn:        item.Fn,
		Extension: item.Extension,
		Sig:       item.Sig,
	}
	nzbID := nzbIDData.String()

	downloadLink := config.BaseURL.JoinPath("/v0/newznab/easynews/getnzb", nzbID)
	query := url.Values{}
	apikey := util.Base64Encode(idxr.user + ":" + idxr.pass)
	query.Set("apikey", apikey)
	downloadLink.RawQuery = query.Encode()

	return newznab_client.Newz{
		Title:       title,
		GUID:        item.Hash,
		PublishDate: date,
		Size:        item.Size,

		Poster:   item.Poster,
		Group:    group,
		Password: item.Password,
		Date:     date,

		Hash: item.Hash,
		Indexer: newznab_client.ChannelItemIndexer{
			Name: "Easynews",
			Host: "members.easynews.com",
		},

		DownloadLink: downloadLink.String(),
	}
}

func (idxr *Indexer) Search(query url.Values, header http.Header) ([]newznab_client.Newz, error) {
	if !idxr.isValidAPIKey() {
		return nil, fmt.Errorf("invalid credentials")
	}

	q := query.Get(znab.SearchParamQ)
	if q == "" {
		return nil, nil
	}

	season := util.SafeParseInt(query.Get(znab.SearchParamSeason), 0)
	ep := util.SafeParseInt(query.Get(znab.SearchParamEp), 0)

	searchQuery := buildSearchQuery(q, season, ep)

	type searchResult struct {
		rType ResultType
		items []searchDataItem
		err   error
	}

	resultTypes := []ResultType{ResultTypeVideo, ResultTypeSinglePartArchive, ResultTypeMultiPartArchive, ResultTypeNZB}

	ch := make(chan searchResult, len(resultTypes))
	wg := sync.WaitGroup{}
	for _, resultType := range resultTypes {
		wg.Go(func() {
			resp, err := idxr.searchItems(searchQuery, resultType)
			if err != nil {
				ch <- searchResult{err: err}
				return
			}
			ch <- searchResult{rType: resultType, items: resp.Data}
		})
	}
	wg.Wait()
	close(ch)

	seen := map[string]struct{}{}
	var result []newznab_client.Newz
	normalizer := util.NewStringNormalizer()
	for sr := range ch {
		if sr.err != nil {
			return nil, sr.err
		}
		for idx := range sr.items {
			item := &sr.items[idx]
			hash := item.Hash
			if _, exists := seen[hash]; exists {
				continue
			}
			seen[hash] = struct{}{}
			title := item.getTitle()
			if util.FuzzyTokenSetRatio(q, title, normalizer) < 90 {
				continue
			}
			result = append(result, idxr.convertSearchItemToNewz(sr.rType, item))
		}
	}
	return result, nil
}

func DownloadNZB(nzbID string) (*nzb_info.NZBFile, error) {
	idData, err := ParseEasynewsNZBID(nzbID)
	if err != nil {
		return nil, err
	}
	switch idData.Type {
	case ResultTypeVideo, ResultTypeSinglePartArchive:
		nzbFile, err := createNZB(idData.Fn, []nzbContainer{idData})
		if err != nil {
			return nil, err
		}
		return nzbFile, nil
	case ResultTypeMultiPartArchive:
		idxr := &Indexer{skipAuth: true}
		items, err := idxr.getAllArchiveVolumeFiles(idData.Fn + idData.Extension)
		if err != nil {
			return nil, err
		}
		files := []nzbContainer{}
		for i := range items {
			files = append(files, &items[i])
		}
		nzbFile, err := createNZB(idData.Fn, files)
		if err != nil {
			return nil, err
		}
		return nzbFile, nil
	}
	return nil, fmt.Errorf("unsupported nzb id type")
}
