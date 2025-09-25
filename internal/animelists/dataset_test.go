package animelists

import (
	"encoding/xml"
	"strconv"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/stretchr/testify/assert"
)

func TestPrepareAniDBTVDBEpisodeMaps(t *testing.T) {
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
					Start:       1,
					End:         24,
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
					Start:       1,
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
					Offset:      0,
					Start:       1,
					End:         175,
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
					Start:       1,
					End:         102,
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
					Start:       1,
					End:         51,
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
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 1,
						2: 3,
					},
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
		{
			"81797",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  -1,
					Offset:      0,
					Start:       1,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1:  []int{27},
						2:  []int{3},
						3:  []int{9},
						4:  []int{10},
						5:  []int{14},
						6:  []int{0},
						7:  []int{0},
						8:  []int{0},
						9:  []int{0},
						10: []int{0},
						11: []int{0},
						12: []int{0},
						13: []int{0},
						14: []int{0},
						15: []int{0},
						16: []int{31},
						17: []int{23},
						18: []int{24},
						19: []int{28},
						20: []int{29},
						21: []int{30},
						22: []int{0},
						23: []int{32},
						24: []int{34},
						25: []int{0},
						26: []int{36},
						27: []int{37},
						28: []int{40},
						29: []int{41},
						30: []int{42},
						31: []int{43},
						32: []int{44},
						33: []int{45},
						34: []int{47},
						35: []int{48},
						36: []int{49},
						37: []int{50},
						38: []int{51},
						39: []int{52},
						40: []int{53},
						41: []int{54},
						42: []int{55},
						43: []int{56},
						44: []int{57},
					},
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Offset:      0,
					Start:       1,
					End:         8,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Offset:      -8,
					Start:       9,
					End:         30,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  3,
					Offset:      -30,
					Start:       31,
					End:         47,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  4,
					Offset:      -47,
					Start:       48,
					End:         60,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  5,
					Offset:      -60,
					Start:       61,
					End:         69,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  6,
					Offset:      -69,
					Start:       70,
					End:         91,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  7,
					Offset:      -91,
					Start:       92,
					End:         130,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  8,
					Offset:      -130,
					Start:       131,
					End:         143,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  9,
					Offset:      -143,
					Start:       144,
					End:         195,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  10,
					Offset:      -195,
					Start:       196,
					End:         226,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  11,
					Offset:      -226,
					Start:       227,
					End:         325,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  12,
					Offset:      -325,
					Start:       326,
					End:         381,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  13,
					Offset:      -381,
					Start:       382,
					End:         481,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  14,
					Offset:      -481,
					Start:       482,
					End:         516,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  15,
					Offset:      -516,
					Start:       517,
					End:         578,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  16,
					Offset:      -578,
					Start:       579,
					End:         628,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  17,
					Offset:      -628,
					Start:       629,
					End:         746,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  18,
					Offset:      -746,
					Start:       747,
					End:         779,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  19,
					Offset:      -779,
					Start:       780,
					End:         877,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  20,
					Offset:      -877,
					Start:       878,
					End:         891,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  21,
					Offset:      -891,
					Start:       892,
					End:         1085,
				},
				{
					AniDBId:     "69",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  22,
					Offset:      -1085,
					Start:       1086,
				},
				{
					AniDBId:     "411",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      1,
				},
				{
					AniDBId:     "893",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{4},
					},
				},
				{
					AniDBId:     "893",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{5},
					},
				},
				{
					AniDBId:     "1253",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{6},
					},
				},
				{
					AniDBId:     "1253",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{7},
					},
				},
				{
					AniDBId:     "1254",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{8},
						2: []int{8},
						3: []int{8},
					},
				},
				{
					AniDBId:     "2036",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{11},
						2: []int{11},
						3: []int{11},
					},
				},
				{
					AniDBId:     "2036",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{12},
					},
				},
				{
					AniDBId:     "2644",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      12,
				},
				{
					AniDBId:     "2736",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      0,
				},
				{
					AniDBId:     "4097",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      14,
				},
				{
					AniDBId:     "4851",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      15,
				},
				{
					AniDBId:     "5691",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      16,
				},
				{
					AniDBId:     "6199",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      17,
				},
				{
					AniDBId:     "6537",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      18,
				},
				{
					AniDBId:     "7538",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      19,
				},
				{
					AniDBId:     "8010",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      20,
				},
				{
					AniDBId:     "8762",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      21,
				},
				{
					AniDBId:     "8940",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{26},
					},
				},
				{
					AniDBId:     "8940",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{25},
						2: []int{25},
						3: []int{25},
					},
				},
				{
					AniDBId:     "11529",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{33},
					},
				},
				{
					AniDBId:     "11529",
					TVDBId:      "81797",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{35},
					},
				},
				{
					AniDBId:     "14318",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      37,
				},
				{
					AniDBId:     "16983",
					TVDBId:      "81797",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      45,
				},
			},
		},
		{
			"431162",
			toAnimeListItems(`
  <anime anidbid="17870" tvdbid="431162" defaulttvdbseason="1">
    <name>Kusuriya no Hitorigoto</name>
  </anime>

  <anime anidbid="18562" tvdbid="431162" defaulttvdbseason="2">
    <name>Kusuriya no Hitorigoto (2025)</name>
  </anime>
			`),
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "17870",
					TVDBId:      "431162",
					AniDBSeason: 1,
					TVDBSeason:  1,
				},
				{
					AniDBId:     "18562",
					TVDBId:      "431162",
					AniDBSeason: 1,
					TVDBSeason:  2,
				},
			},
		},
		{
			"87501",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 13,
						2: 14,
					},
				},
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						3: []int{1},
						4: []int{2},
						5: []int{3},
						6: []int{4},
						7: []int{5},
						8: []int{6},
						9: []int{7},
					},
				},
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  1,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{13},
						2: []int{14},
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 22,
						2: 23,
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						3:  []int{17},
						4:  []int{8},
						5:  []int{9},
						6:  []int{10},
						7:  []int{11},
						8:  []int{12},
						9:  []int{13},
						10: []int{14},
						11: []int{15},
						12: []int{16},
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  2,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{25},
						2: []int{26},
					},
				},
				{
					AniDBId:     "8280",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{18},
					},
				},
			},
		},
		{
			"79060",
			toAnimeListItems(`
  <anime anidbid="449" tvdbid="79060" defaulttvdbseason="1">
    <name>Wolf's Rain</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="1" start="1" end="4" offset="26"/>
    </mapping-list>
    <before>;1-27;2-28;3-29;4-30;</before>
  </anime>
			`),
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "449",
					TVDBId:      "79060",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 27,
						2: 28,
						3: 29,
						4: 30,
					},
				},
				{
					AniDBId:     "449",
					TVDBId:      "79060",
					AniDBSeason: 0,
					TVDBSeason:  1,
					Start:       1,
					End:         4,
					Offset:      26,
				},
			},
		},
		{
			"376144",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "14990",
					TVDBId:      "376144",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Start:       1,
					End:         14,
				},
				{
					AniDBId:     "14990",
					TVDBId:      "376144",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Start:       15,
					End:         23,
					Offset:      -14,
				},
				{
					AniDBId:     "18280",
					TVDBId:      "376144",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{1},
						2: []int{1},
						3: []int{1},
						4: []int{1},
					},
				},
			},
		},
		{
			"87501",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 13,
						2: 14,
					},
				},
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						3: []int{1},
						4: []int{2},
						5: []int{3},
						6: []int{4},
						7: []int{5},
						8: []int{6},
						9: []int{7},
					},
				},
				{
					AniDBId:     "6257",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  1,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{13},
						2: []int{14},
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  2,
					Before: anidb.AniDBTVDBEpisodeMapBefore{
						1: 22,
						2: 23,
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						4:  []int{8},
						5:  []int{9},
						6:  []int{10},
						7:  []int{11},
						8:  []int{12},
						9:  []int{13},
						10: []int{14},
						11: []int{15},
						12: []int{16},
						3:  []int{17},
					},
				},
				{
					AniDBId:     "7307",
					TVDBId:      "87501",
					AniDBSeason: 0,
					TVDBSeason:  2,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{25},
						2: []int{26},
					},
				},
				{
					AniDBId:     "8280",
					TVDBId:      "87501",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Map: anidb.AniDBTVDBEpisodeMapMap{
						1: []int{18},
					},
				},
			},
		},
		{
			"405920",
			toAnimeListItems(`
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
			[]anidb.AniDBTVDBEpisodeMap{
				{
					AniDBId:     "16947",
					TVDBId:      "405920",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Start:       1,
					End:         12,
				},
				{
					AniDBId:     "17061",
					TVDBId:      "405920",
					AniDBSeason: 1,
					TVDBSeason:  1,
					Offset:      12,
				},
				{
					AniDBId:     "17784",
					TVDBId:      "405920",
					AniDBSeason: 1,
					TVDBSeason:  2,
				},
				{
					AniDBId:     "17785",
					TVDBId:      "405920",
					AniDBSeason: 1,
					TVDBSeason:  0,
					Offset:      2,
				},
			},
		},
	} {
		t.Run(tc.tvdbId, func(t *testing.T) {
			if tc.tvdbId != "405920" {
				return
			}
			result := PrepareAniDBTVDBEpisodeMaps(tc.tvdbId, tc.items)
			assert.Len(t, result, len(tc.result))
			for i := range tc.result {
				r := tc.result[i]
				assert.Equal(t, r, result[i], strconv.Itoa(i)+"-"+r.AniDBId+":"+r.TVDBId+":"+strconv.Itoa(r.AniDBSeason)+":"+strconv.Itoa(r.TVDBSeason))
			}
		})
	}
}
