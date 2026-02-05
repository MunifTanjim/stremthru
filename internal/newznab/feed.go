package newznab

import (
	"encoding/json"
	"encoding/xml"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/znab"
)

type ChannelItem struct {
	znab.ChannelItem
	Attributes znab.ChannelItemAttrs `xml:"newznab:attr" json:"attr"`
}

type FeedItem struct {
	Category    Category
	Description string
	Files       int
	GUID        string
	Link        string
	PublishDate time.Time
	Title       string

	Poster     string
	Group      string
	Grabs      int
	Comments   int
	Password   bool
	UsenetDate time.Time

	Episode string
	IMDB    string
	Season  string
	Size    int64
	Year    int

	Indexer struct {
		ID   string
		Name string
	}
}

func (fi FeedItem) toChannelItem() ChannelItem {
	attrs := znab.ChannelItemAttrs{}

	attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameCategory, Value: strconv.Itoa(fi.Category.ID)})

	if fi.Size > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameSize, Value: strconv.FormatInt(fi.Size, 10)})
	}
	if fi.Files > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameFiles, Value: strconv.Itoa(fi.Files)})
	}
	if fi.Poster != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNamePoster, Value: fi.Poster})
	}
	if fi.Group != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameGroup, Value: fi.Group})
	}
	if fi.Grabs > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameGrabs, Value: strconv.Itoa(fi.Grabs)})
	}
	if fi.Comments > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameComments, Value: strconv.Itoa(fi.Comments)})
	}
	if fi.IMDB != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameIMDB, Value: strings.TrimPrefix(fi.IMDB, "tt")})
	}
	if fi.Season != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameSeason, Value: fi.Season})
	}
	if fi.Episode != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameEpisode, Value: fi.Episode})
	}
	if fi.Year > 0 {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameYear, Value: strconv.Itoa(fi.Year)})
	}
	if !fi.UsenetDate.IsZero() {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNameUsenetDate, Value: fi.UsenetDate.Format(znab.TimeFormat)})
	}
	if fi.Password {
		attrs = append(attrs, znab.ChannelItemAttr{Name: znab.NewznabAttrNamePassword, Value: "1"})
	}
	if fi.Indexer.Name != "" {
		attrs = append(attrs, znab.ChannelItemAttr{Name: "indexer", Value: fi.Indexer.Name})
	}

	return ChannelItem{
		ChannelItem: znab.ChannelItem{
			Category:    fi.Category.Name,
			Description: fi.Description,
			Files:       fi.Files,
			GUID:        fi.GUID,
			Link:        fi.Link,
			PublishDate: fi.PublishDate.Format(znab.TimeFormat),
			Title:       fi.Title,
			Enclosure: znab.ChannelItemEnclosure{
				URL:    fi.Link,
				Length: fi.Size,
				Type:   "application/x-nzb",
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

type RSSFeed struct {
	XMLName          xml.Name                  `xml:"rss" json:"-"`
	AtomNamespace    string                    `xml:"xmlns:atom,attr" json:"-"`
	NewznabNamespace string                    `xml:"xmlns:newznab,attr" json:"-"`
	Version          string                    `xml:"version,attr,omitempty" json:"-"`
	Channel          znab.Channel[ChannelItem] `xml:"channel" json:"channel"`
}

func (rf Feed) toRSS() RSSFeed {
	return RSSFeed{
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
		NewznabNamespace: "http://www.newznab.com/DTD/2010/feeds/attributes/",
	}
}

type jsonFeed struct {
	Attributes struct {
		Version string `json:"version,omitempty"`
	} `json:"@attributes"`
	RSSFeed
}

func (rf Feed) MarshalJSON() ([]byte, error) {
	jf := jsonFeed{
		RSSFeed: rf.toRSS(),
	}
	jf.Attributes.Version = jf.RSSFeed.Version
	return json.Marshal(jf)
}

func (rf Feed) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.Encode(rf.toRSS())
}
