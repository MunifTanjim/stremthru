package jackett

import (
	"encoding/xml"

	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
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
	Type           IndexerType           `xml:"type"`
	Attributes     znab.ChannelItemAttrs `xml:"http://torznab.com/schemas/2015/feed attr"`
}

func (o ChannelItem) ToTorz() *tznc.Torz {
	t := tznc.TorzFromChannelItem(&o.ChannelItem, o.Attributes)
	t.Indexer = o.JackettIndexer.ID
	t.Private = o.Type == IndexerTypePrivate || o.Type == IndexerTypeSemiPrivate
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
