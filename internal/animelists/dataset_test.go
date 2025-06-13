package animelists

import (
	"encoding/xml"
	"strconv"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/stretchr/testify/assert"
)

func TestProcessAnimeListItemsForTVDBId(t *testing.T) {
	toAnimeListItems := func(xmlContent string) []AnimeListItem {
		parsed := struct {
			Items []AnimeListItem `xml:"anime"`
		}{}
		err := xml.Unmarshal([]byte("<anime-list>"+xmlContent+"</anime-list>"), &parsed)
		if err != nil {
			panic(err)
		}
		return parsed.Items
	}

	for _, tc := range []struct {
		tvdbId string
		items  []AnimeListItem
		result []anidb.AniDBTVDBEpisodeMap
	}{
		{
			"83692",
			toAnimeListItems(`
  <anime anidbid="19" tvdbid="83692" defaulttvdbseason="a">
    <name>Rizelmine</name>
    <mapping-list>
      <mapping anidbseason="1" tvdbseason="1" start="1" end="12"/>
      <mapping anidbseason="1" tvdbseason="2" start="13" end="24" offset="-12"/>
    </mapping-list>
  </anime>
			`),
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "19",
					TVDBId:      "83692",
					AniDBSeason: 1,
					TVDBSeason:  -1,
					Offset:      0,
					Start:       0,
					End:         0,
				},
				{
					AniDBId:     "19",
					TVDBId:      "83692",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Offset:      0,
					Start:       1,
					End:         12,
				},
				{
					AniDBId:     "19",
					TVDBId:      "83692",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Offset:      -12,
					Start:       13,
					End:         24,
				},
			},
		},
		{
			"81472",
			toAnimeListItems(`
		<anime anidbid="1530" tvdbid="81472" defaulttvdbseason="a">
		  <name>Dragon Ball Z</name>
		  <mapping-list>
		    <mapping anidbseason="0" tvdbseason="0">;1-0;2-0;3-0;4-0;5-0;</mapping>
		  </mapping-list>
		</anime>
			`),
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "1530",
					TVDBId:      "81472",
					AniDBSeason: 1,
					TVDBSeason:  -1,
				},
				{
					AniDBId:     "1530",
					TVDBId:      "81472",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{0},
						2: []int{0},
						3: []int{0},
						4: []int{0},
						5: []int{0},
					},
				},
			},
		},
		{
			"114801",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "6662",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  -1,
				},
				{
					AniDBId:     "6662",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Offset:      0,
					Start:       1,
					End:         48,
				},
				{
					AniDBId:     "6662",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Offset:      -48,
					Start:       49,
					End:         96,
				},
				{
					AniDBId:     "6662",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  3,
					Offset:      -96,
					Start:       97,
					End:         150,
				},
				{
					AniDBId:     "6662",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  4,
					Offset:      -150,
					Start:       151,
					End:         175,
				},
				{
					AniDBId:     "8132",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						4: []int{5},
						5: []int{7},
						6: []int{8},
						7: []int{9},
						8: []int{10},
						9: []int{11},
					},
				},
				{
					AniDBId:     "8788",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      3,
				},
				{
					AniDBId:     "8788",
					TVDBId:      "114801",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{6},
					},
				},
				{
					AniDBId:     "9980",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  -1,
					Offset:      175,
				},
				{
					AniDBId:     "9980",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  5,
					Offset:      0,
					Start:       1,
					End:         51,
				},
				{
					AniDBId:     "9980",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  6,
					Offset:      -51,
					Start:       52,
					End:         90,
				},
				{
					AniDBId:     "9980",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  7,
					Offset:      -90,
					Start:       91,
					End:         102,
				},
				{
					AniDBId:     "11247",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      11,
				},
				{
					AniDBId:     "13295",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  -1,
					Offset:      277,
				},
				{
					AniDBId:     "13295",
					TVDBId:      "114801",
					AniDBSeason: 1,
					TVDBSeason:  8,
					Offset:      0,
					Start:       1,
					End:         51,
				},
			},
		},
		{
			"293088",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "11123",
					TVDBId:      "293088",
					AniDBSeason: 1,
					TVDBSeason:  1,
				},
				{
					AniDBId:     "11123",
					TVDBId:      "293088",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{2},
						2: []int{3},
						3: []int{4},
						4: []int{5},
						5: []int{6},
						6: []int{7},
					},
				},
				{
					AniDBId:     "11637",
					TVDBId:      "293088",
					AniDBSeason: 1,
					TVDBSeason:  0,
				},
				{
					AniDBId:     "12430",
					TVDBId:      "293088",
					AniDBSeason: 1,
					TVDBSeason:  2,
				},
				{
					AniDBId:     "12430",
					TVDBId:      "293088",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{8},
						2: []int{9},
						3: []int{10},
						4: []int{11},
						5: []int{12},
						6: []int{13},
						7: []int{14},
					},
				},
				{
					AniDBId:     "17576",
					TVDBId:      "293088",
					AniDBSeason: 1,
					TVDBSeason:  3,
				},
			},
		},
	} {
		t.Run(tc.tvdbId, func(t *testing.T) {
			result := processAnimeListItemsForTVDBId(tc.tvdbId, tc.items)
			assert.Len(t, result, len(tc.result))
			for i := range tc.result {
				r := tc.result[i]
				assert.Equal(t, r, result[i], strconv.Itoa(i)+"-"+r.AniDBId+":"+r.TVDBId+":"+strconv.Itoa(r.AniDBSeason)+":"+strconv.Itoa(r.TVDBSeason))
			}
		})
	}
}
