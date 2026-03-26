package generic

import (
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type ChannelItem struct {
	znab.ChannelItem
	Attributes znab.ChannelItemAttrs `xml:"http://torznab.com/schemas/2015/feed attr"`
}

func (o ChannelItem) ToTorz() *tznc.Torz {
	t := tznc.TorzFromChannelItem(&o.ChannelItem, o.Attributes)
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
