package torznab_client

type TorznabAttrName string

const (
	TorznabAttrNameCategory             TorznabAttrName = "category"
	TorznabAttrNameInfoHash             TorznabAttrName = "infohash"
	TorznabAttrNameMagnetURL            TorznabAttrName = "magneturl"
	TorznabAttrNameIMDB                 TorznabAttrName = "imdb"
	TorznabAttrNameIMDBId               TorznabAttrName = "imdbid"
	TorznabAttrNameGenre                TorznabAttrName = "genre"
	TorznabAttrNameSeeders              TorznabAttrName = "seeders"
	TorznabAttrNamePeers                TorznabAttrName = "peers"
	TorznabAttrNameMinimumRatio         TorznabAttrName = "minimumratio"
	TorznabAttrNameMinimumSeedTime      TorznabAttrName = "minimumseedtime"
	TorznabAttrNameDownloadVolumeFactor TorznabAttrName = "downloadvolumefactor"
	TorznabAttrNameUploadVolumeFactor   TorznabAttrName = "uploadvolumefactor"
	TorznabAttrNameCoverURL             TorznabAttrName = "coverurl"
)

type TorznabAttr struct {
	Name  TorznabAttrName `xml:"name,attr"`
	Value string          `xml:"value,attr"`
}

type TorznabAttrs []TorznabAttr

func (attrs TorznabAttrs) Get(name TorznabAttrName) string {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value
		}
	}
	return ""
}

func (attrs TorznabAttrs) GetAll(name TorznabAttrName) []string {
	values := []string{}
	for _, attr := range attrs {
		if attr.Name == name {
			values = append(values, attr.Value)
		}
	}
	return values
}

type ItemEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"` // application/x-bittorrent
}
