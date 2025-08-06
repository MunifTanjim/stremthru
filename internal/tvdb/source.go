package tvdb

type SourceType int

const (
	SourceTypeIMDB            SourceType = 2
	SourceTypeZap2It          SourceType = 3
	SourceTypeOfficial        SourceType = 4
	SourceTypeFacebook        SourceType = 5
	SourceTypeTwitter         SourceType = 6
	SourceTypeReddit          SourceType = 7
	SourceTypeFanSite         SourceType = 8
	SourceTypeInstagram       SourceType = 9
	SourceTypeTMDB            SourceType = 10
	SourceTypeYouTube         SourceType = 11
	SourceTypeTMDBTV          SourceType = 12
	SourceTypeEIDR            SourceType = 13
	SourceTypeEIDRParty       SourceType = 14
	SourceTypeTMDBPerson      SourceType = 15
	SourceTypeIMDBPerson      SourceType = 16
	SourceTypeIMDBCompany     SourceType = 17
	SourceTypeWikidata        SourceType = 18
	SourceTypeTVMaze          SourceType = 19
	SourceTypeLinkedIn        SourceType = 20
	SourceTypeTVMazePerson    SourceType = 21
	SourceTypeTVMazeSeason    SourceType = 22
	SourceTypeTVMazeEpisode   SourceType = 23
	SourceTypeWikipedia       SourceType = 24
	SourceTypeTikTok          SourceType = 25
	SourceTypeLinkedInCompany SourceType = 26
	SourceTypeTVMazeCompany   SourceType = 27
	SourceTypeTMDBCollection  SourceType = 28
)

type RemoteId struct {
	Id         string     `json:"id"`
	Type       SourceType `json:"type"`
	SourceName string     `json:"sourceName"`
}
