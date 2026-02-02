package newznab_client

import (
	"encoding/xml"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Newz struct {
	// Core metadata

	Title       string
	GUID        string
	PublishDate time.Time
	Size        int64

	// NZB-specific attributes

	Files        int       // Number of files in the NZB
	Poster       string    // Usenet poster
	Group        string    // Primary newsgroup
	Grabs        int       // Download count
	Comments     int       // Number of comments
	Password     bool      // Whether the release is password protected
	InnerArchive bool      // Whether the release contains inner archive
	Date         time.Time // Original upload date

	// Category info

	Categories []string // Category IDs

	// Media metadata (optional, from extended attributes)

	IMDB    string // IMDB ID
	TVDB    string // TVDB ID
	TVRage  string // TVRage ID
	Season  string // Season number
	Episode string // Episode number

	DownloadLink string // Direct NZB download URL

	Hash string

	LockedDownload bool
	LockedProvider string
}

func (n *Newz) Age() time.Duration {
	return time.Since(n.Date)
}

func (n *Newz) GetHash() string {
	if n.Hash == "" {
		n.Hash = shared.HashNZBDownloadLink(n.DownloadLink)
	}
	return n.Hash
}

type Indexer interface {
	GetId() string
	NewSearchQuery(fn func(caps *znab.Caps) Function) (*Query, error)
	Search(query url.Values) ([]Newz, error)
}

type ChannelItem struct {
	znab.ChannelItem
	Size       int64                 `xml:"size"`
	Comments   string                `xml:"comments"`
	Grabs      int                   `xml:"grabs"`
	Attributes znab.ChannelItemAttrs `xml:"http://www.newznab.com/DTD/2010/feeds/attributes/ attr"`
}

func (o ChannelItem) ToNewz() *Newz {
	nzb := &Newz{}

	nzb.Title = o.Title
	nzb.GUID = o.GUID
	nzb.PublishDate = o.GetPublishDate()
	nzb.Size = o.Size
	if nzb.Size == 0 {
		nzb.Size = o.Enclosure.Length
	}

	nzb.Files = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameFiles), 0)
	nzb.Poster = o.Attributes.Get(znab.NewznabAttrNamePoster)
	nzb.Group = o.Attributes.Get(znab.NewznabAttrNameGroup)
	nzb.Grabs = o.Grabs
	if nzb.Grabs == 0 {
		nzb.Grabs = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameGrabs), 0)
	}
	nzb.Comments = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameComments), 0)
	if password := o.Attributes.Get(znab.NewznabAttrNamePassword); password == "2" {
		nzb.Password = true
		nzb.InnerArchive = true
	} else {
		nzb.Password = util.StringToBool(o.Attributes.Get(znab.NewznabAttrNamePassword), false)
	}
	if t, err := time.Parse(znab.TimeFormat, o.Attributes.Get(znab.NewznabAttrNameUsenetDate)); err == nil {
		nzb.Date = t
	}

	nzb.Categories = o.Attributes.GetAll("category")
	if len(nzb.Categories) == 0 && util.IsNumericString(o.Category) {
		nzb.Categories = []string{o.Category}
	}

	nzb.IMDB = o.Attributes.Get(znab.NewznabAttrNameIMDB)
	if nzb.IMDB != "" {
		nzb.IMDB = "tt" + nzb.IMDB
	} else {
		nzb.IMDB = o.Attributes.Get(znab.NewznabAttrNameIMDBId)
	}
	nzb.TVDB = o.Attributes.Get(znab.NewznabAttrNameTVDBId)
	nzb.TVRage = o.Attributes.Get(znab.NewznabAttrNameTVRageId)
	nzb.Season = o.Attributes.Get(znab.NewznabAttrNameSeason)
	nzb.Episode = o.Attributes.Get(znab.NewznabAttrNameEpisode)

	nzb.DownloadLink = o.Enclosure.URL

	return nzb
}

type Channel struct {
	znab.Channel[ChannelItem]
}

type SearchResponse struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr,omitempty"`
	Channel Channel  `xml:"channel"`
}

func (c *Client) Search(query url.Values) ([]Newz, error) {
	params := &Ctx{}
	params.Query = &query

	var resp Response[SearchResponse]
	_, err := c.Request("GET", "/api", params, &resp)
	if err != nil {
		return nil, err
	}

	items := resp.Data.Channel.Items
	result := make([]Newz, 0, len(items))
	for i := range items {
		item := &items[i]
		if item.Size == 0 && item.Enclosure.Length == 0 {
			continue
		}
		result = append(result, *item.ToNewz())
	}
	return result, nil
}

func (c *Client) GetId() string {
	return c.BaseURL.Host
}
