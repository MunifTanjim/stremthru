package torznab_client

import (
	"encoding/xml"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type CapsLimits struct {
	XMLName xml.Name `xml:"limits"`
	Max     int      `xml:"max,attr,omitempty"`
	Default int      `xml:"default,attr,omitempty"`
}

type CapsSearchingItemAvailable bool

func (b CapsSearchingItemAvailable) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	attr := xml.Attr{Name: name, Value: "no"}
	if b {
		attr.Value = "yes"
	}
	return attr, nil
}

func (b *CapsSearchingItemAvailable) UnmarshalXMLAttr(attr xml.Attr) error {
	*b = strings.ToLower(attr.Value) == "yes"
	return nil
}

type CapsSearchingItemSupportedParams []SearchParam

func (sp CapsSearchingItemSupportedParams) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.Join(sp, ",")}, nil
}

type CapsSearchingItem struct {
	Available       CapsSearchingItemAvailable       `xml:"available,attr"`
	SupportedParams CapsSearchingItemSupportedParams `xml:"supportedParams,attr"`
	SearchEngine    string                           `xml:"searchEngine,attr,omitempty"`

	supportedParam map[SearchParam]struct{} `xml:"-"`
}

func (csi CapsSearchingItem) SupportsParam(param SearchParam) bool {
	if csi.supportedParam == nil {
		csi.supportedParam = make(map[SearchParam]struct{}, len(csi.SupportedParams))
		for _, p := range csi.SupportedParams {
			csi.supportedParam[p] = struct{}{}
		}
	}
	_, ok := csi.supportedParam[param]
	return ok
}

type CapsCategory struct {
	XMLName xml.Name `xml:"category"`
	Category
	Subcat []Category `xml:"subcat"`
}

type Function string

const (
	FunctionCaps        Function = "caps"
	FunctionSearch      Function = "search"
	FunctionSearchTV    Function = "tvsearch"
	FunctionSearchMovie Function = "movie"
	FunctionSearchMusic Function = "music"
	FunctionSearchBook  Function = "book"
)

type Caps struct {
	XMLName xml.Name `xml:"caps"`
	Server  struct {
		Title string `xml:"title,attr"`
	} `xml:"server"`
	Limits    *CapsLimits
	Searching struct {
		Search      CapsSearchingItem `xml:"search"`
		TVSearch    CapsSearchingItem `xml:"tv-search"`
		MovieSearch CapsSearchingItem `xml:"movie-search"`
		MusicSearch CapsSearchingItem `xml:"music-search"`
		AudioSearch CapsSearchingItem `xml:"audio-search"`
		BookSearch  CapsSearchingItem `xml:"book-search"`
	} `xml:"searching"`
	Categories []CapsCategory `xml:"categories>category"`
}

func (caps Caps) getSearchingItem(t Function) *CapsSearchingItem {
	switch t {
	case FunctionSearch:
		return &caps.Searching.Search
	case FunctionSearchTV:
		return &caps.Searching.TVSearch
	case FunctionSearchMovie:
		return &caps.Searching.MovieSearch
	case FunctionSearchMusic:
		return &caps.Searching.MusicSearch
	case FunctionSearchBook:
		return &caps.Searching.BookSearch
	default:
		return nil
	}
}

func (caps Caps) SupportsFunction(t Function) bool {
	if si := caps.getSearchingItem(t); si != nil {
		return bool(si.Available)
	}
	return false
}
func (caps Caps) SupportsParam(t Function, param SearchParam) bool {
	if si := caps.getSearchingItem(t); si != nil {
		return si.SupportsParam(param)
	}
	return false
}

type GetCapsParams struct {
	Ctx
}

func (c *Client) getCaps(params *GetCapsParams) (request.APIResponse[Caps], error) {
	params.Query = &url.Values{
		"t": {string(FunctionCaps)},
	}

	var resp Response[Caps]
	res, err := c.Request("GET", "/api", params, &resp)
	return request.NewAPIResponse(res, resp.Data), err
}

func (c *Client) GetCaps() (Caps, error) {
	return c.caps.Get()
}
