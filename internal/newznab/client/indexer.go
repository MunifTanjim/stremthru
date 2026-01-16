package newznab_client

import (
	"encoding/xml"
	"net/url"

	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

// Newz represents a search result item from a Newznab indexer
type Newz struct {
	Indexer string

	// Core metadata
	Title       string
	GUID        string
	Size        int64
	PublishDate string // RFC822 format: "Mon, 02 Jan 2006 15:04:05 -0700"

	// NZB-specific attributes
	Files    int    // Number of files in the NZB
	Poster   string // Usenet poster
	Group    string // Primary newsgroup
	Grabs    int    // Download count
	Comments int    // Number of comments
	Password bool   // Whether the release is password protected

	// Category info
	Categories []int // Category IDs

	// Media metadata (optional, from extended attributes)
	IMDB    string // IMDB ID (without "tt" prefix)
	TVDB    string // TVDB ID
	TVRage  string // TVRage ID
	Season  string // Season number
	Episode string // Episode number

	// Download link
	DownloadLink string // Direct NZB download URL
	DetailsLink  string // Link to details page
}

type Indexer interface {
	GetId() string
	NewSearchQuery(fn func(caps znab.Caps) Function) (*Query, error)
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

	// Core metadata
	nzb.Title = o.Title
	nzb.GUID = o.GUID
	nzb.PublishDate = o.PublishDate
	nzb.Size = o.Size
	if nzb.Size == 0 {
		nzb.Size = o.Enclosure.Length
	}

	// NZB-specific attributes
	nzb.Files = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameFiles), 0)
	nzb.Poster = o.Attributes.Get(znab.NewznabAttrNamePoster)
	nzb.Group = o.Attributes.Get(znab.NewznabAttrNameGroup)
	nzb.Grabs = o.Grabs
	if nzb.Grabs == 0 {
		nzb.Grabs = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameGrabs), 0)
	}
	nzb.Comments = util.SafeParseInt(o.Attributes.Get(znab.NewznabAttrNameComments), 0)
	nzb.Password = util.StringToBool(o.Attributes.Get(znab.NewznabAttrNamePassword), false)

	// Categories
	// nzb.Categories = o.Category

	// Media metadata
	nzb.IMDB = o.Attributes.Get(znab.NewznabAttrNameIMDB)
	if nzb.IMDB == "" {
		nzb.IMDB = o.Attributes.Get(znab.NewznabAttrNameIMDBId)
	}
	nzb.TVDB = o.Attributes.Get(znab.NewznabAttrNameTVDBId)
	nzb.TVRage = o.Attributes.Get(znab.NewznabAttrNameTVRageId)
	nzb.Season = o.Attributes.Get(znab.NewznabAttrNameSeason)
	nzb.Episode = o.Attributes.Get(znab.NewznabAttrNameEpisode)

	// Download links
	nzb.DownloadLink = o.Enclosure.URL
	nzb.DetailsLink = o.Link

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
