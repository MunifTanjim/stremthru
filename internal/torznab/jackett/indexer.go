package jackett

import (
	"encoding/xml"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
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
	znab.ChannelItem
	Comments       string                `xml:"comments"`
	Grabs          int                   `xml:"grabs"`
	JackettIndexer ItemJackettIndexer    `xml:"jackettindexer"`
	Size           int64                 `xml:"size"`
	Type           IndexerType           `xml:"type"`
	Attributes     znab.ChannelItemAttrs `xml:"http://torznab.com/schemas/2015/feed attr"`
}

func (o ChannelItem) ToTorz() *tznc.Torz {
	t := &tznc.Torz{}
	t.Indexer = o.JackettIndexer.ID
	t.Hash = strings.ToLower(o.Attributes.Get(znab.TorznabAttrNameInfoHash))
	t.Title = o.Title
	t.Size = o.Size
	t.Seeders = util.SafeParseInt(o.Attributes.Get(znab.TorznabAttrNameSeeders), 0)
	if peers := util.SafeParseInt(o.Attributes.Get(znab.TorznabAttrNamePeers), 0); peers > t.Seeders {
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
	znab.Channel[ChannelItem]
}

type SearchResponse struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr,omitempty"`
	Channel Channel  `xml:"channel"`
}
