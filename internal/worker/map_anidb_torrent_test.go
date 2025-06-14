package worker

import (
	"encoding/xml"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/animelists"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/stretchr/testify/assert"
)

func TestMatchAniDBIdsInTVDBEpisodeMaps(t *testing.T) {
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
		result []string
	}

	for _, tc := range []struct {
		tvdbMaps anidb.AniDBTVDBEpisodeMaps
		titles   []anidb.AniDBTitle
		cases    []testCase
	}{
		// {
		// 	toEpisodeMaps(`
		// <anime anidbid="11123" tvdbid="293088" defaulttvdbseason="1">
		//   <name>One Punch Man</name>
		//   <mapping-list>
		//     <mapping anidbseason="0" tvdbseason="0">;1-2;2-3;3-4;4-5;5-6;6-7;</mapping>
		//   </mapping-list>
		// </anime>
		//
		// <anime anidbid="11637" tvdbid="293088" defaulttvdbseason="0">
		//   <name>One Punch Man: Road to Hero</name>
		// </anime>
		//
		// <anime anidbid="12430" tvdbid="293088" defaulttvdbseason="2">
		//   <name>One Punch Man (2019)</name>
		//   <mapping-list>
		//     <mapping anidbseason="0" tvdbseason="0">;1-8;2-9;3-10;4-11;5-12;6-13;7-14</mapping>
		//   </mapping-list>
		//   <before>;1-1;2-3;</before>
		// </anime>
		//
		// <anime anidbid="17576" tvdbid="293088" defaulttvdbseason="3">
		//   <name>One Punch Man (2025)</name>
		// </anime>
		// 	`),
		// 	[]anidb.AniDBTitle{
		// 		{TId: "11123", Value: "One Punch Man", Season: "1"},
		// 		{TId: "11637", Value: "One Punch Man OVA", Season: "1"},
		// 		{TId: "11637", Value: "One Punch Man: Road to Hero", Season: "1"},
		// 		{TId: "12430", Value: "One Punch Man", Season: "2", Year: "2019"},
		// 		{TId: "12430", Value: "One Punch Man (2019)", Season: "2", Year: "2019"},
		// 		{TId: "17576", Value: "One Punch Man", Season: "3", Year: "2025"},
		// 		{TId: "17576", Value: "One Punch Man (2025)", Season: "3", Year: "2025"},
		// 	},
		// 	[]testCase{
		// 		{
		// 			makeTorrentInfo("[LostYears] One Punch Man - S02E07 (WEB 1080p x264 10-bit AAC) [5EA9AF2F].mkv"),
		// 			[]string{"12430"},
		// 		},
		// 		{
		// 			makeTorrentInfo("[GbR] One Punch Man - 02 [2160p H.265].mkv"),
		// 			[]string{"11123"},
		// 		},
		// 		{
		// 			makeTorrentInfo("[AnimeRG] One Punch Man (2019) (Season 2 Complete) (00-12) [1080p] [Eng Subbed] [JRR]"),
		// 			[]string{"12430"},
		// 		},
		// 		{
		// 			makeTorrentInfo("[Anime Time] One Punch Man [S1+S2+OVA&ODA][Dual Audio][1080p BD][HEVC 10bit x265][AAC][Eng Subs]"),
		// 			[]string{"11123", "12430"},
		// 		},
		// 		{
		// 			makeTorrentInfo("[sam] One Punch Man OVA [BD 1080p FLAC]"),
		// 			[]string{"11637"},
		// 		},
		// 		{
		// 			makeTorrentInfo("[Blaze077] One Punch Man - OVA-  Road To Hero [720p].mkv"),
		// 			[]string{"11637"},
		// 		},
		// 	},
		// },
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
			[]anidb.AniDBTitle{},
			[]testCase{
				{
					makeTorrentInfo("[F-D] Fairy Tail Season 1 -6 + Extras [480P][Dual-Audio]"),
					[]string{"6662", "9980"},
				},
				{
					makeTorrentInfo("[AnimeRG] Fairy Tail Final Series (2018) (278-328 Complete) [1080p] [JRR] (S3 01-51)"),
					[]string{"13295"},
				},
			},
		},
	} {
		for _, c := range tc.cases {
			anidbIds, err := matchAniDBIdsInTVDBEpisodeMaps(tc.tvdbMaps, tc.titles, c.tInfo)
			assert.NoError(t, err)
			assert.Equal(t, c.result, anidbIds)
		}
	}
}
