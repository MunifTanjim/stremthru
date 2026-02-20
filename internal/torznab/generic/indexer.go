package generic

import (
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type ChannelItem struct {
	znab.ChannelItem
	Attributes znab.ChannelItemAttrs `xml:"http://torznab.com/schemas/2015/feed attr"`
}

func (o ChannelItem) ToTorz() *tznc.Torz {
	t := &tznc.Torz{}
	t.Hash = strings.ToLower(o.Attributes.Get(znab.TorznabAttrNameInfoHash))
	t.Title = o.Title
	t.Size = o.Enclosure.Length
	t.Seeders = util.SafeParseInt(o.Attributes.Get(znab.TorznabAttrNameSeeders), 0)
	if peers := util.SafeParseInt(o.Attributes.Get(znab.TorznabAttrNamePeers), 0); peers > t.Seeders {
		t.Leechers = peers - t.Seeders
	}
	if dvf := o.Attributes.Get(znab.TorznabAttrNameDownloadVolumeFactor); dvf != "" && dvf != "0" {
		t.Private = true
	}
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
	XMLName struct{} `xml:"rss"`
	Version string   `xml:"version,attr,omitempty"`
	Channel Channel  `xml:"channel"`
}
