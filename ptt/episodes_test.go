package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEpisodes(t *testing.T) {
	intRange := func(start, end int) []int {
		nums := make([]int, end-start+1)
		for i := range end - start + 1 {
			nums[i] = start + i
		}
		return nums
	}

	type tcEpisodes struct {
		name     string
		ttitle   string
		episodes []int
	}

	testAssertEpisodes := func(tcs []tcEpisodes) {
		for _, tc := range tcs {
			name := tc.name
			if name == "" {
				name = tc.ttitle
			}
			t.Run(name, func(t *testing.T) {
				result := Parse(tc.ttitle)
				assert.Equal(t, tc.episodes, result.Episodes)
			})
		}
	}

	type tcSeasonsAndEpisodes struct {
		name     string
		ttitle   string
		seasons  []int
		episodes []int
	}

	testAssertSeasonsAndEpisodes := func(tcs []tcSeasonsAndEpisodes) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				result := Parse(tc.ttitle)
				assert.Equal(t, tc.seasons, result.Seasons)
				assert.Equal(t, tc.episodes, result.Episodes)
			})
		}
	}

	testAssertEpisodes([]tcEpisodes{
		{"regular episode", "The Simpsons S28E21 720p HDTV x264-AVS", []int{21}},
		{"regular episode with lowercase", "breaking.bad.s01e01.720p.bluray.x264-reward", []int{1}},
		{"regular season with O instead of zero", "Arrested Development SO2E04.avi", []int{4}},
		{"regular episode with a space between", "Dragon Ball Super S01 E23 French 1080p HDTV H264-Kesni", []int{23}},
		{"regular episode above 1000", "One.Piece.S01E1116.Lets.Go.Get.It!.Buggys.Big.Declaration.2160p.B-Global.WEB-DL.JPN.AAC2.0.H.264.MSubs-ToonsHub.mkv", []int{1116}},
		{"regular episode without e symbol after season", "The.Witcher.S01.07.2019.Dub.AVC.ExKinoRay.mkv", []int{7}},
		{"regular episode with season symbol but wihout episode symbol", "Vikings.s02.09.AVC.tahiy.mkv", []int{9}},
		{"regular episode with a letter a suffix", "The Twilight Zone 1985 S01E23a Shadow Play.mp4", []int{23}},
		{"regular episode without break at the end", "Desperate_housewives_S03E02Le malheur aime la compagnie.mkv", []int{2}},
		{"regular episode with a letter b suffix", "Mash S10E01b Thats Show Biz Part 2 1080p H.264 (moviesbyrizzo upload).mp4", []int{1}},
		{"regular episode with a letter c suffix", "The Twilight Zone 1985 S01E22c The Library.mp4", []int{22}},
		{"regular episode without e separator", "Desperate.Housewives.S0615.400p.WEB-DL.Rus.Eng.avi", []int{15}},
		{"episode with SxEE format correctly", "Doctor.Who.2005.8x11.Dark.Water.720p.HDTV.x264-FoV", []int{11}},
		{"episode when written as such", "Anubis saison 01 episode 38 tvrip FR", []int{38}},
		{"episode when written as such shortened", "Le Monde Incroyable de Gumball - Saison 5 Ep 14 - L'extérieur", []int{14}},
		{"episode with parenthesis prefix and x separator", "Smallville (1x02 Metamorphosis).avi", []int{2}},
		{"episode with x separator and letter on left", "The.Man.In.The.High.Castle1x01.HDTV.XviD[www.DivxTotaL.com].avi", []int{1}},
		{"episode with x separator and letter on right", "clny.3x11m720p.es[www.planetatorrent.com].mkv", []int{11}},
	})

	testAssertSeasonsAndEpisodes([]tcSeasonsAndEpisodes{
		{"episode when similar digits included", "Friends.S07E20.The.One.With.Rachel's.Big.Kiss.720p.BluRay.2CH.x265.HEVC-PSA.mkv", []int{7}, []int{20}},
		{"episode when separated with x and inside brackets", "Friends - [8x18] - The One In Massapequa.mkv", []int{8}, []int{18}},
		{"episode when separated with X", "Archivo 81 1X7 HDTV XviD Castellano.avi", []int{1}, []int{7}},
		{"multiple episodes with x prefix and hyphen separator", "Friends - [7x23-24] - The One with Monica and Chandler's Wedding + Audio Commentary.mkv", []int{7}, []int{23, 24}},
		{"episode when separated with x and has three digit episode", "Yu-Gi-Oh 3x089 - Awakening of Evil (Part 4).avi", []int{3}, []int{89}},
	})

	testAssertEpisodes([]tcEpisodes{
		{"multiple episodes with hyphen no spaces separator", "611-612 - Desperate Measures, Means & Ends.mp4", []int{611, 612}},
		{"multiple single episode with 10-bit notation in it", "[Final8]Suisei no Gargantia - 05 (BD 10-bit 1920x1080 x264 FLAC)[E0B15ACF].mkv", []int{5}},
		{"multiple episodes with episodes prefix and hyphen separator", "Orange Is The New Black Season 5 Episodes 1-10 INCOMPLETE (LEAKED)", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"multiple episodes with ep prefix and hyphen separator inside parentheses", "Vikings.Season.05.Ep(01-10).720p.WebRip.2Ch.x265.PSA", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"multiple episodes with hyphen separator and follower by open parenthesis", "[TBox] Dragon Ball Z Full 1-291(Subbed Jap Vers)", intRange(1, 291)},
		{"multiple episodes with e prefix and hyphen separator", "Marvel's.Agents.of.S.H.I.E.L.D.S02E01-03.Shadows.1080p.WEB-DL.DD5.1", []int{1, 2, 3}},
		{"absolute episode with ep prefix", "Naruto Shippuden Ep 107 - Strange Bedfellows.mkv", []int{107}},
		{"absolute episode in middle with hyphen dividers", "Naruto Shippuden - 107 - Strange Bedfellows.mkv", []int{107}},
		{"absolute episode in middle with similar resolution value", "[AnimeRG] Naruto Shippuden - 107 [720p] [x265] [pseudo].mkv", []int{107}},
		{"multiple absolute episodes separated by hyphen", "Naruto Shippuuden - 006-007.mkv", []int{6, 7}},
		{"absolute episode correctly not hindered by title digits with hashtag", "321 - Family Guy Viewer Mail #1.avi", []int{321}},
		{"absolute episode correctly not hindered by title digits with apostrophe", "512 - Airport '07.avi", []int{512}},
		{"absolute episode at the begining even though its mixed with season", "102 - The Invitation.avi", []int{102}},
		{"absolute episode double digit at the beginning", "02 The Invitation.mp4", []int{2}},
		{"absolute episode triple digit at the beginning with zero padded", "004 - Male Unbonding - [DVD].avi", []int{4}},
		{"multiple absolute episodes separated by comma", "The Amazing World of Gumball - 103, 104 - The Third - The Debt.mkv", []int{103, 104}},
		{"absolute episode with a possible match at the end", "The Amazing World of Gumball - 103 - The End - The Dress (720p.x264.ac3-5.1) [449].mkv", []int{103}},
		{"absolute episode with a divided episode into a part", "The Amazing World of Gumball - 107a - The Mystery (720p.x264.ac3-5.1) [449].mkv", []int{107}},
		{"absolute episode with a divided episode into b part", "The Amazing World of Gumball - 107b - The Mystery (720p.x264.ac3-5.1) [449].mkv", []int{107}},
		{"episode withing brackets with dot separator", "[5.01] Weight Loss.avi", []int{1}},
		{"episode in hundreds withing brackets with dot separator", "Dragon Ball [5.134] Preliminary Peril.mp4", []int{134}},
		{"episode with spaces and hyphen separator", "S01 - E03 - Fifty-Fifty.mkv", []int{3}},
		{"multiple episodes separated with plus", "The Office S07E25+E26 Search Committee.mp4", []int{25, 26}},
	})

	for _, tc := range []struct {
		ttitle   string
		episodes []int
	}{
		{"[animeawake] Naruto Shippuden - 124 - Art_2.mkv", []int{124}},
		{"[animeawake] Naruto Shippuden - 072 - The Quietly Approaching Threat_2.mkv", []int{72}},
		{"[animeawake] Naruto Shippuden - 120 - Kakashi Chronicles. Boys' Life on the Battlefield. Part 2.mkv", []int{120}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.episodes, result.Episodes)
		})
	}

	testAssertEpisodes([]tcEpisodes{
		{"single episode even when a possible range identifier hyphen is present", "Supernatural - S03E01 - 720p BluRay x264-Belex - Dual Audio + Legenda.mkv", []int{1}},
		{"not episodes when the range is for season", "[F-D] Fairy Tail Season 1 -6 + Extras [480P][Dual-Audio]", nil},
		{"not episodes when the range is for seasons", "House MD All Seasons (1-8) 720p Ultra-Compressed", nil},
		{"not episode when it indicates sequence of the movie in between hyhen separators", "Dragon Ball Z Movie - 09 - Bojack Unbound - 1080p", nil},
		{"not episode when it indicates sequence of the movie at the start", "09 Movie - Dragon Ball Z - Bojack Unbound", nil},
	})

	testAssertSeasonsAndEpisodes([]tcSeasonsAndEpisodes{
		{"detect episode with x separator and e prefix", "24 - S01xE03.mp4", []int{1}, []int{3}},
	})

	testAssertEpisodes([]tcEpisodes{
		{"detect episode correctly and not episode range ", "24 - S01E04 - x264 - dilpill.mkv", []int{4}},
		{"detect episode correctly and not episode range with two codecs", "24.Legacy.S01E05.720p.HEVC.x265-MeGusta", []int{5}},
		{"detect absolute episode with a version", "[F-D] Fairy.Tail.-.004v2.-. [480P][Dual-Audio].mkv", []int{4}},
		{"detect anime episode when title contains number range", "[Erai-raws] 2-5 Jigen no Ririsa - 08 [480p][Multiple Subtitle][972D0669].mkv", []int{8}},
		{"detect absolute episode with a version and ep suffix", "[Exiled-Destiny]_Tokyo_Underground_Ep02v2_(41858470).mkv", []int{2}},
	})

	testAssertSeasonsAndEpisodes([]tcSeasonsAndEpisodes{
		{"detect absolute episode and not detect any season modifier", "[a-s]_fairy_tail_-_003_-_infiltrate_the_everlue_mansion__rs2_[1080p_bd-rip][4CB16872].mkv", nil, []int{3}},
	})

	testAssertEpisodes([]tcEpisodes{
		{"episode after season with separator", "Food Wars! Shokugeki No Souma S4 - 11 (1080p)(HEVC x265 10bit)", []int{11}},
		{"not episode range for mismatch episode marker e vs ep", "Dragon Ball Super S05E53 - Ep.129.mkv", []int{53}},
		{"not episode range with other parameter ending", "DShaun.Micallefs.MAD.AS.HELL.S10E03.576p.x642-YADNUM.mkv", []int{3}},
		{"not episode range with spaced hyphen separator", "The Avengers (EMH) - S01 E15 - 459 (1080p - BluRay).mp4", []int{15}},
		{"episode with a dot and hyphen separator", "My Little Pony FiM - 6.01 - No Second Prances.mkv", []int{1}},
		{"season with a dot and episode prefix", "Desperate Housewives - Episode 1.22 - Goodbye for now.avi", []int{22}},
		{"season with a dot and episode prefix v2", "All of Us Are Dead . 2022 . S01 EP #1.2.mkv", []int{2}},
		{"episode with number in a title", "Mob Psycho 100 - 09 [1080p].mkv", []int{9}},
		{"episode with number and a hyphen after it in a title", "3-Nen D-Gumi Glass no Kamen - 13 [480p]", []int{13}},
		{"episode with of separator", "BBC Indian Ocean with Simon Reeve 5of6 Sri Lanka to Bangladesh.avi", []int{5}},
		{"episode with of separator v1", "Witches Of Salem - 2Of4 - Road To Hell - Gr.mkv", []int{2}},
		{"episode with of separator v2", "Das Boot Miniseries Original Uncut-Reevel Cd2 Of 3.avi", []int{2}},
		{"multiple episodes with multiple E sign and no separator", "Stargate Universe S01E01E02E03.mp4", []int{1, 2, 3}},
		{"multiple episodes with multiple E sign and hyphen separator", "Stargate Universe S01E01-E02-E03.mp4", []int{1, 2, 3}},
		{"multiple episodes with eps prefix and hyphen separator", "MARATHON EPISODES/Orphan Black S3 Eps.05-08.mp4", []int{5, 6, 7, 8}},
		{"multiple episodes with E sign and hyphen spaced separator", "Pokemon Black & White E10 - E17 [CW] AVI", []int{10, 11, 12, 13, 14, 15, 16, 17}},
		{"multiple episodes with E sign and hyphen separator", "Pokémon.S01E01-E04.SWEDISH.VHSRip.XviD-aka", []int{1, 2, 3, 4}},
		{"episode with single episode and not range", "[HorribleSubs] White Album 2 - 06 [1080p].mkv", []int{6}},
		{"episode with E symbols without season", "Mob.Psycho.100.II.E10.720p.WEB.x264-URANiME.mkv", []int{10}},
		{"episode with E symbols without season v2", "E5.mkv", []int{5}},
		{"episode without season", "[OMDA] Bleach - 002 (480p x264 AAC) [rich_jc].mkv", []int{2}},
		{"episode with a episode code including multiple numbers", "[ACX]El_Cazador_de_la_Bruja_-_19_-_A_Man_Who_Protects_[SSJ_Saiyan_Elite]_[9E199846].mkv", []int{19}},
		{"multiple episodes with x episode marker and hyphen separator", "BoJack Horseman [06x01-08 of 16] (2019-2020) WEB-DLRip 720p", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"multiple episodes with russian episode marker and hyphen separator", "Мистер Робот / Mr. Robot / Сезон: 2 / Серии: 1-5 (12) [2016, США, WEBRip 1080p] MVO", []int{1, 2, 3, 4, 5}},
		{"episode with russian episode marker and single episode", "Викинги / Vikings / Сезон: 5 / Серии: 1 [2017, WEB-DL 1080p] MVO", []int{1}},
		{"episode with russian episode marker and single episode and with total episodes value", "Викинги / Vikings / Сезон: 5 / Серии: 1 из 20 [2017, WEB-DL 1080p] MVO", []int{1}},
		{"episode with russian arabic total episodes value separator", "Prehistoric park.3iz6.Supercroc.DVDRip.Xvid.avi", []int{3}},
		{"episode with shortened russian episode name", "Меч (05 сер.) - webrip1080p.mkv", []int{5}},
		{"episode with full russian episode name", "Серия 11.mkv", []int{11}},
		{"episode with full different russian episode name", "Разрушители легенд. MythBusters. Сезон 15. Эпизод 09. Скрытая угроза (2015).avi", []int{9}},
		{"episode with full different russian episode name v2", "Леди Баг и Супер-Кот – Сезон 3, Эпизод 21 – Кукловод 2 [1080p].mkv", []int{21}},
		{"episode with full russian episode name with case suffix", "Проклятие острова ОУК_ 5-й сезон 09-я серия_ Прорыв Дэна.avi", []int{9}},
		{"episode with full russian episode name and no prefix", "Интерны. Сезон №9. Серия №180.avi", []int{180}},
		{"episode with russian episode name in non kirilica", "Tajny.sledstviya-20.01.serya.WEB-DL.(1080p).by.lunkin.mkv", []int{1}},
		{"episode with russian episode name in non kirilica alternative 2", "Zvezdnie.Voiny.Voina.Klonov.3.sezon.22.seria.iz.22.XviD.HDRip.avi", []int{22}},
		{"season with russian episode shortened word", "Otchayannie.domochozyaiki.(8.sez.21.ser.iz.23).2012.XviD.HDTVRip.avi", []int{21}},
		{"episode with russian episode name in non kirilica alternative 3", "MosGaz.(08.seriya).2012.WEB-DLRip(AVC).ExKinoRay.mkv", []int{8}},
		{"episode with russian episode name in non kirilica alternative 5", "Tajny.sledstvija.(2.sezon.12.serija.iz.12).2002.XviD.DVDRip.avi", []int{12}},
		{"episodes with russian x separator", "Discovery. Парни с Юкона / Yokon Men [06х01-08] (2017) HDTVRip от GeneralFilm | P1", []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{"episodes with hyphen separator between episode", "2-06. Девичья сила.mkv", []int{6}},
		{"episodes with hyphen separator between episode v2", "4-13 Cursed (HD).m4v", []int{13}},
	})

	t.Run("not episodes with hyphen separator between episode when it's date", func(t *testing.T) {
		result := Parse("The Ed Show 10-19-12.mp4")
		assert.Empty(t, result.Episodes)
		assert.Equal(t, "2012-10-19", result.Date)
	})

	testAssertEpisodes([]tcEpisodes{
		{"not episodes with hyphen separator between episode when it's not supported date", "Hogan's Heroes - 516 - Get Fit or Go Flight - 1-09-70.divx", []int{516}},
		{"episodes with hyphen separator between episode v3", "Доктор Хаус 03-20.mkv", []int{20}},
		{"episodes with hyphen separator between episode v4", "Комиссар Рекс 11-13.avi", []int{13}},
		{"episode after ordinal season and hyphen separator", "Kyoukai no Rinne (TV) 3rd Season - 23 [1080p]", []int{23}},
		{"episode after ordinal season and hyphen separator and multiple spaces", "[224] Shingeki no Kyojin - S03 - Part 1 -  13 [BDRip.1080p.x265.FLAC].mkv", []int{13}},
		{"spanish full episode identifier", "El Chema Temporada 1 Capitulo 25", []int{25}},
		{"spanish partial episode identifier", "Juego de Tronos - Temp.2 [ALTA DEFINICION 720p][Cap.209][Spanish].mkv", []int{209}},
		{"spanish partial long episode identifier", "Blue Bloods - Temporada 11 [HDTV 720p][Cap.1103][AC3 5.1 Castellano][www.PCTmix.com].mkv", []int{1103}},
		{"spanish partial episode identifier with common typo", "Para toda la humanidad [4k 2160p][Caap.406](wolfmax4k.com).mkv", []int{406}},
		{"spanish partial episode identifier v2", "Mazinger-Z-Cap-52.avi", []int{52}},
		{"latino full episode identifier", "Yu-Gi-Oh! ZEXAL Temporada 1 Episodio 009 Dual Latino e Inglés [B3B4970E].mkv", []int{9}},
		{"spanish multiple episode identifier", "Bleach 10º Temporada - 215 ao 220 - [DB-BR]", []int{215, 216, 217, 218, 219, 220}},
		{"spanish short season identifier", "My Little Pony - A Amizade é Mágica - T02E22.mp4", []int{22}},
		{"spanish short season identifier with xe separator", "30 M0N3D4S ESP T01XE08.mkv", []int{8}},
		{"not episode in episode checksum code", "[CBM]_Medaka_Box_-_11_-_This_Is_the_End!!_[720p]_[436E0E90].mkv", []int{11}},
		{"not episode in episode checksum code without container", "[CBM]_Medaka_Box_-_11_-_This_Is_the_End!!_[720p]_[436E0E90]", []int{11}},
		{"not episode in episode checksum code with paranthesis", "(Hi10)_Re_Zero_Shin_Henshuu-ban_-_02v2_(720p)_(DDY)_(72006E34).mkv", []int{2}},
		{"not episode before season", "22-7 (Season 1) (1080p)(HEVC x265 10bit)(Eng-Subs)-Judas[TGx] ⭐", nil},
		{"multiple episode with tilde separator", "[Erai-raws] Carole and Tuesday - 01 ~ 12 [1080p][Multiple Subtitle]", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"multiple episode with tilde separator and season prefix", "[Erai-raws] 3D Kanojo - Real Girl 2nd Season - 01 ~ 12 [720p]", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"multiple episode with hyphen separator", "[FFA] Koi to Producer: EVOL×LOVE - 01 - 12 [1080p][HEVC][AAC]", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"multiple episode with hyphen separator between parenthesis", "[BenjiD] Quan Zhi Gao Shou (The King’s Avatar) / Full-Time Master S01 (01 - 12) [1080p x265] [Soft sub] V2", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
	})

	for _, tc := range []struct {
		ttitle   string
		episodes []int
	}{
		{"[HR] Boku no Hero Academia 87 (S4-24) [1080p HEVC Multi-Subs] HR-GZ", []int{24}},
		{"Tokyo Ghoul Root A - 07 [S2-07] [Eng Sub] 480p [email protected]", []int{7}},
		{"black-ish.S05E02.1080p..x265.10bit.EAC3.6.0-Qman[UTR].mkv", []int{2}},
		{"[Eng Sub] Rebirth Ep #36 [8CF3ADFA].mkv", []int{36}},
		{"[92 Impatient Eilas & Miyafuji] Strike Witches - Road to Berlin - 01 [1080p][BCDFF6A2].mkv", []int{1}},
		{"[224] Darling in the FranXX - 14 [BDRip.1080p.x265.FLAC].mkv", []int{14}},
		{"[Erai-raws] Granblue Fantasy The Animation Season 2 - 10 [1080p][Multiple Subtitle].mkv", []int{10}},
		{"[Erai-raws] Shingeki no Kyojin Season 3 - 11 [1080p][Multiple Subtitle].mkv", []int{11}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.episodes, result.Episodes)
		})
	}

	testAssertEpisodes([]tcEpisodes{
		{"single zero episode", "DARKER THAN BLACK - S00E00.mkv", []int{0}},
		{"anime episode when title contain similar pattern", "[Erai-raws] 22-7 - 11 .mkv", []int{11}},
		{"anime episode after year in title", "[Golumpa] Star Blazers 2202 - 22 (Uchuu Senkan Yamato 2022) [FuniDub 1080p x264 AAC] [A24B89C8].mkv", []int{22}},
		{"anime episode after year property", "[SubsPlease] Digimon Adventure (2020) - 35 (720p) [4E7BA28A].mkv", []int{35}},
		{"anime episode with hyphen number in title", "[SubsPlease] Fairy Tail - 100 Years Quest - 05 (1080p) [1107F3A9].mkv", []int{5}},
		{"anime episode recap episode", "[KH] Sword Art Online II - 14.5 - Debriefing.mkv", []int{14}},
		{"four digit anime episode", "[SSA] Detective Conan - 1001 [720p].mkv", []int{1001}},
	})

	testAssertSeasonsAndEpisodes([]tcSeasonsAndEpisodes{
		{"season_episode pattern", "Pwer-04_05.avi", []int{4}, []int{5}},
		{"season_episode pattern v2", "SupNat-11_06.avi", []int{11}, []int{6}},
		{"season_episode pattern v3", "office_03_19.avi", []int{3}, []int{19}},
		{"season_episode pattern with years in title", "Spergrl-2016-02_04.avi", []int{2}, []int{4}},
		{"final with dash season_episode pattern with years in title", "Iron-Fist-2017-01_13-F.avi", []int{1}, []int{13}},
		{"final with dot season_episode pattern with years in title", "Lgds.of.Tmrow-02_17.F.avi", []int{2}, []int{17}},
		{"season.episode pattern", "Ozk.02.09.avi", []int{2}, []int{9}},
		{"final season.episode pattern", "Ozk.02.10.F.avi", []int{2}, []int{10}},
		{"not detect season_episode pattern when other pattern present", "Cestovatelé_S02E04_11_27.mkv", []int{2}, []int{4}},
		{"not detect season_episode pattern when it's additional info", "S03E13_91.avi", []int{3}, []int{13}},
		{"not detect season.episode pattern (not working yet)", "wwe.nxt.uk.11.26.mkv", []int{11}, []int{26}},
		{"not detect season.episode pattern when other pattern present", "Chernobyl.S01E01.1.23.45.mkv", []int{1}, []int{1}},
		{"season.episode pattern with S identifier", "The.Witcher.S01.07.mp4", []int{1}, []int{7}},
		{"season episode pattern with S identifier", "Breaking Bad S02 03.mkv", []int{2}, []int{3}},
		{"season episode pattern with Season prefix", "NCIS Season 11 01.mp4", []int{11}, []int{1}},
		{"not detect season.episode pattern when it's a date", "Top Gear - 3x05 - 2003.11.23.avi", []int{3}, []int{5}},
		{"episode in brackets", "[KTKJ]_[BLEACH]_[DVDRIP]_[116]_[x264_640x480_aac].mkv", nil, []int{116}},
		{"episode in brackets but not years", "[GM-Team][国漫][绝代双骄][Legendary Twins][2022][08][HEVC][GB][4K].mp4", nil, []int{8}},
		{"not season-episode pattern when with dot split", "SG-1. Season 4.16. (2010).avi", []int{4}, []int{16}},
		// {"not season-episode pattern when it's a date", "8-6 2006.07.16.avi", []int{8}, []int{6}},
		{"not season episode pattern but absolute episode", "523 23.mp4", nil, []int{523}},
		{"only episode", "Chernobyl E02 1 23 45.mp4", nil, []int{2}},
		{"regular episode with year range before", "'Lucky.Luke.1983-1992.S01E04.PL.720p.WEB-DL.H264-zyl.mkv'", []int{1}, []int{4}},
		{"only episode v2", "Watch Gary And His Demons Episode 10 - 0.00.07-0.11.02.mp4", nil, []int{10}},
		{"only episode v3", "523 23.mp4", nil, []int{523}},
		{"not season.episode pattern when it's a date without other pattern", "wwf.raw.is.war.18.09.00.avi", nil, nil},
		{"not episodes when it's 2.0 sound", "The Rat Race (1960) [1080p] [WEB-DL] [x264] [DD] [2-0] [DReaM] [LEKTOR PL]", nil, nil},
		{"not episodes when it's 2.0 sound v2", "Avatar 2009 [1080p.BDRip.x264.AC3-azjatycki] [2.0] [Lektor PL]", nil, nil},
		{"not episodes when it's 5.1 sound", "A Quiet Place: Day One (2024) [1080p] [WEB-DL] [x264] [AC3] [DD] [5-1] [LEKTOR PL]", nil, nil},
		{"not episodes when it's 5.1 sound v2", "Avatar 2009 [1080p.BDRip.x264.AC3-azjatycki] [5.1] [Lektor PL]", nil, nil},
		{"not episodes when it's 7.1 sound", "Frequency (2000) [1080p] [BluRay] [REMUX] [AVC] [DTS] [HD] [MA] [7-1] [MR]", nil, nil},
		{"not episodes when it's 7.1 sound v2", "Avatar 2009 [1080p.BDRip.x264.AC3-azjatycki] [7.1] [Lektor PL]", nil, nil},
	})

	testAssertEpisodes([]tcEpisodes{
		{"", "Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano][www.AtomoHD.nu].avi", []int{1905}},
		{"", "[SubsPlease] Fairy Tail - 100 Years Quest - 05 (1080p) [1107F3A9].mkv", []int{5}},
		{"", "Mad.Max.Fury.Road.2015.1080p.BluRay.DDP5.1.x265.10bit-GalaxyRG265[TGx]", nil},
		{"", "Vikkatakavi 01E06.mkv", []int{6}},
		{"", "[Deadfish] Hakkenden_Touhou Hakken Ibun S2 [720][AAC]", nil},
		{"", "[Anime Time] Naruto - 116 - 360 Degrees of Vision The Byakugan's Blind Spot.mkv", []int{116}},
	})
}
