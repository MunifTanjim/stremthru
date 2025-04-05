package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeasons(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
	}{
		{"regular season", "The Simpsons S28E21 720p HDTV x264-AVS", []int{28}},
		{"regular season with lowercase", "breaking.bad.s01e01.720p.bluray.x264-reward", []int{1}},
		{"regular season with high number", "48 Hours S51E15 720p WEB x264-CookieMonster", []int{51}},
		{"regular season with high number v2", "Bargain.Hunt.S66E02.Builth Wells4.720p.WEB.H264-BeechyBoy.mp4", []int{66}},
		{"regular season with O instead of zero", "Arrested Development SO2E04.avi", []int{2}},
		{"regular season with 3 digits", "S011E16.mkv", []int{11}},
		{"regular season with a space between", "Dragon Ball Super S01 E23 French 1080p HDTV H264-Kesni", []int{1}},
		{"regular season with a letter a suffix", "The Twilight Zone 1985 S01E23a Shadow Play.mp4", []int{1}},
		{"regular season with a letter b suffix", "Mash S10E01b Thats Show Biz Part 2 1080p H.264 (moviesbyrizzo upload).mp4", []int{10}},
		{"regular season with a letter c suffix", "The Twilight Zone 1985 S01E22c The Library.mp4", []int{1}},
		{"regular episode without e separator", "Desperate.Housewives.S0615.400p.WEB-DL.Rus.Eng.avi", []int{6}},
		{"season with SxEE format correctly", "Doctor.Who.2005.8x11.Dark.Water.720p.HDTV.x264-FoV", []int{8}},
		{"season when written as such", "Orange Is The New Black Season 5 Episodes 1-10 INCOMPLETE (LEAKED)", []int{5}},
		{"season with parenthesis prefix and x separator", "Smallville (1x02 Metamorphosis).avi", []int{1}},
		{"season with x separator and letter on left", "The.Man.In.The.High.Castle1x01.HDTV.XviD[www.DivxTotaL.com].avi", []int{1}},
		{"season with x separator and letter on right", "clny.3x11m720p.es[www.planetatorrent.com].mkv", []int{3}},
		{"multiple seasons separated with comma", "Game Of Thrones Complete Season 1,2,3,4,5,6,7 406p mkv + Subs", []int{1, 2, 3, 4, 5, 6, 7}},
		{"multiple seasons separated with space with redundant digit suffix", "Futurama Season 1 2 3 4 5 6 7 + 4 Movies - threesixtyp", []int{1, 2, 3, 4, 5, 6, 7}},
		{"multiple season separated with spaces and comma", "Breaking Bad Complete Season 1 , 2 , 3, 4 ,5 ,1080p HEVC", []int{1, 2, 3, 4, 5}},
		{"multiple season separated with comma, space and symbol at the end", "True Blood Season 1, 2, 3, 4, 5 & 6 + Extras BDRip TSV", []int{1, 2, 3, 4, 5, 6}},
		{"How I Met Your Mother Season 1, 2, 3, 4, 5, & 6 + Extras DVDRip", "How I Met Your Mother Season 1, 2, 3, 4, 5, & 6 + Extras DVDRip", []int{1, 2, 3, 4, 5, 6}},
		{"multiple seasons separated with space", "The Simpsons Season 20 21 22 23 24 25 26 27 - threesixtyp", []int{20, 21, 22, 23, 24, 25, 26, 27}},
		{"multiple seasons separated with space an spanish season name", "Perdidos: Lost: Castellano: Temporadas 1 2 3 4 5 6 (Serie Com", []int{1, 2, 3, 4, 5, 6}},
		{"multiple seasons with with unequal separators", "The Boondocks Season 1, 2 & 3", []int{1, 2, 3}},
		{"multiple seasons with with space and plus symbol", "Boondocks, The - Seasons 1 + 2", []int{1, 2}},
		{"multiple seasons with implied range without s prefix", "The Boondocks Seasons 1-4 MKV", []int{1, 2, 3, 4}},
		{"multiple seasons separated with space plus and symbol", "The Expanse Complete Seasons 01 & 02 1080p", []int{1, 2}},
		{"multiple seasons with s prefix and implied range", "Friends.Complete.Series.S01-S10.720p.BluRay.2CH.x265.HEVC-PSA", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"multiple seasons with s prefix separated with slash", "Stargate Atlantis ALL Seasons - S01 / S02 / S03 / S04 / S05", []int{1, 2, 3, 4, 5}},
		{"multiple seasons with season and parenthesis prefix", "Stargate Atlantis Complete (Season 1 2 3 4 5) 720p HEVC x265", []int{1, 2, 3, 4, 5}},
		{"multiple seasons with s prefix separated with hyphen", "Skam.S01-S02-S03.SweSub.720p.WEB-DL.H264", []int{1, 2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
		title   string
	}{
		{"detect correct title with multiple season definitions", "Seinfeld S02 Season 2 720p WebRip ReEnc-DeeJayAhmed", []int{2}, "Seinfeld"},
		{"correct title with multiple season definitions", "Seinfeld Season 2 S02 720p AMZN WEBRip x265 HEVC Complete", []int{2}, "Seinfeld"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
			assert.Equal(t, tc.title, result.Title)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
	}{
		{"multiple season when given implied range inside parenthesis without s prefix", "House MD All Seasons (1-8) 720p Ultra-Compressed", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"multiple season when given implied range with season prefix", "Teen Titans Season 1-5", []int{1, 2, 3, 4, 5}},
		{"multiple season when given implied range y words with season prefix", "Game Of Thrones - Season 1 to 6 (Eng Subs)", []int{1, 2, 3, 4, 5, 6}},
		{"multiple season with and separator", "Travelers - Seasons 1 and 2 - Mp4 x264 AC3 1080p", []int{1, 2}},
		{"multiple season when given implied range separated with colon", "Naruto Shippuden Season 1:11", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{"multiple season when given implied range separated with colon and space", "South Park Complete Seasons 1: 11", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{"multiple season when title is with numbers", "24 Season 1-8 Complete with Subtitles", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"single season when contains non related number range", "One Punch Man 01 - 12 Season 1 Complete [720p] [Eng Subs] [Xerxe:16", []int{1}},
		{"season withing brackets with dot separator", "[5.01] Weight Loss.avi", []int{5}},
		{"season withing brackets with dot separator and episode in hunderds", "Dragon Ball [5.134] Preliminary Peril.mp4", []int{5}},
		{"season with s prefix and single digit", "Bron - S4 - 720P - SweSub.mp4", []int{4}},
		{"Mad Men S02 Season 2 720p 5.1Ch BluRay ReEnc-DeeJayAhmed", "Mad Men S02 Season 2 720p 5.1Ch BluRay ReEnc-DeeJayAhmed", []int{2}},
		{"Friends S04 Season 4 1080p 5.1Ch BluRay ReEnc-DeeJayAhmed", "Friends S04 Season 4 1080p 5.1Ch BluRay ReEnc-DeeJayAhmed", []int{4}},
		{"multiple season with double seperators", "Doctor Who S01--S07--Complete with holiday episodes", []int{1, 2, 3, 4, 5, 6, 7}},
		{"season with a dot and hyphen separator", "My Little Pony FiM - 6.01 - No Second Prances.mkv", []int{6}},
		{"season with a dot and epsiode prefix", "Desperate Housewives - Episode 1.22 - Goodbye for now.avi", []int{1}},
		{"season with a dot and episode prefix v2", "All of Us Are Dead . 2022 . S01 EP #1.2.mkv", []int{1}},
		{"season with a year range afterwards", "Empty Nest Season 1 (1988 - 89) fiveofseven", []int{1}},
		{"multiple seasons with russian season and hyphen separator", "Game of Thrones / Сезон: 1-8 / Серии: 1-73 из 73 [2011-2019, США, BDRip 1080p] MVO (LostFilm)", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"multiple seasons with russian season and comma separator", "Друзья / Friends / Сезон: 1, 2 / Серии: 1-24 из 24 [1994-1999, США, BDRip 720p] MVO", []int{1, 2}},
		{"season with russian season word", "Друзья / Friends / Сезон: 1 / Серии: 1-24 из 24 [1994-1995, США, BDRip 720p] MVO + Original + Sub (Rus, Eng)", []int{1}},
		{"season with russian season word as folder name", "Сезон 5/Серия 11.mkv", []int{5}},
		{"season with russian season word followed by the number", "Разрушители легенд. MythBusters. Сезон 15. Эпизод 09. Скрытая угроза (2015).avi", []int{15}},
		{"season with russian season word followed by the number v2", "Леди Баг и Супер-Кот – Сезон 3, Эпизод 21 – Кукловод 2 [1080p].mkv", []int{3}},
		{"episode with full russian season name with case suffix", "Проклятие острова ОУК_ 5-й сезон 09-я серия_ Прорыв Дэна.avi", []int{5}},
		{"season with russian season word with number at front", "2 сезон 24 серия.avi", []int{2}},
		{"season with russian season word with number at front and nothing else", "3 сезон", []int{3}},
		{"season with russian season word and underscore", "2. Discovery-Kak_ustroena_Vselennaya.(2.sezon_8.serii.iz.8).2012.XviD.HDTVRip.Krasnodarka", []int{2}},
		{"season with russian season shortened word", "Otchayannie.domochozyaiki.(8.sez.21.ser.iz.23).2012.XviD.HDTVRip.avi", []int{8}},
		{"season with russian season word and no prefix", "Интерны. Сезон №9. Серия №180.avi", []int{9}},
		{"season with russian x separator", "Discovery. Парни с Юкона / Yokon Men [06х01-08] (2017) HDTVRip от GeneralFilm | P1", []int{6}},
		{"season with russian season word in araic letters", "Zvezdnie.Voiny.Voina.Klonov.3.sezon.22.seria.iz.22.XviD.HDRip.avi", []int{3}},
		{"season with hyphen separator between episode", "2-06. Девичья сила.mkv", []int{2}},
		{"season with hyphen separator between episode v2", "4-13 Cursed (HD).m4v", []int{4}},
		{"season with hyphen separator between episode v3", "Доктор Хаус 03-20.mkv", []int{3}},
		{"episodes with hyphen separator between episode v4", "Комиссар Рекс 11-13.avi", []int{11}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
		title   string
	}{
		{"not season with hyphen separator when it's the title", "13-13-13 2013 DVDrip x264 AAC-MiLLENiUM", nil, "13-13-13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
			assert.Equal(t, tc.title, result.Title)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
	}{
		{"correct season with eps prefix and hyphen separator", "MARATHON EPISODES/Orphan Black S3 Eps.05-08.mp4", []int{3}},
		{"multiple seasons with end season without s symbol", "Once Upon a Time [S01-07] (2011-2017) WEB-DLRip by Generalfilm", []int{1, 2, 3, 4, 5, 6, 7}},
		{"multiple seasons with one space and hyphen separator", "[F-D] Fairy Tail Season 1 -6 + Extras [480P][Dual-Audio]", []int{1, 2, 3, 4, 5, 6}},
		{"multiple seasons with spaces and hyphen separator", "Coupling Season 1 - 4 Complete DVDRip - x264 - MKV by RiddlerA", []int{1, 2, 3, 4}},
		{"single season with spaces and hyphen separator", "[HR] Boku no Hero Academia 87 (S4-24) [1080p HEVC Multi-Subs] HR-GZ", []int{4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		ttitle  string
		seasons []int
	}{
		{"Tokyo Ghoul Root A - 07 [S2-07] [Eng Sub] 480p [email protected]", []int{2}},
		{"Ace of the Diamond: 1st Season", []int{1}},
		{"Ace of the Diamond: 2nd Season", []int{2}},
		{"Adventure Time 10 th season", []int{10}},
		{"Kyoukai no Rinne (TV) 3rd Season - 23 [1080p]", []int{3}},
		{"[Erai-raws] Granblue Fantasy The Animation Season 2 - 08 [1080p][Multiple Subtitle].mkv", []int{2}},
		{"The Nile Egypts Great River with Bettany Hughes Series 1 4of4 10", []int{1}},
		{"Teen Wolf - 04ª Temporada 720p", []int{4}},
		{"Vikings 3 Temporada 720p", []int{3}},
		{"Eu, a Patroa e as Crianças  4° Temporada Completa - HDTV - Dublado", []int{4}},
		{"Merl - Temporada 1", []int{1}},
		{"Elementar 3º Temporada Dublado", []int{3}},
		{"Beavis and Butt-Head - 1a. Temporada", []int{1}},
		{"3Âº Temporada Bob esponja Pt-Br", []int{3}},
		{"Juego de Tronos - Temp.2 [ALTA DEFINICION 720p][Cap.209][Spanish].mkv", []int{2}},
		{"Los Simpsons Temp 7 DVDrip Espanol De Espana", []int{7}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
	}{
		{"spanish season range with & separator", "The Walking Dead [Temporadas 1 & 2 Completas Em HDTV E Legena", []int{1, 2}},
		{"spanish short season identifier", "My Little Pony - A Amizade é Mágica - T02E22.mp4", []int{2}},
		{"spanish short season identifier with xe separator", "30 M0N3D4S ESP T01XE08.mkv", []int{1}},
		{"sn naming scheme", "Sons of Anarchy Sn4 Ep14 HD-TV - To Be, Act 2, By Cool Release", []int{4}},
		{"single season and not range in filename", "[FFA] Kiratto Pri☆chan Season 3 - 11 [1080p][HEVC].mkv", []int{3}},
		{"single season and not range in filename v2", "[Erai-raws] Granblue Fantasy The Animation Season 2 - 10 [1080p][Multiple Subtitle].mkv", []int{2}},
		{"single season and not range in filename v3", "[SCY] Attack on Titan Season 3 - 11 (BD 1080p Hi10 FLAC) [1FA13150].mkv", []int{3}},
		{"single zero season", "DARKER THAN BLACK - S00E04 - Darker Than Black Gaiden OVA 3.mkv", []int{0}},
		{"nl season word", "Seizoen 22 - Zon & Maan Ultra Legendes/afl.18 Je ogen op de bal houden!.mp4", []int{22}},
		{"italian season word", "Nobody Wants This - Stagione 1 (2024) [COMPLETA] 720p H264 ITA AAC 2.0-Zer0landia", []int{1}},
		{"italian season range", "Red Oaks - Stagioni 01-03 (2014-2017) [COMPLETA] SD x264 AAC ITA SUB ITA - mkeagle3", []int{1, 2, 3}},
		{"polish season", "Star.Wars.Skeleton.Crew.Sezon01.PLDUB.480p.DSNP.WEB-DL.H264.DDP5.1-K83", []int{1}},
		{"polish season with S prefix", "Bitten.SezonSO3.PL.480p.NF.WEB-DL.DD5.1.XviD-Ralf", []int{3}},
		{"regular season with year range before", "'Lucky.Luke.1983-1992.S01E04.PL.720p.WEB-DL.H264-zyl.mkv'", []int{1}},
		{"polish season range", "Rizzoli & Isles 2010-2016 [Sezon 01-07] [1080p.WEB-DL.H265.EAC3-FT][Alusia]", []int{1, 2, 3, 4, 5, 6, 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		name     string
		ttitle   string
		seasons  []int
		episodes []int
	}{
		{"season episode when not in boundary", "Those.About.to.DieS01E06.MULTi.720p.AMZN.WEB-DL.H264.DDP5.1-K83.mkv", []int{1}, []int{6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
			assert.Equal(t, tc.episodes, result.Episodes)
		})
	}

	for _, tc := range []struct {
		name    string
		ttitle  string
		seasons []int
	}{
		{"not season when it's part of the name", "Ranma-12-86.mp4", nil},
		{"not season when it's part of group", "The Killer's Game 2024 PL 1080p WEB-DL H264 DD5.1-S56", nil},
		{"not season when it's part of group v2", "Apollo 13 (1995) [1080p] [WEB-DL] [x264] [E-AC3-S78] [Lektor PL]", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle  string
		seasons []int
	}{
		{"2 сезон 24 серия.avi", []int{2}},
		{"2-06. Девичья сила.mkv", []int{2}},
		{"2. Discovery-Kak_ustroena_Vselennaya.(2.sezon_8.serii.iz.8).2012.XviD.HDTVRip.Krasnodarka", []int{2}},
		{"3 сезон", []int{3}},
		{"3Âº Temporada Bob esponja Pt-Br", []int{3}},
		{"4-13 Cursed (HD).m4v", []int{4}},
		{"13-13-13 2013 DVDrip x264 AAC-MiLLENiUM", nil},
		{"24 Season 1-8 Complete with Subtitles", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"30 M0N3D4S ESP T01XE08.mkv", []int{1}},
		{"Ace of the Diamond: 1st Season", []int{1}},
		{"Ace of the Diamond: 2nd Season", []int{2}},
		{"Adventure Time 10 th season", []int{10}},
		{"All of Us Are Dead . 2022 . S01 EP #1.2.mkv", []int{1}},
		{"Beavis and Butt-Head - 1a. Temporada", []int{1}},
		{"Boondocks, The - Seasons 1 + 2", []int{1, 2}},
		{"breaking.bad.s01e01.720p.bluray.x264-reward", []int{1}},
		{"Breaking Bad Complete Season 1 , 2 , 3, 4 ,5 ,1080p HEVC", []int{1, 2, 3, 4, 5}},
		{"Bron - S4 - 720P - SweSub.mp4", []int{4}},
		{"clny.3x11m720p.es[www.planetatorrent.com].mkv", []int{3}},
		{"Coupling Season 1 - 4 Complete DVDRip - x264 - MKV by RiddlerA", []int{1, 2, 3, 4}},
		{"DARKER THAN BLACK - S00E04 - Darker Than Black Gaiden OVA 3.mkv", []int{0}},
		{"Desperate.Housewives.S0615.400p.WEB-DL.Rus.Eng.avi", []int{6}},
		{"Desperate Housewives - Episode 1.22 - Goodbye for now.avi", []int{1}},
		{"Discovery. Парни с Юкона / Yokon Men [06х01-08] (2017) HDTVRip от GeneralFilm | P1", []int{6}},
		{"Doctor.Who.2005.8x11.Dark.Water.720p.HDTV.x264-FoV", []int{8}},
		{"Doctor Who S01--S07--Complete with holiday episodes", []int{1, 2, 3, 4, 5, 6, 7}},
		{"Dragon Ball Super S01 E23 French 1080p HDTV H264-Kesni", []int{1}},
		{"Dragon Ball [5.134] Preliminary Peril.mp4", []int{5}},
		{"Elementar 3º Temporada Dublado", []int{3}},
		{"Empty Nest Season 1 (1988 - 89) fiveofseven", []int{1}},
		{"Eu, a Patroa e as Crianças  4° Temporada Completa - HDTV - Dublado", []int{4}},
		{"Friends.Complete.Series.S01-S10.720p.BluRay.2CH.x265.HEVC-PSA", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"Friends S04 Season 4 1080p 5.1Ch BluRay ReEnc-DeeJayAhmed", []int{4}},
		{"Futurama Season 1 2 3 4 5 6 7 + 4 Movies - threesixtyp", []int{1, 2, 3, 4, 5, 6, 7}},
		{"Game Of Thrones - Season 1 to 6 (Eng Subs)", []int{1, 2, 3, 4, 5, 6}},
		{"Game Of Thrones Complete Season 1,2,3,4,5,6,7 406p mkv + Subs", []int{1, 2, 3, 4, 5, 6, 7}},
		{"Game of Thrones / Сезон: 1-8 / Серии: 1-73 из 73 [2011-2019, США, BDRip 1080p] MVO (LostFilm)", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"House MD All Seasons (1-8) 720p Ultra-Compressed", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"How I Met Your Mother Season 1, 2, 3, 4, 5, & 6 + Extras DVDRip", []int{1, 2, 3, 4, 5, 6}},
		{"Juego de Tronos - Temp.2 [ALTA DEFINICION 720p][Cap.209][Spanish].mkv", []int{2}},
		{"Kyoukai no Rinne (TV) 3rd Season - 23 [1080p]", []int{3}},
		{"Los Simpsons Temp 7 DVDrip Espanol De Espana", []int{7}},
		{"Mad Men S02 Season 2 720p 5.1Ch BluRay ReEnc-DeeJayAhmed", []int{2}},
		{"MARATHON EPISODES/Orphan Black S3 Eps.05-08.mp4", []int{3}},
		{"Mash S10E01b Thats Show Biz Part 2 1080p H.264 (moviesbyrizzo upload).mp4", []int{10}},
		{"Merl - Temporada 1", []int{1}},
		{"My Little Pony - A Amizade é Mágica - T02E22.mp4", []int{2}},
		{"My Little Pony FiM - 6.01 - No Second Prances.mkv", []int{6}},
		{"Naruto Shippuden Season 1:11", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{"Once Upon a Time [S01-07] (2011-2017) WEB-DLRip by Generalfilm", []int{1, 2, 3, 4, 5, 6, 7}},
		{"One Punch Man 01 - 12 Season 1 Complete [720p] [Eng Subs] [Xerxe:16", []int{1}},
		{"Orange Is The New Black Season 5 Episodes 1-10 INCOMPLETE (LEAKED)", []int{5}},
		{"Otchayannie.domochozyaiki.(8.sez.21.ser.iz.23).2012.XviD.HDTVRip.avi", []int{8}},
		{"Perdidos: Lost: Castellano: Temporadas 1 2 3 4 5 6 (Serie Com", []int{1, 2, 3, 4, 5, 6}},
		{"Ranma-12-86.mp4", nil},
		{"S011E16.mkv", []int{11}},
		{"Seinfeld S02 Season 2 720p WebRip ReEnc-DeeJayAhmed", []int{2}},
		{"Seinfeld Season 2 S02 720p AMZN WEBRip x265 HEVC Complete", []int{2}},
		{"Seizoen 22 - Zon & Maan Ultra Legendes/afl.18 Je ogen op de bal houden!.mp4", []int{22}},
		{"Skam.S01-S02-S03.SweSub.720p.WEB-DL.H264", []int{1, 2, 3}},
		{"Smallville (1x02 Metamorphosis).avi", []int{1}},
		{"Sons of Anarchy Sn4 Ep14 HD-TV - To Be, Act 2, By Cool Release", []int{4}},
		{"South Park Complete Seasons 1: 11", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}},
		{"Stargate Atlantis ALL Seasons - S01 / S02 / S03 / S04 / S05", []int{1, 2, 3, 4, 5}},
		{"Stargate Atlantis Complete (Season 1 2 3 4 5) 720p HEVC x265", []int{1, 2, 3, 4, 5}},
		{"Teen Titans Season 1-5", []int{1, 2, 3, 4, 5}},
		{"Teen Wolf - 04ª Temporada 720p", []int{4}},
		{"The.Man.In.The.High.Castle1x01.HDTV.XviD[www.DivxTotaL.com].avi", []int{1}},
		{"The Boondocks Season 1, 2 & 3", []int{1, 2, 3}},
		{"The Boondocks Seasons 1-4 MKV", []int{1, 2, 3, 4}},
		{"The Expanse Complete Seasons 01 & 02 1080p", []int{1, 2}},
		{"The Nile Egypts Great River with Bettany Hughes Series 1 4of4 10", []int{1}},
		{"The Simpsons S28E21 720p HDTV x264-AVS", []int{28}},
		{"The Simpsons Season 20 21 22 23 24 25 26 27 - threesixtyp", []int{20, 21, 22, 23, 24, 25, 26, 27}},
		{"The Twilight Zone 1985 S01E22c The Library.mp4", []int{1}},
		{"The Twilight Zone 1985 S01E23a Shadow Play.mp4", []int{1}},
		{"The Walking Dead [Temporadas 1 & 2 Completas Em HDTV E Legena", []int{1, 2}},
		{"Tokyo Ghoul Root A - 07 [S2-07] [Eng Sub] 480p [email protected]", []int{2}},
		{"Travelers - Seasons 1 and 2 - Mp4 x264 AC3 1080p", []int{1, 2}},
		{"True Blood Season 1, 2, 3, 4, 5 & 6 + Extras BDRip TSV", []int{1, 2, 3, 4, 5, 6}},
		{"Vikings 3 Temporada 720p", []int{3}},
		{"Zvezdnie.Voiny.Voina.Klonov.3.sezon.22.seria.iz.22.XviD.HDRip.avi", []int{3}},
		{"[5.01] Weight Loss.avi", []int{5}},
		{"[Erai-raws] Granblue Fantasy The Animation Season 2 - 08 [1080p][Multiple Subtitle].mkv", []int{2}},
		{"[Erai-raws] Granblue Fantasy The Animation Season 2 - 10 [1080p][Multiple Subtitle].mkv", []int{2}},
		{"[Erai-raws] Shingeki no Kyojin Season 3 - 11 (BD 1080p Hi10 FLAC) [1FA13150].mkv", []int{3}},
		{"[F-D] Fairy Tail Season 1 -6 + Extras [480P][Dual-Audio]", []int{1, 2, 3, 4, 5, 6}},
		{"[FFA] Kiratto Pri☆chan Season 3 - 11 [1080p][HEVC].mkv", []int{3}},
		{"[HR] Boku no Hero Academia 87 (S4-24) [1080p HEVC Multi-Subs] HR-GZ", []int{4}},
		{"[SCY] Attack on Titan Season 3 - 11 (BD 1080p Hi10 FLAC) [1FA13150].mkv", []int{3}},
		{"Доктор Хаус 03-20.mkv", []int{3}},
		{"Друзья / Friends / Сезон: 1 / Серии: 1-24 из 24 [1994-1995, США, BDRip 720p] MVO + Original + Sub (Rus, Eng)", []int{1}},
		{"Друзья / Friends / Сезон: 1, 2 / Серии: 1-24 из 24 [1994-1999, США, BDRip 720p] MVO", []int{1, 2}},
		{"Интерны. Сезон №9. Серия №180.avi", []int{9}},
		{"Комиссар Рекс 11-13.avi", []int{11}},
		{"Леди Баг и Супер-Кот – Сезон 3, Эпизод 21 – Кукловод 2 [1080p].mkv", []int{3}},
		{"Проклятие острова ОУК_ 5-й сезон 09-я серия_ Прорыв Дэна.avi", []int{5}},
		{"Разрушители легенд. MythBusters. Сезон 15. Эпизод 09. Скрытая угроза (2015).avi", []int{15}},
		{"Сезон 5/Серия 11.mkv", []int{5}},
		{"Vikkatakavi 01E06.mkv", []int{1}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}

	for _, tc := range []struct {
		ttitle  string
		seasons []int
	}{
		{"Archer.S02.1080p.BluRay.DTSMA.AVC.Remux", []int{2}},
		{"The Simpsons S01E01 1080p BluRay x265 HEVC 10bit AAC 5.1 Tigole", []int{1}},
		{"[F-D] Fairy Tail Season 1 - 6 + Extras [480P][Dual-Audio]", []int{1, 2, 3, 4, 5, 6}},
		{"House MD All Seasons (1-8) 720p Ultra-Compressed", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"Bleach 10º Temporada - 215 ao 220 - [DB-BR]", []int{10}},
		{"Lost.[Perdidos].6x05.HDTV.XviD.[www.DivxTotaL.com]", []int{6}},
		{"4-13 Cursed (HD)", []int{4}},
		{"Dragon Ball Z Movie - 09 - Bojack Unbound - 1080p BluRay x264 DTS 5.1 -DDR", nil},
		{"BoJack Horseman [06x01-08 of 16] (2019-2020) WEB-DLRip 720p", []int{6}},
		{"[HR] Boku no Hero Academia 87 (S4-24) [1080p HEVC Multi-Subs] HR-GZ", []int{4}},
		{"The Simpsons S28E21 720p HDTV x264-AVS", []int{28}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.seasons, result.Seasons)
		})
	}
}
