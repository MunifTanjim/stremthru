package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPTT(t *testing.T) {
	for _, tc := range []struct {
		ttitle string
		result Result
	}{
		{"sons.of.anarchy.s05e10.480p.BluRay.x264-GAnGSteR", Result{
			Title:      "sons of anarchy",
			Resolution: "480p",
			Seasons:    []int{5},
			Episodes:   []int{10},
			Source:     "BluRay",
			Codec:      "x264",
			Group:      "GAnGSteR",
		}},
		{"Color.Of.Night.Unrated.DC.VostFR.BRrip.x264", Result{
			Title:     "Color Of Night",
			Unrated:   true,
			Languages: []string{"french"},
			Source:    "BRRip",
			Codec:     "x264",
		}},
		{"Da Vinci Code DVDRip", Result{
			Title:  "Da Vinci Code",
			Source: "DVDRip",
		}},
		{"Some.girls.1998.DVDRip", Result{
			Title:  "Some girls",
			Source: "DVDRip",
			Year:   "1998",
		}},
		{"Ecrit.Dans.Le.Ciel.1954.MULTI.DVDRIP.x264.AC3-gismo65", Result{
			Title:     "Ecrit Dans Le Ciel",
			Source:    "DVDRip",
			Year:      "1954",
			Languages: []string{"multi audio"},
			Dubbed:    true,
			Codec:     "x264",
			Audio:     "ac3",
			Group:     "gismo65",
		}},
		{"2019 After The Fall Of New York 1983 REMASTERED BDRip x264-GHOULS", Result{
			Title:      "2019 After The Fall Of New York",
			Source:     "BDRip",
			Remastered: true,
			Year:       "1983",
			Codec:      "x264",
			Group:      "GHOULS",
		}},
		{"Ghost In The Shell 2017 720p HC HDRip X264 AC3-EVO", Result{
			Title:      "Ghost In The Shell",
			Source:     "HDRip",
			Hardcoded:  true,
			Year:       "2017",
			Resolution: "720p",
			Codec:      "x264",
			Audio:      "ac3",
			Group:      "EVO",
		}},
		{"Rogue One 2016 1080p BluRay x264-SPARKS", Result{
			Title:      "Rogue One",
			Source:     "BluRay",
			Year:       "2016",
			Resolution: "1080p",
			Codec:      "x264",
			Group:      "SPARKS",
		}},
		{"Desperation 2006 Multi Pal DvdR9-TBW1973", Result{
			Title:     "Desperation",
			Source:    "DVD",
			Year:      "2006",
			Languages: []string{"multi audio"},
			Dubbed:    true,
			Region:    "R9",
			Group:     "TBW1973",
		}},
		{"Maman, j'ai raté l'avion 1990 VFI 1080p BluRay DTS x265-HTG", Result{
			Title:      "Maman, j'ai raté l'avion",
			Source:     "BluRay",
			Year:       "1990",
			Audio:      "dts",
			Resolution: "1080p",
			Languages:  []string{"french"},
			Codec:      "x265",
			Group:      "HTG",
		}},
		{"Game of Thrones - The Complete Season 3 [HDTV]", Result{
			Title:   "Game of Thrones",
			Seasons: []int{3},
			Source:  "HDTV",
		}},
		{"The Sopranos: The Complete Series (Season 1,2,3,4,5&6) + Extras", Result{
			Title:    "The Sopranos",
			Seasons:  []int{1, 2, 3, 4, 5, 6},
			Complete: true,
		}},
		{"Skins Season S01-S07 COMPLETE UK Soundtrack 720p WEB-DL", Result{
			Title:      "Skins",
			Seasons:    []int{1, 2, 3, 4, 5, 6, 7},
			Resolution: "720p",
			Source:     "WEB-DL",
		}},
		{"Futurama.COMPLETE.S01-S07.720p.BluRay.x265-HETeam", Result{
			Title:      "Futurama",
			Seasons:    []int{1, 2, 3, 4, 5, 6, 7},
			Resolution: "720p",
			Source:     "BluRay",
			Codec:      "x265",
			Group:      "HETeam",
		}},
		{"You.[Uncut].S01.SweSub.1080p.x264-Justiso", Result{
			Title:      "You",
			Seasons:    []int{1},
			Languages:  []string{"swedish"},
			Resolution: "1080p",
			Codec:      "x264",
			Group:      "Justiso",
		}},
		{"Stephen Colbert 2019 10 25 Eddie Murphy 480p x264-mSD [eztv]", Result{
			Title:      "Stephen Colbert",
			Date:       "2019-10-25",
			Resolution: "480p",
			Codec:      "x264",
		}},
		{"House MD Season 7 Complete MKV", Result{
			Title:     "House MD",
			Seasons:   []int{7},
			Container: "mkv",
		}},
		{"2008 The Incredible Hulk Feature Film.mp4", Result{
			Title:     "The Incredible Hulk Feature Film",
			Year:      "2008",
			Container: "mp4",
			Extension: "mp4",
		}},
		{"【4月/悠哈璃羽字幕社】[UHA-WINGS][不要输！恶之军团][Makeruna!! Aku no Gundan!][04][1080p AVC_AAC][简繁外挂][sc_tc]", Result{
			Title:      "Makeruna!! Aku no Gundan!",
			Episodes:   []int{4},
			Resolution: "1080p",
			Codec:      "avc",
			Audio:      "aac",
		}},
		{"[GM-Team][国漫][西行纪之集结篇][The Westward Ⅱ][2019][17][AVC][GB][1080P]", Result{
			Title:      "The Westward Ⅱ",
			Year:       "2019",
			Episodes:   []int{17},
			Resolution: "1080p",
			Codec:      "avc",
			Group:      "GM-Team",
		}},
		{"Черное зеркало / Black Mirror / Сезон 4 / Серии 1-6 (6) [2017, США, WEBRip 1080p] MVO + Eng Sub", Result{
			Title:      "Black Mirror",
			Year:       "2017",
			Seasons:    []int{4},
			Episodes:   []int{1, 2, 3, 4, 5, 6},
			Languages:  []string{"english"},
			Resolution: "1080p",
			Source:     "WEBRip",
		}},
		{"[neoHEVC] Student Council's Discretion / Seitokai no Ichizon [Season 1} [BD 1080p x265 HEVC AAC]", Result{
			Title:      "Student Council's Discretion / Seitokai no Ichizon",
			Seasons:    []int{1},
			Resolution: "1080p",
			Source:     "BDRip",
			Audio:      "aac",
			Codec:      "hevc",
			Group:      "neoHEVC",
		}},
		{"[Commie] Chihayafuru 3 - 21 [BD 720p AAC] [5F1911ED].mkv", Result{
			Title:       "Chihayafuru 3",
			Episodes:    []int{21},
			Resolution:  "720p",
			Source:      "BDRip",
			Audio:       "aac",
			Container:   "mkv",
			Extension:   "mkv",
			EpisodeCode: "5F1911ED",
			Group:       "Commie",
		}},
		{"[DVDRip-ITA]The Fast and the Furious: Tokyo Drift [CR-Bt]", Result{
			Title:     "The Fast and the Furious: Tokyo Drift",
			Source:    "DVDRip",
			Languages: []string{"italian"},
		}},
		{"[BluRay Rip 720p ITA AC3 - ENG AC3 SUB] Hostel[2005]-LIFE[ultimafrontiera]", Result{
			Title:      "Hostel",
			Year:       "2005",
			Resolution: "720p",
			Source:     "BRRip",
			Audio:      "ac3",
			Languages:  []string{"english", "italian"},
			Group:      "LIFE",
		}},
		{"[OFFICIAL ENG SUB] Soul Land Episode 121-125 [1080p][Soft Sub][Web-DL][Douluo Dalu][斗罗大陆]", Result{
			Title:      "Soul Land",
			Episodes:   []int{121, 122, 123, 124, 125},
			Resolution: "1080p",
			Source:     "WEB-DL",
			Languages:  []string{"english"},
		}},
		{"[720p] The God of Highschool Season 1", Result{
			Title:      "The God of Highschool",
			Seasons:    []int{1},
			Resolution: "720p",
		}},
		{"Heidi Audio Latino DVDRip [cap. 3 Al 18}", Result{
			Title:     "Heidi",
			Episodes:  []int{3},
			Source:    "DVDRip",
			Languages: []string{"latino"},
		}},
		{"Sprint.2024.S01.COMPLETE.1080p.WEB.h264-EDITH[TGx]", Result{
			Title:      "Sprint",
			Year:       "2024",
			Seasons:    []int{1},
			Resolution: "1080p",
			Codec:      "h264",
			Group:      "EDITH",
		}},
		{"High Heat *2022* [BRRip.XviD] Lektor PL", Result{
			Title:     "High Heat",
			Year:      "2022",
			Codec:     "xvid",
			Source:    "BRRip",
			Languages: []string{"polish"},
		}},
		{"Ghost Busters *1984-2021* [720p] [BDRip] [AC-3] [XviD] [Lektor + Dubbing PL] [DYZIO]", Result{
			Title:      "Ghost Busters",
			Year:       "1984-2021",
			Resolution: "720p",
			Audio:      "ac3",
			Codec:      "xvid",
			Source:     "BDRip",
			Languages:  []string{"polish"},
		}},
		{"20-20.2024.11.15.Fatal.Disguise.XviD-AFG[EZTVx.to].avi", Result{
			Title:     "20-20",
			Date:      "2024-11-15",
			Codec:     "xvid",
			Source:    "XviD",
			Group:     "AFG",
			Container: "avi",
			Extension: "avi",
		}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, &tc.result, result)
		})
	}
}
