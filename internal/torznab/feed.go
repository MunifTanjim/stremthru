package torznab

import (
	"encoding/json"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type ChannelItem struct {
	znab.ChannelItem
	Attributes znab.ChannelItemAttrs `xml:"torznab:attr" json:"attr"`
}

type FeedItem struct {
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

func (ri FeedItem) toChannelItem() ChannelItem {
	attrs := znab.ChannelItemAttrs{}
	if ri.Audio != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "audio", Value: ri.Audio})
	}
	attrs = append(attrs, znab.ChannelItemAttr{Name: "category", Value: strconv.Itoa(ri.Category.ID)})
	if ri.IMDB != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "imdb", Value: strings.TrimPrefix(ri.IMDB, "tt")})
	}
	if ri.InfoHash != "" {
		if magnet, err := core.ParseMagnetLink(ri.InfoHash); err == nil {
			attrs = append(
				attrs,
				znab.ChannelItemAttr{Name: "infohash", Value: magnet.Hash},
				znab.ChannelItemAttr{Name: "magneturl", Value: magnet.Link},
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
		attrs = append(attrs, znab.ChannelItemAttr{Name: "language", Value: ri.Language})
	}
	if ri.Leechers > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "leechers", Value: strconv.Itoa(ri.Leechers)})
	}
	if ri.Resolution != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "resolution", Value: ri.Resolution})
	}
	if ri.Seeders > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "seeders", Value: strconv.Itoa(ri.Seeders)})
	}
	if ri.Site != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "site", Value: ri.Site})
	}
	if ri.Size > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "size", Value: strconv.FormatInt(ri.Size, 10)})
	}
	if ri.Codec != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "video", Value: ri.Codec})
	}
	if ri.Year != 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "year", Value: strconv.Itoa(ri.Year)})
	}

	return ChannelItem{
		ChannelItem: znab.ChannelItem{
			Category:    ri.Category.Name,
			Description: ri.Description,
			Files:       ri.Files,
			GUID:        ri.GUID,
			Link:        ri.Link,
			PublishDate: ri.PublishDate.Format(znab.TimeFormat),
			Title:       ri.Title,
			Enclosure: znab.ChannelItemEnclosure{
				URL:    ri.Link,
				Length: ri.Size,
				Type:   "application/x-bittorrent;x-scheme-handler/magnet",
			},
		},
		Attributes: attrs,
	}
}

type FeedItems []FeedItem

func (fis FeedItems) toChannelItems() []ChannelItem {
	items := make([]ChannelItem, len(fis))
	for i, fi := range fis {
		items[i] = fi.toChannelItem()
	}
	return items
}

type Feed struct {
	Info  znab.Info
	Items FeedItems
}

func (rf Feed) toRSS() znab.RSSFeed[ChannelItem] {
	return znab.RSSFeed[ChannelItem]{
		Version: "2.0",
		Channel: znab.Channel[ChannelItem]{
			Category:    rf.Info.Category,
			Description: rf.Info.Description,
			Items:       rf.Items.toChannelItems(),
			Language:    rf.Info.Language,
			Link:        rf.Info.Link,
			Title:       rf.Info.Title,
		},
		AtomNamespace:    "http://www.w3.org/2005/Atom",
		TorznabNamespace: "http://torznab.com/schemas/2015/feed",
	}
}

func (rf Feed) MarshalJSON() ([]byte, error) {
	jsonFeed := znab.JSONFeed[ChannelItem]{
		RSSFeed: rf.toRSS(),
	}
	jsonFeed.Attributes.Version = jsonFeed.RSSFeed.Version
	return json.Marshal(jsonFeed)
}

func (rf Feed) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.Encode(rf.toRSS())
}
