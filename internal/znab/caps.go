package znab

import (
	"encoding/json"
	"encoding/xml"
	"strings"
)

type CapsServer struct {
	XMLName   xml.Name `xml:"server" json:"-"`
	Version   string   `xml:"version,attr,omitempty" json:"version,omitempty"`
	Title     string   `xml:"title,attr,omitempty" json:"title,omitempty"`
	Strapline string   `xml:"strapline,attr,omitempty" json:"strapline,omitempty"`
	Email     string   `xml:"email,attr,omitempty" json:"email,omitempty"`
	URL       string   `xml:"url,attr,omitempty" json:"url,omitempty"`
	Image     string   `xml:"image,attr,omitempty" json:"image,omitempty"`
}

type jsonCapsServer struct {
	Attributes *CapsServer `json:"@attributes"`
}

func (jcs jsonCapsServer) IsZero() bool {
	return jcs.Attributes == nil
}

type CapsLimits struct {
	XMLName xml.Name `xml:"limits" json:"-"`
	Max     int      `xml:"max,attr,omitempty" json:"max,omitempty"`
	Default int      `xml:"default,attr,omitempty" json:"default,omitempty"`
}

type jsonCapsLimits struct {
	Attributes *CapsLimits `json:"@attributes"`
}

func (jcl jsonCapsLimits) IsZero() bool {
	return jcl.Attributes == nil
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

func (sp CapsSearchingItemSupportedParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Join(sp, ","))
}

func (sp *CapsSearchingItemSupportedParams) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*sp = nil
		return nil
	}
	*sp = strings.Split(s, ",")
	return nil
}

func (sp CapsSearchingItemSupportedParams) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.Join(sp, ",")}, nil
}

func (sp *CapsSearchingItemSupportedParams) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "" {
		*sp = nil
		return nil
	}
	*sp = strings.Split(attr.Value, ",")
	return nil
}

type CapsSearchingItem struct {
	Available       CapsSearchingItemAvailable       `xml:"available,attr" json:"available"`
	SupportedParams CapsSearchingItemSupportedParams `xml:"supportedParams,attr" json:"supportedParams,omitempty"`
	SearchEngine    string                           `xml:"searchEngine,attr,omitempty" json:"searchEngine,omitempty"` // raw

	supportedParam map[SearchParam]struct{} `xml:"-" json:"-"`
}

func (csi *CapsSearchingItem) IsZero() bool {
	return !bool(csi.Available) && len(csi.SupportedParams) == 0 && csi.SearchEngine == ""
}

func (csi *CapsSearchingItem) IsEmpty() bool {
	return csi.IsZero()
}

func (csi *CapsSearchingItem) SupportsParam(param SearchParam) bool {
	if len(csi.supportedParam) == 0 {
		csi.supportedParam = make(map[SearchParam]struct{}, len(csi.SupportedParams))
		for _, p := range csi.SupportedParams {
			csi.supportedParam[p] = struct{}{}
		}
	}
	_, ok := csi.supportedParam[param]
	return ok
}

type jsonCapsSearchingItem struct {
	Attributes *CapsSearchingItem `json:"@attributes"`
}

type CapsSearching struct {
	XMLName     xml.Name           `xml:"searching"`
	Search      *CapsSearchingItem `xml:"search,omitempty"`
	TVSearch    *CapsSearchingItem `xml:"tv-search,omitempty"`
	MovieSearch *CapsSearchingItem `xml:"movie-search,omitempty"`
	MusicSearch *CapsSearchingItem `xml:"music-search,omitempty"`
	AudioSearch *CapsSearchingItem `xml:"audio-search,omitempty"`
	BookSearch  *CapsSearchingItem `xml:"book-search,omitempty"`
}

type jsonCapsSearching struct {
	Search      jsonCapsSearchingItem `json:"search,omitzero"`
	TVSearch    jsonCapsSearchingItem `json:"tv-search,omitzero"`
	MovieSearch jsonCapsSearchingItem `json:"movie-search,omitzero"`
	MusicSearch jsonCapsSearchingItem `json:"music-search,omitzero"`
	AudioSearch jsonCapsSearchingItem `json:"audio-search,omitzero"`
	BookSearch  jsonCapsSearchingItem `json:"book-search,omitzero"`
}

func (cs CapsSearching) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonCapsSearching{
		Search:      jsonCapsSearchingItem{cs.Search},
		TVSearch:    jsonCapsSearchingItem{cs.TVSearch},
		MovieSearch: jsonCapsSearchingItem{cs.MovieSearch},
		MusicSearch: jsonCapsSearchingItem{cs.MusicSearch},
		AudioSearch: jsonCapsSearchingItem{cs.AudioSearch},
		BookSearch:  jsonCapsSearchingItem{cs.BookSearch},
	})
}

type CapsCategory struct {
	XMLName xml.Name `xml:"category"`
	Category
	Subcat []Category `xml:"subcat"`
}

type jsonCapsCategory struct {
	jsonCategory
	Subcat []jsonCategory `json:"subcat,omitempty"`
}

type jsonCapsCategories struct {
	Category []jsonCapsCategory `json:"category"`
}

func (jcc *jsonCapsCategories) IsZero() bool {
	return len(jcc.Category) == 0
}

type xmlCapsCategories struct {
	XMLName  xml.Name `xml:"categories"`
	Children []CapsCategory
}

type Caps struct {
	Server     *CapsServer
	Limits     *CapsLimits
	Searching  *CapsSearching
	Categories []CapsCategory
}

type jsonCaps struct {
	Server     jsonCapsServer     `json:"server,omitzero"`
	Limits     jsonCapsLimits     `json:"limits,omitzero"`
	Searching  *CapsSearching     `json:"searching,omitempty"`
	Categories jsonCapsCategories `json:"categories,omitzero"`
}

type xmlCaps struct {
	XMLName    xml.Name `xml:"caps"`
	Server     *CapsServer
	Limits     *CapsLimits
	Searching  *CapsSearching
	Categories xmlCapsCategories
}

func (c Caps) MarshalJSON() ([]byte, error) {
	jc := jsonCaps{
		Server:    jsonCapsServer{c.Server},
		Limits:    jsonCapsLimits{c.Limits},
		Searching: c.Searching,
		Categories: jsonCapsCategories{
			Category: make([]jsonCapsCategory, len(c.Categories)),
		},
	}
	for i, cat := range c.Categories {
		category := jsonCapsCategory{
			jsonCategory: jsonCategory{cat.Category},
		}
		if len(cat.Subcat) > 0 {
			category.Subcat = make([]jsonCategory, len(cat.Subcat))
			for i, subcat := range cat.Subcat {
				subcat.Name = strings.TrimPrefix(subcat.Name, cat.Name+"/")
				category.Subcat[i] = jsonCategory{subcat}
			}
		}
		jc.Categories.Category[i] = category
	}
	return json.Marshal(jc)
}

func (c Caps) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	cx := xmlCaps{
		Server: c.Server,
		Limits: c.Limits,
		Categories: xmlCapsCategories{
			Children: c.Categories,
		},
		Searching: c.Searching,
	}

	for i := range cx.Categories.Children {
		cat := &cx.Categories.Children[i]
		for i := range cat.Subcat {
			subcat := &cat.Subcat[i]
			subcat.Name = strings.TrimPrefix(subcat.Name, cat.Name+"/")
		}
	}

	return e.Encode(cx)
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

func (caps Caps) getSearchingItem(t Function) *CapsSearchingItem {
	switch t {
	case FunctionSearch:
		return caps.Searching.Search
	case FunctionSearchTV:
		return caps.Searching.TVSearch
	case FunctionSearchMovie:
		return caps.Searching.MovieSearch
	case FunctionSearchMusic:
		return caps.Searching.MusicSearch
	case FunctionSearchBook:
		return caps.Searching.BookSearch
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
