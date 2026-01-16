package znab

const (
	// Common attributes (ALL categories)

	NewznabAttrNameCategory   ChannelItemAttrName = "category"   // required - Item's category ID
	NewznabAttrNameSize       ChannelItemAttrName = "size"       // required - Size in bytes
	NewznabAttrNameFiles      ChannelItemAttrName = "files"      // Number of files
	NewznabAttrNamePoster     ChannelItemAttrName = "poster"     // NNTP Poster
	NewznabAttrNameGroup      ChannelItemAttrName = "group"      // NNTP Group(s)
	NewznabAttrNamePassword   ChannelItemAttrName = "password"   // 0=no, 1=rar pass, 2=inner archive
	NewznabAttrNameGrabs      ChannelItemAttrName = "grabs"      // Number of times downloaded
	NewznabAttrNameComments   ChannelItemAttrName = "comments"   // Number of comments
	NewznabAttrNameUsenetDate ChannelItemAttrName = "usenetdate" // Date posted to usenet
	NewznabAttrNameInfo       ChannelItemAttrName = "info"       // Info URL
	NewznabAttrNameGUID       ChannelItemAttrName = "guid"       // GUID
	NewznabAttrNameYear       ChannelItemAttrName = "year"       // Release year
	NewznabAttrNameTeam       ChannelItemAttrName = "team"       // Team doing the release
	NewznabAttrNameGenre      ChannelItemAttrName = "genre"      // Genre
	NewznabAttrNameNFO        ChannelItemAttrName = "nfo"        // Contains NFO (1/0)
	NewznabAttrNameSHA1       ChannelItemAttrName = "sha1"       // SHA1 hash
	NewznabAttrNamePrematch   ChannelItemAttrName = "prematch"   // Has valid PreDB match (0/1)
	NewznabAttrNameLanguage   ChannelItemAttrName = "language"   // Content language
	NewznabAttrNameSubs       ChannelItemAttrName = "subs"       // Subtitle languages
	NewznabAttrNameReview     ChannelItemAttrName = "review"     // Review text/description

	// TV attributes

	NewznabAttrNameSeason     ChannelItemAttrName = "season"     // Numeric season
	NewznabAttrNameEpisode    ChannelItemAttrName = "episode"    // Numeric episode
	NewznabAttrNameEp         ChannelItemAttrName = "ep"         // Episode (alternative)
	NewznabAttrNameTVDBId     ChannelItemAttrName = "tvdbid"     // TVDB ID
	NewznabAttrNameTVRageId   ChannelItemAttrName = "rageid"     // TVRage ID
	NewznabAttrNameTVRageId2  ChannelItemAttrName = "tvrageid"   // TVRage ID (alternative)
	NewznabAttrNameTVMazeId   ChannelItemAttrName = "tvmazeid"   // TVMaze ID
	NewznabAttrNameTraktId    ChannelItemAttrName = "traktid"    // Trakt.tv ID
	NewznabAttrNameTVTitle    ChannelItemAttrName = "tvtitle"    // TV Show Title
	NewznabAttrNameFirstAired ChannelItemAttrName = "firstaired" // TV Show Air date

	// Movie/TV shared attributes

	NewznabAttrNameIMDB     ChannelItemAttrName = "imdb"     // IMDB ID (numeric, without "tt" prefix)
	NewznabAttrNameIMDBId   ChannelItemAttrName = "imdbid"   // IMDB ID (alternative)
	NewznabAttrNameTMDBId   ChannelItemAttrName = "tmdbid"   // TMDB ID
	NewznabAttrNameDoubanId ChannelItemAttrName = "doubanid" // Douban ID (Asian content)

	// Video/Audio quality attributes

	NewznabAttrNameVideo       ChannelItemAttrName = "video"       // Video codec
	NewznabAttrNameAudio       ChannelItemAttrName = "audio"       // Audio codec
	NewznabAttrNameResolution  ChannelItemAttrName = "resolution"  // Video resolution
	NewznabAttrNameCoverURL    ChannelItemAttrName = "coverurl"    // Cover image URL
	NewznabAttrNameBackdropURL ChannelItemAttrName = "backdropurl" // Backdrop/fanart image URL
	NewznabAttrNameBannerURL   ChannelItemAttrName = "bannerurl"   // Banner image URL

	// Music attributes

	NewznabAttrNameArtist        ChannelItemAttrName = "artist"        // Artist name
	NewznabAttrNameAlbum         ChannelItemAttrName = "album"         // Album name
	NewznabAttrNameLabel         ChannelItemAttrName = "label"         // Record label
	NewznabAttrNameTracks        ChannelItemAttrName = "tracks"        // Number of tracks
	NewznabAttrNameMusicBrainzId ChannelItemAttrName = "musicbrainzid" // MusicBrainz ID

	// Book attributes

	NewznabAttrNameAuthor      ChannelItemAttrName = "author"      // Author name
	NewznabAttrNameBookTitle   ChannelItemAttrName = "booktitle"   // Book title
	NewznabAttrNamePublisher   ChannelItemAttrName = "publisher"   // Publisher
	NewznabAttrNamePublishDate ChannelItemAttrName = "publishdate" // Publication date

	// Console/Games attributes

	NewznabAttrNamePlatform ChannelItemAttrName = "platform" // Gaming platform (Xbox360, PS3, etc.)
)
