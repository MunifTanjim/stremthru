package worker

import (
	"encoding/xml"
	"strconv"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/animelists"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/stretchr/testify/assert"
)

func TestPrepareAniDBTorrentsFromTVDBEpisodeMaps(t *testing.T) {
	makeTorrentInfo := func(title string) torrent_info.TorrentInfo {
		tInfo := torrent_info.TorrentInfo{TorrentTitle: title}
		err := tInfo.Parse()
		if err != nil {
			panic(err)
		}
		return tInfo
	}

	toAnimeListItems := func(xmlContent string) []animelists.AnimeListItem {
		parsed := struct {
			Items []animelists.AnimeListItem `xml:"anime"`
		}{}
		err := xml.Unmarshal([]byte("<anime-list>"+xmlContent+"</anime-list>"), &parsed)
		if err != nil {
			panic(err)
		}
		return parsed.Items
	}

	toEpisodeMaps := func(xmlContent string) anidb.AniDBTVDBEpisodeMaps {
		items := toAnimeListItems(xmlContent)
		return animelists.PrepareAniDBTVDBEpisodeMaps(items[0].TVDBId, items)
	}

	type testCase struct {
		tInfo  torrent_info.TorrentInfo
		result []torrentMap
	}

	for _, tc := range []struct {
		tvdbMaps anidb.AniDBTVDBEpisodeMaps
		titles   []anidb.AniDBTitle
		cases    []testCase
	}{
		{
			toEpisodeMaps(`
		<anime anidbid="11123" tvdbid="293088" defaulttvdbseason="1">
		  <name>One Punch Man</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-2;2-3;3-4;4-5;5-6;6-7;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="11637" tvdbid="293088" defaulttvdbseason="0">
		  <name>One Punch Man: Road to Hero</name>
		</anime>

		<anime anidbid="12430" tvdbid="293088" defaulttvdbseason="2">
		  <name>One Punch Man (2019)</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-8;2-9;3-10;4-11;5-12;6-13;7-14</mapping>
		  </mapping-list>
		  <before>;1-1;2-3;</before>
		</anime>

		<anime anidbid="17576" tvdbid="293088" defaulttvdbseason="3">
		  <name>One Punch Man (2025)</name>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "11123", Value: "One Punch Man", Season: "1"},
				{TId: "11637", Value: "One Punch Man OVA", Season: "1"},
				{TId: "11637", Value: "One Punch Man: Road to Hero", Season: "1"},
				{TId: "12430", Value: "One Punch Man", Season: "2", Year: "2019"},
				{TId: "12430", Value: "One Punch Man (2019)", Season: "2", Year: "2019"},
				{TId: "17576", Value: "One Punch Man", Season: "3", Year: "2025"},
				{TId: "17576", Value: "One Punch Man (2025)", Season: "3", Year: "2025"},
			},
			[]testCase{
				{
					makeTorrentInfo("[LostYears] One Punch Man - S02E07 (WEB 1080p x264 10-bit AAC) [5EA9AF2F].mkv"),
					[]torrentMap{
						{
							anidbId:      "12430",
							seasonType:   "ani",
							season:       2,
							episodeStart: 7,
							episodeEnd:   7,
						},
						{
							anidbId:      "12430",
							seasonType:   "tv",
							season:       2,
							episodeStart: 7,
							episodeEnd:   7,
						},
					},
				},
				{
					makeTorrentInfo("[GbR] One Punch Man - 02 [2160p H.265].mkv"),
					[]torrentMap{
						{
							anidbId:      "11123",
							seasonType:   "ani",
							season:       1,
							episodeStart: 2,
							episodeEnd:   2,
						},
						{
							anidbId:      "11123",
							seasonType:   "tv",
							season:       1,
							episodeStart: 2,
							episodeEnd:   2,
						},
					},
				},
				{
					makeTorrentInfo("[AnimeRG] One Punch Man (2019) (Season 2 Complete) (00-12) [1080p] [Eng Subbed] [JRR]"),
					[]torrentMap{
						{
							anidbId:      "12430",
							seasonType:   "ani",
							season:       2,
							episodeStart: 0,
							episodeEnd:   12,
						},
						{
							anidbId:      "12430",
							seasonType:   "tv",
							season:       2,
							episodeStart: 0,
							episodeEnd:   12,
						},
					},
				},
				{
					makeTorrentInfo("[Anime Time] One Punch Man [S1+S2+OVA&ODA][Dual Audio][1080p BD][HEVC 10bit x265][AAC][Eng Subs]"),
					[]torrentMap{
						{
							anidbId:    "11123",
							seasonType: "ani",
							season:     1,
						},
						{
							anidbId:    "11123",
							seasonType: "tv",
							season:     1,
						},
						{
							anidbId:    "12430",
							seasonType: "ani",
							season:     2,
						},
						{
							anidbId:    "12430",
							seasonType: "tv",
							season:     2,
						},
					},
				},
				{
					makeTorrentInfo("[sam] One Punch Man OVA [BD 1080p FLAC]"),
					[]torrentMap{
						{
							anidbId:    "11637",
							seasonType: "ani",
							season:     1,
						},
						{
							anidbId:    "11637",
							seasonType: "tv",
							season:     0,
						},
					},
				},
				{
					makeTorrentInfo("[Blaze077] One Punch Man - OVA-  Road To Hero [720p].mkv"),
					[]torrentMap{
						{
							anidbId:    "11637",
							seasonType: "ani",
							season:     1,
						},
						{
							anidbId:    "11637",
							seasonType: "tv",
							season:     0,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="6662" tvdbid="114801" defaulttvdbseason="a">
		  <name>Fairy Tail</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="1" start="1" end="48" offset="0"/>
		    <mapping anidbseason="1" tvdbseason="2" start="49" end="96" offset="-48"/>
		    <mapping anidbseason="1" tvdbseason="3" start="97" end="150" offset="-96"/>
		    <mapping anidbseason="1" tvdbseason="4" start="151" end="175" offset="-150"/>
		  </mapping-list>
		</anime>

		<anime anidbid="8132" tvdbid="114801" defaulttvdbseason="0">
		  <name>Fairy Tail (2011)</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;4-5;5-7;6-8;7-9;8-10;9-11;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="8788" tvdbid="114801" defaulttvdbseason="0" episodeoffset="3" tmdbid="135531" imdbid="tt2085795">
		  <name>Gekijouban Fairy Tail: Houou no Miko</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-6;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="9980" tvdbid="114801" defaulttvdbseason="a" episodeoffset="175">
		  <name>Fairy Tail (2014)</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="5" start="1" end="51" offset="0"/>
		    <mapping anidbseason="1" tvdbseason="6" start="52" end="90" offset="-51"/>
		    <mapping anidbseason="1" tvdbseason="7" start="91" end="102" offset="-90"/>
		  </mapping-list>
		</anime>

		<anime anidbid="11247" tvdbid="114801" defaulttvdbseason="0" episodeoffset="11" tmdbid="433422" imdbid="tt6548966">
		  <name>Gekijouban Fairy Tail: Dragon Cry</name>
		</anime>

		<anime anidbid="13295" tvdbid="114801" defaulttvdbseason="a" episodeoffset="277">
		  <name>Fairy Tail (2018)</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="8" start="1" end="51" offset="0"/>
		  </mapping-list>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "6662", Value: "Fairy Tail", Season: "1"},
				{TId: "8132", Value: "Fairy Tail", Season: "1", Year: "2011"},
				{TId: "8788", Value: "Gekijouban Fairy Tail: Houou no Miko", Season: "1"},
				{TId: "9980", Value: "Fairy Tail", Season: "2", Year: "2014"},
				{TId: "11247", Value: "Gekijouban Fairy Tail: Dragon Cry", Season: "1"},
				{TId: "13295", Value: "Fairy Tail", Season: "3", Year: "2018"},
			},
			[]testCase{
				{
					makeTorrentInfo("[The_Wyandotte] Fairy Tail (2014) (h.264 BD 1080p Dual-Audio FLAC)"),
					[]torrentMap{
						{
							anidbId:      "9980",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 176,
							episodeEnd:   277,
						},
						{
							anidbId:      "9980",
							seasonType:   "ani",
							season:       2,
							episodeStart: 1,
							episodeEnd:   102,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       5,
							episodeStart: 1,
							episodeEnd:   51,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       6,
							episodeStart: 1,
							episodeEnd:   39,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       7,
							episodeStart: 1,
							episodeEnd:   12,
						},
					},
				},
				{
					makeTorrentInfo("[F-D] Fairy Tail Season 1 -6 + Extras [480P][Dual-Audio]"),
					[]torrentMap{
						{
							anidbId:      "6662",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 1,
							episodeEnd:   175,
						},
						{
							anidbId:      "6662",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   175,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   48,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       2,
							episodeStart: 1,
							episodeEnd:   48,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       3,
							episodeStart: 1,
							episodeEnd:   54,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       4,
							episodeStart: 1,
							episodeEnd:   25,
						},
						{
							anidbId:      "9980",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 176,
							episodeEnd:   265,
						},
						{
							anidbId:      "9980",
							seasonType:   "ani",
							season:       2,
							episodeStart: 1,
							episodeEnd:   90,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       5,
							episodeStart: 1,
							episodeEnd:   51,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       6,
							episodeStart: 1,
							episodeEnd:   39,
						},
					},
				},
				{
					makeTorrentInfo("[AnimeRG] Fairy Tail Final Series (2018) (278-328 Complete) [1080p] [JRR] (S3 01-51)"),
					[]torrentMap{
						{
							anidbId:      "13295",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 278,
							episodeEnd:   328,
						},
						{
							anidbId:      "13295",
							seasonType:   "ani",
							season:       3,
							episodeStart: 1,
							episodeEnd:   51,
						},
						{
							anidbId:      "13295",
							seasonType:   "tv",
							season:       8,
							episodeStart: 1,
							episodeEnd:   51,
						},
					},
				},
				{
					makeTorrentInfo("[HorribleRips] Fairy Tail 073-100 [1080p]"),
					[]torrentMap{
						{
							anidbId:      "6662",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 73,
							episodeEnd:   100,
						},
						{
							anidbId:      "6662",
							seasonType:   "ani",
							season:       1,
							episodeStart: 73,
							episodeEnd:   100,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       2,
							episodeStart: 25,
							episodeEnd:   48,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       3,
							episodeStart: 1,
							episodeEnd:   4,
						},
					},
				},
				{
					makeTorrentInfo("Fairy Tail S02E24 Um Mago da Fairy Tail 1080p MAX WEB-DL DDP2 0 H 264 DUAL-OLYMPUS.mkv"),
					[]torrentMap{
						{
							anidbId:      "6662",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 72,
							episodeEnd:   72,
						},
						{
							anidbId:      "6662",
							seasonType:   "ani",
							season:       1,
							episodeStart: 72,
							episodeEnd:   72,
						},
						{
							anidbId:      "6662",
							seasonType:   "tv",
							season:       2,
							episodeStart: 24,
							episodeEnd:   24,
						},
					},
				},
				{
					makeTorrentInfo("Fairy Tail S06E02"),
					[]torrentMap{
						{
							anidbId:      "9980",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 228,
							episodeEnd:   228,
						},
						{
							anidbId:      "9980",
							seasonType:   "ani",
							season:       2,
							episodeStart: 53,
							episodeEnd:   53,
						},
						{
							anidbId:      "9980",
							seasonType:   "tv",
							season:       6,
							episodeStart: 2,
							episodeEnd:   2,
						},
					},
				},
				{
					makeTorrentInfo("Fairy Tail Season 8 (278-328) (Bluray Remux Dual Audio TrueHD 5.1)"),
					[]torrentMap{
						{
							anidbId:      "13295",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 278,
							episodeEnd:   328,
						},
						{
							anidbId:      "13295",
							seasonType:   "ani",
							season:       3,
							episodeStart: 1,
							episodeEnd:   51,
						},
						{
							anidbId:      "13295",
							seasonType:   "tv",
							season:       8,
							episodeStart: 1,
							episodeEnd:   51,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="69" tvdbid="81797" defaulttvdbseason="a">
		  <name>One Piece</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-27;2-3;3-9;4-10;5-14;6-0;7-0;8-0;9-0;10-0;11-0;12-0;13-0;14-0;15-0;16-31;17-23;18-24;19-28;20-29;21-30;22-0;23-32;24-34;25-0;26-36;27-37;28-40;29-41;30-42;31-43;32-44;33-45;34-47;35-48;36-49;37-50;38-51;39-52;40-53;41-54;42-55;43-56;44-57;</mapping>
		    <mapping anidbseason="1" tvdbseason="2" start="9" end="30" offset="-8"/>
		    <mapping anidbseason="1" tvdbseason="3" start="31" end="47" offset="-30"/>
		    <mapping anidbseason="1" tvdbseason="4" start="48" end="60" offset="-47"/>
		    <mapping anidbseason="1" tvdbseason="5" start="61" end="69" offset="-60"/>
		    <mapping anidbseason="1" tvdbseason="6" start="70" end="91" offset="-69"/>
		    <mapping anidbseason="1" tvdbseason="7" start="92" end="130" offset="-91"/>
		    <mapping anidbseason="1" tvdbseason="8" start="131" end="143" offset="-130"/>
		    <mapping anidbseason="1" tvdbseason="9" start="144" end="195" offset="-143"/>
		    <mapping anidbseason="1" tvdbseason="10" start="196" end="226" offset="-195"/>
		    <mapping anidbseason="1" tvdbseason="11" start="227" end="325" offset="-226"/>
		    <mapping anidbseason="1" tvdbseason="12" start="326" end="381" offset="-325"/>
		    <mapping anidbseason="1" tvdbseason="13" start="382" end="481" offset="-381"/>
		    <mapping anidbseason="1" tvdbseason="14" start="482" end="516" offset="-481"/>
		    <mapping anidbseason="1" tvdbseason="15" start="517" end="578" offset="-516"/>
		    <mapping anidbseason="1" tvdbseason="16" start="579" end="628" offset="-578"/>
		    <mapping anidbseason="1" tvdbseason="17" start="629" end="746" offset="-628"/>
		    <mapping anidbseason="1" tvdbseason="18" start="747" end="779" offset="-746"/>
		    <mapping anidbseason="1" tvdbseason="19" start="780" end="877" offset="-779"/>
		    <mapping anidbseason="1" tvdbseason="20" start="878" end="891" offset="-877"/>
		    <mapping anidbseason="1" tvdbseason="21" start="892" end="1085" offset="-891"/>
		    <mapping anidbseason="1" tvdbseason="22" start="1086" offset="-1085"/>
		  </mapping-list>
		</anime>

		<anime anidbid="411" tvdbid="81797" defaulttvdbseason="0" episodeoffset="1" tmdbid="19576" imdbid="tt0814243">
		  <name>One Piece (2000)</name>
		</anime>

		<anime anidbid="893" tvdbid="81797" defaulttvdbseason="0" tmdbid="23446" imdbid="tt0832449">
		  <name>One Piece: Nejimakijima no Bouken</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-5;</mapping>
		    <mapping anidbseason="1" tvdbseason="0">;1-4;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="1253" tvdbid="81797" defaulttvdbseason="0" tmdbid="44725" imdbid="tt0997084">
		  <name>One Piece: Chinjuujima no Chopper Oukoku</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-7;</mapping>
		    <mapping anidbseason="1" tvdbseason="0">;1-6;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="1254" tvdbid="81797" defaulttvdbseason="0" tmdbid="44727" imdbid="tt1006926">
		  <name>One Piece The Movie: Dead End no Bouken</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-8;2-8;3-8;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="2036" tvdbid="81797" defaulttvdbseason="0" tmdbid="44728" imdbid="tt1010435">
		  <name>One Piece: Norowareta Seiken</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-12;</mapping>
		    <mapping anidbseason="1" tvdbseason="0">;1-11;2-11;3-11;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="2644" tvdbid="81797" defaulttvdbseason="0" episodeoffset="12" tmdbid="44729" imdbid="tt1018764">
		  <name>One Piece: Omatsuri Danshaku to Himitsu no Shima</name>
		</anime>

		<anime anidbid="2736" tvdbid="81797" defaulttvdbseason="0" episodeoffset="0" tmdbid="116315" imdbid="tt1012788">
		  <name>One Piece: Taose! Kaizoku Ganzack</name>
		</anime>

		<anime anidbid="4097" tvdbid="81797" defaulttvdbseason="0" episodeoffset="14" tmdbid="44730" imdbid="tt1059950">
		  <name>One Piece The Movie: Karakurijou no Mecha Kyohei</name>
		</anime>

		<anime anidbid="4851" tvdbid="81797" defaulttvdbseason="0" episodeoffset="15" tmdbid="25278" imdbid="tt1037116">
		  <name>One Piece: Episode of Arabasta - Sabaku no Oujo to Kaizoku-tachi</name>
		</anime>

		<anime anidbid="5691" tvdbid="81797" defaulttvdbseason="0" episodeoffset="16" tmdbid="44731" imdbid="tt1206326">
		  <name>One Piece The Movie: Episode of Chopper Plus - Fuyu ni Saku, Kiseki no Sakura</name>
		</anime>

		<anime anidbid="6199" tvdbid="81797" defaulttvdbseason="0" episodeoffset="17" tmdbid="422807" imdbid="unknown">
		  <name>One Piece Special: Romance Dawn Story</name>
		</anime>

		<anime anidbid="6537" tvdbid="81797" defaulttvdbseason="0" episodeoffset="18" tmdbid="41498" imdbid="tt1485763">
		  <name>One Piece Film: Strong World</name>
		</anime>

		<anime anidbid="7538" tvdbid="81797" defaulttvdbseason="0" episodeoffset="19" tmdbid="270413">
		  <name>One Piece Film: Strong World - Episode 0</name>
		</anime>

		<anime anidbid="8010" tvdbid="81797" defaulttvdbseason="0" episodeoffset="20" tmdbid="79082" imdbid="tt1865467">
		  <name>One Piece 3D: Mugiwara Chase</name>
		</anime>

		<anime anidbid="8762" tvdbid="81797" defaulttvdbseason="0" episodeoffset="21" tmdbid="462387" imdbid="tt27639430">
		  <name>One Piece 3D: Gekisou! Trap Coaster</name>
		</anime>

		<anime anidbid="8940" tvdbid="81797" defaulttvdbseason="0" tmdbid="176983" imdbid="tt2375379">
		  <name>One Piece Film: Z</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-25;2-25;3-25;</mapping>
		    <mapping anidbseason="1" tvdbseason="0">;1-26;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="11529" tvdbid="81797" defaulttvdbseason="0" tmdbid="374205" imdbid="tt5251328">
		  <name>One Piece Film: Gold</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-35;</mapping>
		    <mapping anidbseason="1" tvdbseason="0">;1-33;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="14318" tvdbid="81797" defaulttvdbseason="0" episodeoffset="37" tmdbid="568012" imdbid="tt9430698">
		  <name>Gekijouban One Piece: Stampede</name>
		</anime>

		<anime anidbid="16983" tvdbid="81797" defaulttvdbseason="0" episodeoffset="45" tmdbid="900667" imdbid="tt16183464">
		  <name>One Piece Film: Red</name>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "69", Value: "One Piece", Season: "1"},
				{TId: "411", Value: "One Piece", Season: "1", Year: "2000"},
				{TId: "411", Value: "One Piece The Movie", Season: "1", Year: "2000"},
				{TId: "893", Value: "One Piece: Nejimakijima no Bouken", Season: "1"},
				{TId: "893", Value: "One Piece Adventure of Spiral Island", Season: "1"},
				{TId: "893", Value: "One Piece: Clockwork Island Adventure", Season: "1"},
				{TId: "1253", Value: "One Piece Chinjuujima no Chopper Oukoku", Season: "1"},
				{TId: "1253", Value: "One Piece Chopper`s Kingdom in the Strange Animal Island", Season: "1"},
				{TId: "1254", Value: "One Piece The Movie Dead End no Bouken", Season: "1"},
				{TId: "1254", Value: "One Piece The Movie The Dead End Adventure", Season: "1"},
				{TId: "2036", Value: "One Piece Norowareta Seiken", Season: "1"},
				{TId: "2036", Value: "One Piece The Curse of the Sacred Sword", Season: "1"},
				{TId: "2644", Value: "One Piece Omatsuri Danshaku to Himitsu no Shima", Season: "1"},
				{TId: "2644", Value: "One Piece Baron Omatsuri and the Island of Secrets", Season: "1"},
				{TId: "2736", Value: "One Piece Taose Kaizoku Ganzack", Season: "1"},
				{TId: "2736", Value: "One Piece Defeat the Pirate Ganzack ", Season: "1"},
				{TId: "4097", Value: "One Piece The Movie Karakurijou no Mecha Kyohei", Season: "1"},
				{TId: "4097", Value: "One Piece The Giant Mechanical Soldier of Karakuri Castle", Season: "1"},
				{TId: "4851", Value: "One Piece Episode of Arabasta Sabaku no Oujo to Kaizoku tachi", Season: "1"},
				{TId: "4851", Value: "One Piece The Desert Princess and the Pirates Adventures in Alabasta", Season: "1"},
				{TId: "5691", Value: "One Piece The Movie Episode of Chopper Plus Fuyu ni Saku Kiseki no Sakura", Season: "1"},
				{TId: "5691", Value: "One Piece Episode of Chopper Plus Bloom in the Winter Miracle Cherry Blossom", Season: "1"},
				{TId: "6199", Value: "One Piece Special Romance Dawn Story", Season: "1"},
				{TId: "6199", Value: "One Piece Romance Dawn Story", Season: "1"},
				{TId: "6537", Value: "One Piece Film Strong World", Season: "1"},
				{TId: "7538", Value: "One Piece Film Strong World Episode 0", Season: "1"},
				{TId: "8010", Value: "One Piece 3D Mugiwara Chase", Season: "1"},
				{TId: "8010", Value: "One Piece 3D Straw Hat Chase", Season: "1"},
				{TId: "8762", Value: "One Piece 3D Gekisou Trap Coaster", Season: "1"},
				{TId: "8940", Value: "One Piece Film Z", Season: "1"},
				{TId: "11529", Value: "One Piece Film Gold", Season: "1"},
				{TId: "14318", Value: "Gekijouban One Piece Stampede", Season: "1"},
				{TId: "14318", Value: "One Piece Stampede", Season: "1"},
				{TId: "16983", Value: "One Piece Film Red", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("[Anime Time] One Piece (0001-1071+Movies+Specials) [BD+CR] [Dual Audio] [1080p][HEVC 10bit x265][AAC][Multi Sub]"),
					[]torrentMap{
						{
							anidbId:      "69",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 1,
							episodeEnd:   1071,
						},
						{
							anidbId:      "69",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   1071,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   8,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       2,
							episodeStart: 1,
							episodeEnd:   22,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       3,
							episodeStart: 1,
							episodeEnd:   17,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       4,
							episodeStart: 1,
							episodeEnd:   13,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       5,
							episodeStart: 1,
							episodeEnd:   9,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       6,
							episodeStart: 1,
							episodeEnd:   22,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       7,
							episodeStart: 1,
							episodeEnd:   39,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       8,
							episodeStart: 1,
							episodeEnd:   13,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       9,
							episodeStart: 1,
							episodeEnd:   52,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       10,
							episodeStart: 1,
							episodeEnd:   31,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       11,
							episodeStart: 1,
							episodeEnd:   99,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       12,
							episodeStart: 1,
							episodeEnd:   56,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       13,
							episodeStart: 1,
							episodeEnd:   100,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       14,
							episodeStart: 1,
							episodeEnd:   35,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       15,
							episodeStart: 1,
							episodeEnd:   62,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       16,
							episodeStart: 1,
							episodeEnd:   50,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       17,
							episodeStart: 1,
							episodeEnd:   118,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       18,
							episodeStart: 1,
							episodeEnd:   33,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       19,
							episodeStart: 1,
							episodeEnd:   98,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       20,
							episodeStart: 1,
							episodeEnd:   14,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       21,
							episodeStart: 1,
							episodeEnd:   180,
						},
					},
				},
				{
					makeTorrentInfo("[Erai-raws] One Piece - 601~700 [1080p][Multiple Subtitle]"),
					[]torrentMap{
						{
							anidbId:      "69",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 601,
							episodeEnd:   700,
						},
						{
							anidbId:      "69",
							seasonType:   "ani",
							season:       1,
							episodeStart: 601,
							episodeEnd:   700,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       16,
							episodeStart: 23,
							episodeEnd:   50,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       17,
							episodeStart: 1,
							episodeEnd:   72,
						},
					},
				},
				{
					makeTorrentInfo("[df68] One Piece Movie 2 - Clockwork Island Adventure [BD][1080p][x264][JPN][SUB]"),
					[]torrentMap{
						{
							anidbId:    "893",
							seasonType: "ani",
							season:     1,
							episodes:   []int{1},
						},
						{
							anidbId:    "893",
							seasonType: "tv",
							season:     0,
							episodes:   []int{4},
						},
					},
				},
				{
					makeTorrentInfo("One Piece - 1080-1090"),
					[]torrentMap{
						{
							anidbId:      "69",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 1080,
							episodeEnd:   1090,
						},
						{
							anidbId:      "69",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1080,
							episodeEnd:   1090,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       21,
							episodeStart: 189,
							episodeEnd:   194,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       22,
							episodeStart: 1,
							episodeEnd:   5,
						},
					},
				},
				{
					makeTorrentInfo("One.Piece.S21.JPN.Sub.ITA.1080p.WEB-DL.x264-UBi"),
					[]torrentMap{
						{
							anidbId:      "69",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 892,
							episodeEnd:   1085,
						},
						{
							anidbId:      "69",
							seasonType:   "ani",
							season:       1,
							episodeStart: 892,
							episodeEnd:   1085,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       21,
							episodeStart: 1,
							episodeEnd:   194,
						},
					},
				},
				{
					makeTorrentInfo("[CameEsp] One Piece - 1085 [1080p][ESP-ENG][mkv].mkv"),
					[]torrentMap{
						{
							anidbId:      "69",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 1085,
							episodeEnd:   1085,
						},
						{
							anidbId:      "69",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1085,
							episodeEnd:   1085,
						},
						{
							anidbId:      "69",
							seasonType:   "tv",
							season:       21,
							episodeStart: 194,
							episodeEnd:   194,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="17870" tvdbid="431162" defaulttvdbseason="1">
		  <name>Kusuriya no Hitorigoto</name>
		</anime>

		<anime anidbid="18562" tvdbid="431162" defaulttvdbseason="2">
		  <name>Kusuriya no Hitorigoto (2025)</name>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "17870", Value: "Kusuriya no Hitorigoto", Season: "1", Year: "2023"},
				{TId: "17870", Value: "The Apothecary Diaries", Season: "1", Year: "2023"},
				{TId: "18562", Value: "Kusuriya no Hitorigoto", Season: "2", Year: "2025"},
				{TId: "18562", Value: "The Apothecary Diaries", Season: "2", Year: "2025"},
			},
			[]testCase{
				{
					makeTorrentInfo("[LostYears] The Apothecary Diaries - S01E20 (WEB 1080p x264 E-AC-3 AAC) [8C2EE5A9].mkv"),
					[]torrentMap{
						{
							anidbId:      "17870",
							seasonType:   "ani",
							season:       1,
							episodeStart: 20,
							episodeEnd:   20,
						},
						{
							anidbId:      "17870",
							seasonType:   "tv",
							season:       1,
							episodeStart: 20,
							episodeEnd:   20,
						},
					},
				},
				{
					makeTorrentInfo("The Apothecary Diaries S01 (E01-E12) 1080p BluRay Remux AVC FLAC 2.0-CRUCiBLE [Dual Audio] | Kusuriya no Hitorigoto"),
					[]torrentMap{
						{
							anidbId:      "17870",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   12,
						},
						{
							anidbId:      "17870",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   12,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="266" tvdbid="72454" defaulttvdbseason="a">
    <name>Meitantei Conan</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="0">;1-0;2-0;3-0;4-73;5-75;</mapping>
      <mapping anidbseason="1" tvdbseason="2" start="29" end="54" offset="-28"/>
      <mapping anidbseason="1" tvdbseason="3" start="55" end="82" offset="-54"/>
      <mapping anidbseason="1" tvdbseason="4" start="83" end="106" offset="-82"/>
      <mapping anidbseason="1" tvdbseason="5" start="107" end="134" offset="-106"/>
      <mapping anidbseason="1" tvdbseason="6" start="135" end="162" offset="-134"/>
      <mapping anidbseason="1" tvdbseason="7" start="163" end="193" offset="-162"/>
      <mapping anidbseason="1" tvdbseason="8" start="194" end="219" offset="-193"/>
      <mapping anidbseason="1" tvdbseason="9" start="220" end="254" offset="-219"/>
      <mapping anidbseason="1" tvdbseason="10" start="255" end="285" offset="-254"/>
      <mapping anidbseason="1" tvdbseason="11" start="286" end="315" offset="-285"/>
      <mapping anidbseason="1" tvdbseason="12" start="316" end="353" offset="-315"/>
      <mapping anidbseason="1" tvdbseason="13" start="354" end="389" offset="-353"/>
      <mapping anidbseason="1" tvdbseason="14" start="390" end="426" offset="-389"/>
      <mapping anidbseason="1" tvdbseason="15" start="427" end="465" offset="-426"/>
      <mapping anidbseason="1" tvdbseason="16" start="466" end="490" offset="-465"/>
      <mapping anidbseason="1" tvdbseason="17" start="491" end="523" offset="-490"/>
      <mapping anidbseason="1" tvdbseason="18" start="524" end="565" offset="-523"/>
      <mapping anidbseason="1" tvdbseason="19" start="566" end="605" offset="-565"/>
      <mapping anidbseason="1" tvdbseason="20" start="606" end="645" offset="-605"/>
      <mapping anidbseason="1" tvdbseason="21" start="646" end="680" offset="-645"/>
      <mapping anidbseason="1" tvdbseason="22" start="681" end="723" offset="-680"/>
      <mapping anidbseason="1" tvdbseason="23" start="724" end="762" offset="-723"/>
      <mapping anidbseason="1" tvdbseason="24" start="763" end="803" offset="-762"/>
      <mapping anidbseason="1" tvdbseason="25" start="804" end="886" offset="-803"/>
      <mapping anidbseason="1" tvdbseason="26" start="887" end="926" offset="-886"/>
      <mapping anidbseason="1" tvdbseason="27" start="927" end="964" offset="-926"/>
      <mapping anidbseason="1" tvdbseason="28" start="965" end="992" offset="-964"/>
      <mapping anidbseason="1" tvdbseason="29" start="993" end="1032" offset="-992"/>
      <mapping anidbseason="1" tvdbseason="30" start="1033" end="1067" offset="-1032"/>
      <mapping anidbseason="1" tvdbseason="31" start="1068" end="1108" offset="-1067"/>
      <mapping anidbseason="1" tvdbseason="32" start="1109" offset="-1108"/>
    </mapping-list>
  </anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "266", Value: "Meitantei Conan", Season: "1"},
				{TId: "266", Value: "Detective Conan", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("Detective Conan S01"),
					[]torrentMap{
						{
							anidbId:      "266",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 1,
							episodeEnd:   28,
						},
						{
							anidbId:      "266",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   28,
						},
						{
							anidbId:      "266",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   28,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="449" tvdbid="79060" defaulttvdbseason="1">
    <name>Wolf's Rain</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="1" start="1" end="4" offset="26"/>
    </mapping-list>
    <before>;1-27;2-28;3-29;4-30;</before>
  </anime>
			`),
			[]anidb.AniDBTitle{},
			[]testCase{
				{
					makeTorrentInfo("Wolf's.Rain.S01.2003.1080p.BluRay.X265.10bit.JAP.TrueHD.HunSub-PluSUltra"),
					[]torrentMap{
						{
							anidbId:    "449",
							seasonType: "ani",
							season:     1,
						},
						{
							anidbId:    "449",
							seasonType: "tv",
							season:     1,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="2369" tvdbid="74796" defaulttvdbseason="a">
    <name>Bleach</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="0">;2-99;3-2;4-0;</mapping>
      <mapping anidbseason="1" tvdbseason="1" start="1" end="20"/>
      <mapping anidbseason="1" tvdbseason="2" start="21" end="41" offset="-20"/>
      <mapping anidbseason="1" tvdbseason="3" start="42" end="63" offset="-41"/>
      <mapping anidbseason="1" tvdbseason="4" start="64" end="91" offset="-63"/>
      <mapping anidbseason="1" tvdbseason="5" start="92" end="109" offset="-91"/>
      <mapping anidbseason="1" tvdbseason="6" start="110" end="131" offset="-109"/>
      <mapping anidbseason="1" tvdbseason="7" start="132" end="151" offset="-131"/>
      <mapping anidbseason="1" tvdbseason="8" start="152" end="167" offset="-151"/>
      <mapping anidbseason="1" tvdbseason="9" start="168" end="189" offset="-167"/>
      <mapping anidbseason="1" tvdbseason="10" start="190" end="205" offset="-189"/>
      <mapping anidbseason="1" tvdbseason="11" start="206" end="212" offset="-205"/>
      <mapping anidbseason="1" tvdbseason="12" start="213" end="229" offset="-212"/>
      <mapping anidbseason="1" tvdbseason="13" start="230" end="265" offset="-229"/>
      <mapping anidbseason="1" tvdbseason="14" start="266" end="316" offset="-265"/>
      <mapping anidbseason="1" tvdbseason="15" start="317" end="342" offset="-316"/>
      <mapping anidbseason="1" tvdbseason="16" start="343" end="366" offset="-342"/>
    </mapping-list>
    <supplemental-info>
      <studio>Studio Pierrot</studio>
    </supplemental-info>
  </anime>

  <anime anidbid="15449" tvdbid="74796" defaulttvdbseason="a" episodeoffset="366">
    <name>Bleach: Sennen Kessen Hen</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="17" start="1" end="13" offset="0"/>
    </mapping-list>
  </anime>

  <anime anidbid="17765" tvdbid="74796" defaulttvdbseason="a" episodeoffset="379">
    <name>Bleach: Sennen Kessen Hen - Ketsubetsu Tan</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="17" start="1" end="13" offset="13"/>
    </mapping-list>
  </anime>

  <anime anidbid="18220" tvdbid="74796" defaulttvdbseason="a" episodeoffset="392">
    <name>Bleach: Sennen Kessen Hen - Soukoku Tan</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="17" start="1" end="14" offset="26"/>
    </mapping-list>
  </anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "2369", Value: "Bleach", Season: "1"},
				{TId: "15449", Value: "Bleach", Season: "2"},
				{TId: "17765", Value: "Bleach", Season: "3"},
				{TId: "18220", Value: "Bleach", Season: "4"},
			},
			[]testCase{
				{
					makeTorrentInfo("Bleach.S17E02.MULTi.1080p.WEB.H264-FW"),
					[]torrentMap{
						{
							anidbId:      "15449",
							seasonType:   "abs",
							season:       -1,
							episodeStart: 368,
							episodeEnd:   368,
						},
						{
							anidbId:      "15449",
							seasonType:   "ani",
							season:       2,
							episodeStart: 2,
							episodeEnd:   2,
						},
						{
							anidbId:      "15449",
							seasonType:   "tv",
							season:       17,
							episodeStart: 2,
							episodeEnd:   2,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="6327" tvdbid="102261" defaulttvdbseason="1">
    <name>Bakemonogatari</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="0">;4-1;</mapping>
      <mapping anidbseason="0" tvdbseason="1">;1-13;2-14;3-15;</mapping>
    </mapping-list>
    <before>;1-13;2-14;3-15;</before>
  </anime>
			`),
			[]anidb.AniDBTitle{},
			[]testCase{
				{
					makeTorrentInfo("MONOGATARI.Series.OFF.and.MONSTER.Season.S01E10.SHINOBUMONOGATARI.Shinobu.Mustard.Part.Two.1080p.CR.WEB-DL.AAC2.0.H.264-VARYG.mkv"),
					[]torrentMap{
						{
							anidbId:      "6327",
							seasonType:   "ani",
							season:       1,
							episodeStart: 10,
							episodeEnd:   10,
						},
						{
							anidbId:      "6327",
							seasonType:   "tv",
							season:       1,
							episodeStart: 10,
							episodeEnd:   10,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="2400" tvdbid="78916" defaulttvdbseason="a">
		  <name>Gantz</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="1" start="1" end="13"/>
		    <mapping anidbseason="1" tvdbseason="2" start="14" end="26" offset="-13"/>
		  </mapping-list>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "2400", Value: "Gantz", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("Gantz.2010.1080p.BluRay.x264-[YTS.AM].mp4"),
					[]torrentMap{},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="6327" tvdbid="102261" defaulttvdbseason="1">
		  <name>Bakemonogatari</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;4-1;</mapping>
		    <mapping anidbseason="0" tvdbseason="1">;1-13;2-14;3-15;</mapping>
		  </mapping-list>
		  <before>;1-13;2-14;3-15;</before>
		</anime>

		<anime anidbid="8357" tvdbid="102261" defaulttvdbseason="0" imdbid="tt3138698,tt5084196,tt5084198">
		  <name>Kizumonogatari</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-2;2-19;3-20;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="8658" tvdbid="102261" defaulttvdbseason="2">
		  <name>Nisemonogatari</name>
		</anime>

		<anime anidbid="9183" tvdbid="102261" defaulttvdbseason="3">
		  <name>Monogatari Series: Second Season</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;6-7;11-8;16-9;</mapping>
		    <mapping anidbseason="1" tvdbseason="3">;1-1;2-2;3-3;4-4;5-5;7-6;8-7;9-8;10-9;12-10;13-11;14-12;15-13;17-14;18-15;19-16;20-17;21-18;22-19;23-20;24-21;25-22;26-23;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="9453" tvdbid="102261" defaulttvdbseason="0">
		  <name>Nekomonogatari (Kuro): Tsubasa Family</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-0;2-3;3-4;4-5;5-6;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="10046" tvdbid="102261" defaulttvdbseason="0">
		  <name>Hanamonogatari: Suruga Devil</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-0;2-10;3-11;4-12;5-13;6-14;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="10891" tvdbid="102261" defaulttvdbseason="0">
		  <name>Tsukimonogatari: Yotsugi Doll</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-0;2-15;3-16;4-17;5-18;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="11350" tvdbid="102261" defaulttvdbseason="4">
		  <name>Owarimonogatari</name>
		</anime>

		<anime anidbid="11827" tvdbid="102261" defaulttvdbseason="0" episodeoffset="20">
		  <name>Koyomimonogatari</name>
		</anime>

		<anime anidbid="13033" tvdbid="102261" defaulttvdbseason="5">
		  <name>Owarimonogatari (2017)</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="5">;1-0;2-0;402-1;403-2;405-3;406-4;407-5;408-6;409-7;</mapping>
		    <mapping anidbseason="1" tvdbseason="5">;1-0;2-0;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="13691" tvdbid="102261" defaulttvdbseason="0">
		  <name>Zoku Owarimonogatari</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-33;2-33;3-34;4-35;5-36;6-37;7-38;</mapping>
		  </mapping-list>
		</anime>

		<anime anidbid="18424" tvdbid="102261" defaulttvdbseason="6">
		  <name>Monogatari Series: Off &amp; Monster Season</name>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "6327", Value: "Bakemonogatari", Season: "1"},
				{TId: "8357", Value: "Kizumonogatari", Season: "1"},
				{TId: "8658", Value: "Nisemonogatari", Season: "2"},
				{TId: "9183", Value: "Monogatari Series Second Season", Season: "4"},
				{TId: "9453", Value: "Nekomonogatari Black Tsubasa Family", Season: "3"},
				{TId: "10046", Value: "Hanamonogatari Suruga Devil", Season: "5"},
				{TId: "10891", Value: "Tsukimonogatari Yotsugi Doll", Season: "6"},
				{TId: "11350", Value: "Owarimonogatari", Season: "7"},
				{TId: "11827", Value: "Koyomimonogatari", Season: "1"},
				{TId: "13033", Value: "Owarimonogatari", Season: "8", Year: "2017"},
				{TId: "13691", Value: "Zoku Owarimonogatari", Season: "9"},
				{TId: "18424", Value: "Monogatari Series Off Monster Season", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("[HorribleSubs] Owarimonogatari - 12 [720p].mkv"),
					[]torrentMap{
						{
							anidbId:      "11350",
							seasonType:   "ani",
							season:       7,
							episodeStart: 12,
							episodeEnd:   12,
						},
						{
							anidbId:      "11350",
							seasonType:   "tv",
							season:       4,
							episodeStart: 12,
							episodeEnd:   12,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="14990" tvdbid="376144" defaulttvdbseason="1">
		  <name>Great Pretender</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="2" start="15" end="23" offset="-14"/>
		  </mapping-list>
		</anime>

		<anime anidbid="18280" tvdbid="376144" defaulttvdbseason="0" tmdbid="1220441" imdbid="tt30827040">
		  <name>Great Pretender: Razbliuto</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-1;2-1;3-1;4-1;</mapping>
		  </mapping-list>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "14990", Value: "Great Pretender", Season: "1"},
				{TId: "18280", Value: "Great Pretender Razbliuto", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("[Erai-raws] Great Pretender - 01 ~ 15 [1080p][Multiple Subtitle]"),
					[]torrentMap{
						{
							anidbId:      "14990",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   15,
						},
						{
							anidbId:      "14990",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   14,
						},
						{
							anidbId:      "14990",
							seasonType:   "tv",
							season:       2,
							episodeStart: 15,
							episodeEnd:   15,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
		<anime anidbid="6257" tvdbid="87501" defaulttvdbseason="1">
		  <name>K-On!</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;3-1;4-2;5-3;6-4;7-5;8-6;9-7;</mapping>
		    <mapping anidbseason="0" tvdbseason="1">;1-13;2-14;</mapping>
		  </mapping-list>
		  <before>;1-13;2-14;</before>
		</anime>

		<anime anidbid="7307" tvdbid="87501" defaulttvdbseason="2">
		  <name>K-On!!</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;4-8;5-9;6-10;7-11;8-12;9-13;10-14;11-15;12-16;3-17;</mapping>
		    <mapping anidbseason="0" tvdbseason="2">;1-25;2-26;</mapping>
		  </mapping-list>
		  <before>;1-22;2-23;</before>
		</anime>

		<anime anidbid="8280" tvdbid="87501" defaulttvdbseason="0" imdbid="tt1909796">
		  <name>Eiga K-On!</name>
		  <mapping-list>
		    <mapping anidbseason="1" tvdbseason="0">;1-18;</mapping>
		  </mapping-list>
		</anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "6257", Value: "K On", Season: "1"},
				{TId: "7307", Value: "K On", Season: "2"},
				{TId: "8280", Value: "Eiga K On", Season: "1"},
				{TId: "8280", Value: "K On The Movie", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("[Anime Time] K-On! (Season 01+02+Movie+OVA+Specials) [BD][Dual Audio][1080p][HEVC 10bit x265][AAC][Eng Sub]"),
					[]torrentMap{
						{
							anidbId:    "6257",
							seasonType: "ani",
							season:     1,
						},
						{
							anidbId:    "6257",
							seasonType: "tv",
							season:     1,
						},
						{
							anidbId:    "7307",
							seasonType: "ani",
							season:     2,
						},
						{
							anidbId:    "7307",
							seasonType: "tv",
							season:     2,
						},
					},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="3303" tvdbid="76906" defaulttvdbseason="a">
    <name>Medarot</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="1" start="1" end="26" offset="0"/>
      <mapping anidbseason="1" tvdbseason="2" start="27" end="52" offset="-26"/>
    </mapping-list>
  </anime>

  <anime anidbid="4694" tvdbid="76906" defaulttvdbseason="a" episodeoffset="52">
    <name>Medarot Damashii</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="3" start="1" end="39" offset="0"/>
    </mapping-list>
  </anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "3303", Value: "Medarot", Season: "1"},
				{TId: "4694", Value: "Medarot Damashii", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("Medarot (1999) S01e01-52 [480p H264 Ita Jap SubITA] REPACK byMC-08"),
					[]torrentMap{},
				},
			},
		},
		{
			toEpisodeMaps(`
  <anime anidbid="16947" tvdbid="405920" defaulttvdbseason="1" tmdbtv="120089" tmdbseason="1">
    <name>Spy x Family</name>
  </anime>

  <anime anidbid="17061" tvdbid="405920" defaulttvdbseason="1" episodeoffset="12" tmdbtv="120089" tmdbseason="1" tmdboffset="12">
    <name>Spy x Family (2022)</name>
  </anime>

  <anime anidbid="17784" tvdbid="405920" defaulttvdbseason="2" tmdbtv="120089" tmdbseason="2">
    <name>Spy x Family (2023)</name>
  </anime>

  <anime anidbid="17785" tvdbid="405920" defaulttvdbseason="0" episodeoffset="2" tmdbid="1062807" imdbid="tt26684398">
    <name>Gekijouban Spy x Family Code: White</name>
  </anime>
			`),
			[]anidb.AniDBTitle{
				{TId: "16947", Value: "Spy x Family", Season: "1"},
				{TId: "17061", Value: "Spy x Family", Season: "1"},
				{TId: "17784", Value: "Spy x Family", Season: "2"},
				{TId: "17785", Value: "Gekijouban Spy x Family Code White", Season: "1"},
			},
			[]testCase{
				{
					makeTorrentInfo("[LostYears] SPY x FAMILY - S01E04 (WEB 1080p HEVC AAC E-AC-3) [CDD0F677].mkv"),
					[]torrentMap{
						{
							anidbId:      "16947",
							seasonType:   "ani",
							season:       1,
							episodeStart: 4,
							episodeEnd:   4,
						},
						{
							anidbId:      "16947",
							seasonType:   "tv",
							season:       1,
							episodeStart: 4,
							episodeEnd:   4,
						},
					},
				},
				{
					makeTorrentInfo("[ToonsHub] SPY x FAMILY (Multi Dub) - S01E16 (WEB 1080p x264 AAC).mkv"),
					[]torrentMap{
						{
							anidbId:      "17061",
							seasonType:   "ani",
							season:       1,
							episodeStart: 4,
							episodeEnd:   4,
						},
						{
							anidbId:      "17061",
							seasonType:   "tv",
							season:       1,
							episodeStart: 16,
							episodeEnd:   16,
						},
					},
				},
				{
					makeTorrentInfo("Spy x Family - 01-25 END (2160p) (Bilibili)"),
					[]torrentMap{
						{
							anidbId:      "16947",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   12,
						},
						{
							anidbId:      "16947",
							seasonType:   "tv",
							season:       1,
							episodeStart: 1,
							episodeEnd:   12,
						},
						{
							anidbId:      "17061",
							seasonType:   "ani",
							season:       1,
							episodeStart: 1,
							episodeEnd:   13,
						},
					},
				},
			},
		},
	} {
		for _, c := range tc.cases {
			// if c.tInfo.TorrentTitle != "[LostYears] SPY x FAMILY - S01E04 (WEB 1080p HEVC AAC E-AC-3) [CDD0F677].mkv" {
			// 	continue
			// }
			tMaps, err := prepareAniDBTorrentMaps(tc.tvdbMaps, tc.titles, c.tInfo)
			assert.NoError(t, err)
			assert.Len(t, tMaps, len(c.result), tc.tvdbMaps[0].TVDBId+" - "+c.tInfo.TorrentTitle)
			for i := range c.result {
				r := c.result[i]
				assert.Equal(t, r, tMaps[i], tc.tvdbMaps[0].TVDBId+":"+r.anidbId+":"+string(r.seasonType)+":"+strconv.Itoa(r.season))
			}
		}
	}
}
