package znab

import (
	"encoding/xml"
)

// rfc822
const TimeFormat = "Mon, 02 Jan 2006 15:04:05 -0700"

type RSSFeed[ChannelItem any] struct {
	XMLName          xml.Name             `xml:"rss" json:"-"`
	AtomNamespace    string               `xml:"xmlns:atom,attr" json:"-"`
	TorznabNamespace string               `xml:"xmlns:torznab,attr" json:"-"`
	Version          string               `xml:"version,attr,omitempty" json:"-"`
	Channel          Channel[ChannelItem] `xml:"channel" json:"channel"`
}

type JSONFeed[ChannelItem any] struct {
	Attributes struct {
		Version string `json:"version,omitempty"`
	} `json:"@attributes"`
	RSSFeed[ChannelItem]
}

type Info struct {
	ID          string
	Title       string
	Description string
	Link        string
	Language    string
	Category    string
}
