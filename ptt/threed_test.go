package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreeD(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		threeD string
	}{
		{"3D HSBS together", "Incredibles 2 (2018) 3D HSBS 1080p BluRay H264 DolbyD 5.1 + nickarad", "3D HSBS"},
		{"3D H-SBS apart", "Despicable.Me.2010.1080p.PROPER.3D.BluRay.H-SBS.x264-CULTHD [Pub", "3D HSBS"},
		{"3D Half-SBS apart", "Avengers.Infinity.War.2018.3D.BluRay.1080p.Half-SBS.DTS.x264-CHC", "3D HSBS"},
		{"3D Half-SBS when 3D is in the name", "Gravity.3D.2013.1080p.BluRay.Half-SBS.DTS.x264-PublicHD", "3D HSBS"},
		{"3D SBS apart", "Guardians of the Galaxy Vol 3 2023 1080p 3D BluRay SBS x264", "3D SBS"},
		{"3D SBS together", "3-D Zen Extreme Ecstasy 3D SBS (2011) [BDRip 1080p].avi", "3D SBS"},
		{"3D SBS when 3D is small letter", "Saw 3D (2010) 1080p 3d BrRip x264 SBS - 1.3GB - YIFY", "3D SBS"},
		{"3D Full-SBS", "Puss.In.Boots.The.Last.Wish.3D.(2022).Full-SBS.1080p.x264.ENG.AC3-JFC", "3D SBS"},
		{"3D HOU together", "The Lego Ninjago Movie (2017) 3D HOU German DTS 1080p BluRay x264", "3D HOU"},
		{"3D HOU apart", "47 Ronin 2013 3D BluRay HOU 1080p DTS x264-CHD3D", "3D HOU"},
		{"3D H-OU", "The Three Musketeers 3D 2011 1080p H-OU BDRip x264 ac3 vice", "3D HOU"},
		{"3D H/OU", "Kiss Me, Kate 3D (1953) [BRRip.1080p.x264.3D H/OU-DTS/AC3] [Lektor PL] [Eng]", "3D HOU"},
		{"3D Half-OU", "Pixels.2015.1080p.3D.BluRay.Half-OU.x264.DTS-HD.MA.7.1-RARBG", "3D HOU"},
		{"3D HalfOU", "Солнце 3D / 3D Sun (2007) BDRip 1080p от Ash61 | 3D-Video | halfOU | L1", "3D HOU"},
		{"3D OU together", "Amazing Africa (2013) 3D OU 2160p Eng Rus", "3D OU"},
		{"3D Full OU together", "For the Birds (2000) 3D Full OU 1080p", "3D OU"},
		{"3D", "Incredibles 2 2018 3D BluRay", "3D"},
		{"3D with dots", "Despicable.Me.3.2017.1080p.3D.BluRay.AVC.DTS-X.7.1-FGT", "3D"},
		{"3D with hyphen separator", "Гемини / Gemini Man (2019) BDRemux 1080p от селезень | 3D-Video | Лицензия", "3D"},
		{"3D in brackets", "Pokémon Detective Pikachu (2019) [BluRay] [3D]", "3D"},
		{"3D in brackets with something else", "Doctor Strange in the Multiverse of Madness (2022) [1080p 3D] [B", "3D"},
		{"HD3D", "Бамблби / Bumblebee [2018, BDRemux, 1080p] BD3D", "3D"},
		{"SideBySide", "Дэдпул и Росомаха / Deadpool & Wolverine [2024, BDRip, 1080p] SideBySide", "3D SBS"},
		{"Half SideBySide", "Вий / Forbidden Kingdom [2014, WEB-DL] Half SideBySide", "3D HSBS"},
		{"OverUnder", "Дэдпул и Росомаха / Deadpool & Wolverine [2024, BDRip, 1080p] OverUnder", "3D OU"},
		{"Half OverUnder", "Миссия «Луна» / Лунный / Mooned [2023, BDRip] Half OverUnder", "3D HOU"},
		{"not 3D in name", "Texas.Chainsaw.3D.2013.PROPER.1080p.BluRay.x264-LiViDiTY", ""},
		{"not 3D in name v2", "Step Up 3D (2010) 720p BrRip x264 - 650MB - YIFY", ""},
		{"not 3D in name v3", "[YakuboEncodes] 3D Kanojo Real Girl - 01 ~ 24 [BD 1080p 10bit x265 HEVC][Dual-Audio Opus][Multi-Subs]", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.threeD, result.ThreeD)
		})
	}
}
