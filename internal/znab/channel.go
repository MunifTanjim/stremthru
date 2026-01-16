package znab

import (
	"encoding/json"
	"encoding/xml"
	"time"
)

type ChannelItemEnclosure struct {
	XMLName xml.Name `xml:"enclosure" json:"-"`
	URL     string   `xml:"url,attr,omitempty"`
	Length  int64    `xml:"length,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"` // application/x-bittorrent, application/x-nzb
}

type jsonChannelItemEnclosure struct {
	Attributes struct {
		URL    string `json:"url,omitempty"`
		Length int64  `json:"length,omitempty"`
		Type   string `json:"type,omitempty"`
	} `json:"@attributes"`
}

func (e ChannelItemEnclosure) MarshalJSON() ([]byte, error) {
	je := jsonChannelItemEnclosure{}
	je.Attributes.URL = e.URL
	je.Attributes.Length = e.Length
	je.Attributes.Type = e.Type
	return json.Marshal(je)
}

type ChannelItemAttrName string

type ChannelItemAttr struct {
	XMLName xml.Name            `json:"-"`
	Name    ChannelItemAttrName `xml:"name,attr" json:"name"`
	Value   string              `xml:"value,attr" json:"value"`
}

type jsonChannelItemAttr struct {
	Attributes ChannelItemAttr `json:"@attributes"`
}

type ChannelItemAttrs []ChannelItemAttr

func (attrs ChannelItemAttrs) MarshalJSON() ([]byte, error) {
	jsonAttrs := make([]jsonChannelItemAttr, len(attrs))
	for i, attr := range attrs {
		jsonAttrs[i] = jsonChannelItemAttr{attr}
	}
	return json.Marshal(jsonAttrs)
}

func (attrs ChannelItemAttrs) Get(name ChannelItemAttrName) string {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value
		}
	}
	return ""
}

func (attrs ChannelItemAttrs) GetAll(name ChannelItemAttrName) []string {
	values := []string{}
	for _, attr := range attrs {
		if attr.Name == name {
			values = append(values, attr.Value)
		}
	}
	return values
}

type ChannelItem struct {
	XMLName xml.Name `xml:"item" json:"-"`

	Category    string               `xml:"category,omitempty" json:"category,omitempty"`
	Description string               `xml:"description,omitempty" json:"description,omitempty"`
	Enclosure   ChannelItemEnclosure `xml:"enclosure,omitempty" json:"-"`
	Files       int                  `xml:"files,omitempty" json:"files,omitempty"`
	GUID        string               `xml:"guid,omitempty" json:"guid,omitempty"`
	Link        string               `xml:"link,omitempty" json:"link,omitempty"`
	PublishDate string               `xml:"pubDate,omitempty" json:"pubDate,omitempty"` // Mon, 02 Jan 2006 15:04:05 -0700
	Title       string               `xml:"title,omitempty" json:"title,omitempty"`
	Attributes  ChannelItemAttrs     `xml:"-" json:"attr"`
}

func (ci ChannelItem) GetPublishDate() time.Time {
	if ci.PublishDate == "" {
		return time.Time{}
	}
	if t, err := time.Parse(TimeFormat, ci.PublishDate); err == nil {
		return t
	}
	return time.Time{}
}

type Channel[Item any] struct {
	XMLName     xml.Name `xml:"channel" json:"-"`
	Title       string   `xml:"title,omitempty" json:"title,omitempty"`
	Description string   `xml:"description,omitempty" json:"description,omitempty"`
	Link        string   `xml:"link,omitempty" json:"link,omitempty"`
	Language    string   `xml:"language,omitempty" json:"language,omitempty"`
	Category    string   `xml:"category,omitempty" json:"category,omitempty"`
	Items       []Item   `xml:"item" json:"items"`
}
