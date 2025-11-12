package jackett

import (
	"encoding/xml"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type IndexerType string

const (
	IndexerTypePublic      IndexerType = "public"
	IndexerTypePrivate     IndexerType = "private"
	IndexerTypeSemiPrivate IndexerType = "semi-private"
)

type IndexerDetails struct {
	XMLName     xml.Name    `xml:"indexer"`
	ID          string      `xml:"id,attr"`
	Configured  string      `xml:"configured,attr"`
	Title       string      `xml:"title"`
	Description string      `xml:"description"`
	Link        string      `xml:"link"`
	Language    string      `xml:"language"`
	Type        IndexerType `xml:"type"`
	Caps        tznc.Caps   `xml:"caps"`
}

type IndexersResponse struct {
	XMLName  xml.Name         `xml:"indexers"`
	Indexers []IndexerDetails `xml:"indexer"`
}

type ItemJackettIndexer struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type ChannelItem struct {
	Title          string             `xml:"title"`
	GUID           string             `xml:"guid"`
	JackettIndexer ItemJackettIndexer `xml:"jackettindexer"`
	Type           IndexerType        `xml:"type"`
	Comments       string             `xml:"comments"`
	PubDate        string             `xml:"pubDate"` // Mon, 02 Jan 2006 15:04:05 -0700
	Size           int64              `xml:"size"`
	Grabs          int                `xml:"grabs"`
	Description    string             `xml:"description"`
	Link           string             `xml:"link"`
	Categories     []int              `xml:"category"`
	Enclosure      tznc.ItemEnclosure `xml:"enclosure"`
	TorznabAttrs   tznc.TorznabAttrs  `xml:"http://torznab.com/schemas/2015/feed attr"`
}

func (o ChannelItem) ToTorz() *tznc.Torz {
	t := &tznc.Torz{}
	t.Indexer = o.JackettIndexer.ID
	t.Hash = strings.ToLower(o.TorznabAttrs.Get(tznc.TorznabAttrNameInfoHash))
	t.Title = o.Title
	t.Size = o.Size
	t.Seeders = util.SafeParseInt(o.TorznabAttrs.Get(tznc.TorznabAttrNameSeeders), 0)
	if peers := util.SafeParseInt(o.TorznabAttrs.Get(tznc.TorznabAttrNamePeers), 0); peers > t.Seeders {
		t.Leechers = peers - t.Seeders
	}
	t.Private = o.Type == IndexerTypePrivate || o.Type == IndexerTypeSemiPrivate
	if strings.HasPrefix(o.Enclosure.URL, "magnet:?") {
		t.MagnetLink = o.Enclosure.URL
		if t.Hash == "" {
			if m, err := core.ParseMagnetLink(t.MagnetLink); err == nil {
				t.Hash = m.Hash
			}
		}
	} else if strings.HasPrefix(o.Enclosure.URL, "http") {
		t.SourceLink = o.Enclosure.URL
	}
	return t
}

type Channel struct {
	XMLName     xml.Name      `xml:"channel"`
	Title       string        `xml:"title,omitempty"`
	Description string        `xml:"description,omitempty"`
	Link        string        `xml:"link,omitempty"`
	Language    string        `xml:"language,omitempty"`
	Category    string        `xml:"category,omitempty"`
	Items       []ChannelItem `xml:"item"`
}

type SearchResponse struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr,omitempty"`
	Channel Channel  `xml:"channel"`
}
