package torznab

import (
	"encoding/json"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

const rfc822 = "Mon, 02 Jan 2006 15:04:05 -0700"

type ChannelItemEnclosure struct {
	XMLName xml.Name `xml:"enclosure" json:"-"`
	URL     string   `xml:"url,attr,omitempty" json:"url"`
	Length  int64    `xml:"length,attr,omitempty" json:"length"`
	Type    string   `xml:"type,attr,omitempty" json:"type"`
}

type jsonChannelItemEnclosure struct {
	Attributes ChannelItemEnclosure `json:"@attributes"`
}

type ChannelItemAttribute struct {
	XMLName xml.Name `xml:"torznab:attr" json:"-"`
	Name    string   `xml:"name,attr" json:"name"`
	Value   string   `xml:"value,attr" json:"value"`
}

type channelItemAttributes []ChannelItemAttribute

type jsonChannelItemAttribute struct {
	Attributes ChannelItemAttribute `json:"@attributes"`
}

func (attrs channelItemAttributes) MarshalJSON() ([]byte, error) {
	jsonAttrs := make([]jsonChannelItemAttribute, len(attrs))
	for i, attr := range attrs {
		jsonAttrs[i] = jsonChannelItemAttribute{attr}
	}
	return json.Marshal(jsonAttrs)
}

type ChannelItem struct {
	XMLName xml.Name `xml:"item" json:"-"`

	// standard rss elements
	Category    string               `xml:"category,omitempty" json:"category,omitempty"`
	Description string               `xml:"description,omitempty" json:"description,omitempty"`
	Enclosure   ChannelItemEnclosure `xml:"enclosure,omitempty" json:"-"`
	Files       int                  `xml:"files,omitempty" json:"files,omitempty"`
	GUID        string               `xml:"guid,omitempty" json:"guid,omitempty"`
	Link        string               `xml:"link,omitempty" json:"link,omitempty"`
	PublishDate string               `xml:"pubDate,omitempty" json:"pubDate,omitempty"`
	Title       string               `xml:"title,omitempty" json:"title,omitempty"`

	Attributes []ChannelItemAttribute `json:"-"`
}

type jsonChannelItem struct {
	ChannelItem
	Enclosure  jsonChannelItemEnclosure `json:"enclosure"`
	Attributes channelItemAttributes    `json:"attr"`
}

type Channel struct {
	XMLName     xml.Name     `xml:"channel" json:"-"`
	Title       string       `xml:"title,omitempty" json:"title,omitempty"`
	Description string       `xml:"description,omitempty" json:"description,omitempty"`
	Link        string       `xml:"link,omitempty" json:"link,omitempty"`
	Language    string       `xml:"language,omitempty" json:"language,omitempty"`
	Category    string       `xml:"category,omitempty" json:"category,omitempty"`
	Items       []ResultItem `json:"items"`
}

type RSS struct {
	XMLName          xml.Name `xml:"rss" json:"-"`
	AtomNamespace    string   `xml:"xmlns:atom,attr" json:"-"`
	TorznabNamespace string   `xml:"xmlns:torznab,attr" json:"-"`
	Version          string   `xml:"version,attr,omitempty" json:"-"`
	Channel          Channel  `xml:"channel" json:"channel"`
}

type jsonRSS struct {
	Attributes struct {
		Version string `json:"version,omitempty"`
	} `json:"@attributes"`
	RSS
}

type ResultItem struct {
	Category    Category
	Description string
	Files       int
	GUID        string
	Link        string
	PublishDate time.Time
	Title       string

	Audio      string
	Codec      string
	IMDB       string
	InfoHash   string
	Language   string
	Leechers   int
	Resolution string
	Seeders    int
	Site       string
	Size       int64
	Year       int
}

func (ri ResultItem) toChannelItem() ChannelItem {
	attrs := channelItemAttributes{}
	if ri.Audio != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "audio", Value: ri.Audio})
	}
	attrs = append(attrs, ChannelItemAttribute{Name: "category", Value: strconv.Itoa(ri.Category.ID)})
	if ri.IMDB != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "imdb", Value: strings.TrimPrefix(ri.IMDB, "tt")})
	}
	if ri.InfoHash != "" {
		if magnet, err := core.ParseMagnetLink(ri.InfoHash); err == nil {
			attrs = append(
				attrs,
				ChannelItemAttribute{Name: "infohash", Value: magnet.Hash},
				ChannelItemAttribute{Name: "magneturl", Value: magnet.Link},
			)
			if ri.GUID == "" {
				ri.GUID = magnet.Hash
			}
			if ri.Link == "" {
				ri.Link = magnet.Link
			}
		}
	}
	if ri.Language != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "language", Value: ri.Language})
	}
	if ri.Leechers > 0 {
		attrs = append(attrs, ChannelItemAttribute{Name: "leechers", Value: strconv.Itoa(ri.Leechers)})
	}
	if ri.Resolution != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "resolution", Value: ri.Resolution})
	}
	if ri.Seeders > 0 {
		attrs = append(attrs, ChannelItemAttribute{Name: "seeders", Value: strconv.Itoa(ri.Seeders)})
	}
	if ri.Site != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "site", Value: ri.Site})
	}
	if ri.Size > 0 {
		attrs = append(attrs, ChannelItemAttribute{Name: "size", Value: strconv.FormatInt(ri.Size, 10)})
	}
	if ri.Codec != "" {
		attrs = append(attrs, ChannelItemAttribute{Name: "video", Value: ri.Codec})
	}
	if ri.Year != 0 {
		attrs = append(attrs, ChannelItemAttribute{Name: "year", Value: strconv.Itoa(ri.Year)})
	}

	return ChannelItem{
		Attributes:  attrs,
		Category:    ri.Category.Name,
		Description: ri.Description,
		Files:       ri.Files,
		GUID:        ri.GUID,
		Link:        ri.Link,
		PublishDate: ri.PublishDate.Format(rfc822),
		Title:       ri.Title,
		Enclosure: ChannelItemEnclosure{
			URL:    ri.Link,
			Length: ri.Size,
			Type:   "application/x-bittorrent;x-scheme-handler/magnet",
		},
	}
}

func (ri ResultItem) MarshalJSON() ([]byte, error) {
	item := jsonChannelItem{ChannelItem: ri.toChannelItem()}
	item.Attributes = channelItemAttributes(item.ChannelItem.Attributes)
	item.Enclosure = jsonChannelItemEnclosure{item.ChannelItem.Enclosure}
	return json.Marshal(item)
}

func (ri ResultItem) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.Encode(ri.toChannelItem())
}

type ResultFeed struct {
	Info  Info
	Items []ResultItem
}

func (rf ResultFeed) toRSS() RSS {
	return RSS{
		Version: "2.0",
		Channel: Channel{
			Category:    rf.Info.Category,
			Description: rf.Info.Description,
			Items:       rf.Items,
			Language:    rf.Info.Language,
			Link:        rf.Info.Link,
			Title:       rf.Info.Title,
		},
		AtomNamespace:    "http://www.w3.org/2005/Atom",
		TorznabNamespace: "http://torznab.com/schemas/2015/feed",
	}
}

func (rf ResultFeed) MarshalJSON() ([]byte, error) {
	rss := rf.toRSS()
	jsonRSS := jsonRSS{
		RSS: rss,
	}
	jsonRSS.Attributes.Version = rss.Version
	return json.Marshal(jsonRSS)
}

func (rf ResultFeed) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.Encode(rf.toRSS())
}
