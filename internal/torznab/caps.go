package torznab

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

type CapsSearchingItemSupportedParams []string

func (sp CapsSearchingItemSupportedParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.Join(sp, ","))
}

func (sp CapsSearchingItemSupportedParams) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: strings.Join(sp, ",")}, nil
}

type CapsSearchingItem struct {
	Name            string                           `json:"-"`
	Available       CapsSearchingItemAvailable       `json:"available"`
	SupportedParams CapsSearchingItemSupportedParams `json:"supportedParams,omitempty"`
}

type jsonCapsSearchingItem struct {
	Attributes CapsSearchingItem `json:"@attributes"`
}

type xmlCapsSearchingItem struct {
	XMLName         xml.Name
	Available       CapsSearchingItemAvailable       `xml:"available,attr"`
	SupportedParams CapsSearchingItemSupportedParams `xml:"supportedParams,attr"`
}

type xmlCapsSearching struct {
	XMLName  xml.Name `xml:"searching"`
	Children []xmlCapsSearchingItem
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

type xmlCaps struct {
	XMLName    xml.Name `xml:"caps"`
	Server     *CapsServer
	Limits     *CapsLimits
	Searching  xmlCapsSearching
	Categories xmlCapsCategories
}

type jsonCaps struct {
	Server     jsonCapsServer                   `json:"server,omitzero"`
	Limits     jsonCapsLimits                   `json:"limits,omitzero"`
	Searching  map[string]jsonCapsSearchingItem `json:"searching,omitempty"`
	Categories jsonCapsCategories               `json:"categories,omitzero"`
}

type Caps struct {
	Server     *CapsServer
	Limits     *CapsLimits
	Searching  []CapsSearchingItem
	Categories []CapsCategory
}

func (c Caps) MarshalJSON() ([]byte, error) {
	jc := jsonCaps{
		Server: jsonCapsServer{c.Server},
		Limits: jsonCapsLimits{c.Limits},
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
	if len(c.Searching) > 0 {
		jc.Searching = make(map[string]jsonCapsSearchingItem, len(c.Searching))
		for _, mode := range c.Searching {
			jc.Searching[mode.Name] = jsonCapsSearchingItem{mode}
		}
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
	}

	for i := range cx.Categories.Children {
		cat := &cx.Categories.Children[i]
		for i := range cat.Subcat {
			subcat := &cat.Subcat[i]
			subcat.Name = strings.TrimPrefix(subcat.Name, cat.Name+"/")
		}
	}

	for _, mode := range c.Searching {
		cx.Searching.Children = append(cx.Searching.Children, xmlCapsSearchingItem{
			xml.Name{Space: "", Local: mode.Name},
			mode.Available,
			mode.SupportedParams,
		})
	}

	return e.Encode(cx)
}
