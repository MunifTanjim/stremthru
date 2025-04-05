package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAudio(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		audio  []string
	}{
		{"the dts audio correctly", "Nocturnal Animals 2016 VFF 1080p BluRay DTS HEVC-HD2", []string{"DTS Lossy"}},
		{"the DTS-HD audio correctly", "Gold 2016 1080p BluRay DTS-HD MA 5 1 x264-HDH", []string{"DTS Lossless"}},
		{"the AAC audio correctly", "Rain Man 1988 REMASTERED 1080p BRRip x264 AAC-m2g", []string{"AAC"}},
		{"convert the AAC2.0 audio to AAC", "The Vet Life S02E01 Dunk-A-Doctor 1080p ANPL WEB-DL AAC2 0 H 264-RTN", []string{"AAC"}},
		{"the dd5 audio correctly", "Jimmy Kimmel 2017 05 03 720p HDTV DD5 1 MPEG2-CTL", []string{"DD"}},
		{"the AC3 audio correctly", "A Dog's Purpose 2016 BDRip 720p X265 Ac3-GANJAMAN", []string{"AC3"}},
		{"convert the AC-3 audio to AC3", "Retroactive 1997 BluRay 1080p AC-3 HEVC-d3g", []string{"AC3"}},
		{"the mp3 audio correctly", "Tempete 2016-TrueFRENCH-TVrip-H264-mp3", []string{"MP3"}},
		// {"the MD audio correctly", "Detroit.2017.BDRip.MD.GERMAN.x264-SPECTRE", []string{"md"}},
		{"the eac3 5.1 audio correctly", "The Blacklist S07E04 (1080p AMZN WEB-DL x265 HEVC 10bit EAC-3 5.1)[Bandi]", []string{"EAC3"}},
		{"the eac3 6.0 audio correctly", "Condor.S01E03.1080p.WEB-DL.x265.10bit.EAC3.6.0-Qman[UTR].mkv", []string{"EAC3"}},
		{"the eac3 2.0 audio correctly 2", "The 13 Ghosts of Scooby-Doo (1985) S01 (1080p AMZN Webrip x265 10bit EAC-3 2.0 - Frys) [TAoE]", []string{"EAC3"}},
		{"not mp3 audio inside a word", "[Thund3r3mp3ror] Attack on Titan - 23.mp4", []string(nil)},
		{"2.0x2 audio", "Buttobi!! CPU - 02 (DVDRip 720x480p x265 HEVC AC3x2 2.0x2)(Dual Audio)[sxales].mkv", []string{"AC3"}},
		{"qaac2 audio", "[naiyas] Fate Stay Night - Unlimited Blade Works Movie [BD 1080P HEVC10 QAACx2 Dual Audio]", []string{"AAC"}},
		{"2.0x5.1 audio", "Sakura Wars the Movie (2001) (BDRip 1920x1036p x265 HEVC FLACx2, AC3 2.0+5.1x2)(Dual Audio)[sxales].mkv", []string{"FLAC", "AC3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
		})
	}

	for _, tc := range []struct {
		name     string
		ttitle   string
		audio    []string
		episodes []int
	}{
		{"5.1x2.0 audio", "Macross ~ Do You Remember Love (1984) (BDRip 1920x1036p x265 HEVC DTS-HD MA, FLAC, AC3x2 5.1+2.0x3)(Dual Audio)[sxales].mkv", []string{"DTS Lossless", "FLAC", "AC3"}, nil},
		{"5.1x2+2.0x3 audio", "Escaflowne (2000) (BDRip 1896x1048p x265 HEVC TrueHD, FLACx3, AC3 5.1x2+2.0x3)(Triple Audio)[sxales].mkv", []string{"TrueHD", "FLAC", "AC3"}, nil},
		{"FLAC2.0x2 audio", "[SAD] Inuyasha - The Movie 4 - Fire on the Mystic Island [BD 1920x1036 HEVC10 FLAC2.0x2] [84E9A4A1].mkv", []string{"FLAC"}, nil},
		{"FLACx2 2.0x3 audio", "Outlaw Star - 23 (BDRip 1440x1080p x265 HEVC AC3, FLACx2 2.0x3)(Dual Audio)[sxales].mkv", []string{"FLAC", "AC3"}, []int{23}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
		})
	}

	for _, tc := range []struct {
		name     string
		ttitle   string
		audio    []string
		channels []string
	}{
		{"7.1 Atmos audio", "Spider-Man.No.Way.Home.2021.2160p.BluRay.REMUX.HEVC.TrueHD.7.1.Atmos-FraMeSToR", []string{"Atmos", "TrueHD"}, []string{"7.1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
			assert.Equal(t, tc.channels, result.Channels)
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle string
		audio  []string
		title  string
	}{
		{"The Shawshank Redemption 1994.MULTi.1080p.Blu-ray.DTS-HDMA.5.1.HEVC-DDR[EtHD]", []string{"DTS Lossless"}, "The Shawshank Redemption"},
		{"Oppenheimer.2023.BluRay.1080p.DTS-HD.MA.5.1.AVC.REMUX-FraMeSToR.mkv", []string{"DTS Lossless"}, "Oppenheimer"},
		{"Guardians.of.the.Galaxy.Vol.3.2023.BluRay.1080p.DTS-HD.MA.7.1.x264-MTeam[TGx]", []string{"DTS Lossless"}, "Guardians of the Galaxy Vol 3"},
		{"Oppenheimer.2023.2160p.MA.WEB-DL.DUAL.DTS.HD.MA.5.1+DD+5.1.DV-HDR.H.265-TheBiscuitMan.mkv", []string{"DTS Lossless", "DDP"}, "Oppenheimer"},
		{"The.Equalizer.3.2023.BluRay.1080p.DTS-HD.MA.5.1.x264-MTeam", []string{"DTS Lossless"}, "The Equalizer 3"},
		{"Point.Break.1991.2160p.Blu-ray.Remux.DV.HDR.HEVC.DTS-HD.MA.5.1-CiNEPHiLES.mkv", []string{"DTS Lossless"}, "Point Break"},
		{"The.Mechanic.2011.2160p.UHD.Blu-ray.Remux.DV.HDR.HEVC.DTS-HD.MA.5.1-CiNEPHiLES.mkv", []string{"DTS Lossless"}, "The Mechanic"},
		{"Face.Off.1997.UHD.BluRay.2160p.DTS-HD.MA.5.1.DV.HEVC.REMUX-FraMeSToR.mkv", []string{"DTS Lossless"}, "Face Off"},
		{"Killers of the Flower Moon 2023 2160p UHD Blu-ray Remux HEVC DV DTS-HD MA 5.1-HDT.mkv", []string{"DTS Lossless"}, "Killers of the Flower Moon"},
		{"Ghostbusters.Frozen.Empire.2024.1080p.BluRay.ENG.LATINO.HINDI.ITA.DTS-HD.Master.5.1.H264-BEN.THE.MEN", []string{"DTS Lossless"}, "Ghostbusters Frozen Empire"},
		{"How.To.Train.Your.Dragon.2.2014.1080p.BluRay.ENG.LATINO.DTS-HD.Master.H264-BEN.THE.MEN", []string{"DTS Lossless"}, "How To Train Your Dragon 2"},
		{"【高清影视之家发布 www.HDBTHD.com】奥本海默[IMAX满屏版][简繁英字幕].Oppenheimer.2023.IMAX.2160p.BluRay.x265.10bit.DTS-HD.MA.5.1-CTRLHD", []string{"DTS Lossless"}, "Oppenheimer"},
		{"Ocean's.Thirteen.2007.UHD.BluRay.2160p.DTS-HD.MA.5.1.DV.HEVC.HYBRID.REMUX-FraMeSToR.mkv", []string{"DTS Lossless"}, "Ocean's Thirteen"},
		{"Sleepy.Hollow.1999.BluRay.1080p.2Audio.DTS-HD.HR.5.1.x265.10bit-ALT", []string{"DTS Lossy"}, "Sleepy Hollow"},
		{"The Flash 2023 WEBRip 1080p DTS DD+ 5.1 Atmos x264-MgB", []string{"DTS Lossy", "Atmos", "DDP"}, "The Flash"},
		{"Indiana Jones and the Last Crusade 1989 BluRay 1080p DTS AC3 x264-MgB", []string{"DTS Lossy", "AC3"}, "Indiana Jones and the Last Crusade"},
		{"2012.London.Olympics.BBC.Bluray.Set.1080p.DTS-HD", []string{"DTS Lossy"}, "London Olympics BBC"},
		{"www.1TamilMV.phd - Oppenheimer (2023) English BluRay - 1080p - x264 - (DTS 5.1) - 7.3GB - ESub.mkv", []string{"DTS Lossy"}, "Oppenheimer"},
		{"【高清影视之家发布 www.HDBTHD.com】年会不能停！[60帧率版本][国语音轨+中文字幕].Johnny.Keep.Walking.2023.60FPS.2160p.WEB-DL.H265.10bit.DTS.5.1-GPTHD", []string{"DTS Lossy"}, "Johnny Keep Walking"},
		{"Big.Stan.2007.1080p.BluRay.Remux.DTS-HD.HR.5.1", []string{"DTS Lossy"}, "Big Stan"},
		{"Ditched.2022.1080p.Bluray.DTS-HD.HR.5.1.X264-EVO[TGx]", []string{"DTS Lossy"}, "Ditched"},
		{"Basic.Instinct.1992.Unrated.Directors.Cut.Bluray.1080p.DTS-HD-HR-6.1.x264-Grym@BTNET", []string{"DTS Lossy"}, "Basic Instinct"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
			assert.Equal(t, tc.title, result.Title)
		})
	}
}
