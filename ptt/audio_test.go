package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAudio(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		audio  string
	}{
		{"the dts audio correctly", "Nocturnal Animals 2016 VFF 1080p BluRay DTS HEVC-HD2", "dts"},
		{"the DTS-HD audio correctly", "Gold 2016 1080p BluRay DTS-HD MA 5 1 x264-HDH", "dts-hd"},
		{"the AAC audio correctly", "Rain Man 1988 REMASTERED 1080p BRRip x264 AAC-m2g", "aac"},
		{"convert the AAC2.0 audio to AAC", "The Vet Life S02E01 Dunk-A-Doctor 1080p ANPL WEB-DL AAC2 0 H 264-RTN", "aac"},
		{"the dd5 audio correctly", "Jimmy Kimmel 2017 05 03 720p HDTV DD5 1 MPEG2-CTL", "dd5.1"},
		{"the AC3 audio correctly", "A Dog's Purpose 2016 BDRip 720p X265 Ac3-GANJAMAN", "ac3"},
		{"convert the AC-3 audio to AC3", "Retroactive 1997 BluRay 1080p AC-3 HEVC-d3g", "ac3"},
		{"the mp3 audio correctly", "Tempete 2016-TrueFRENCH-TVrip-H264-mp3", "mp3"},
		// {"the MD audio correctly", "Detroit.2017.BDRip.MD.GERMAN.x264-SPECTRE", "md"},
		{"the eac3 5.1 audio correctly", "The Blacklist S07E04 (1080p AMZN WEB-DL x265 HEVC 10bit EAC-3 5.1)[Bandi]", "eac3"},
		{"the eac3 6.0 audio correctly", "Condor.S01E03.1080p.WEB-DL.x265.10bit.EAC3.6.0-Qman[UTR].mkv", "eac3"},
		{"the eac3 2.0 audio correctly 2", "The 13 Ghosts of Scooby-Doo (1985) S01 (1080p AMZN Webrip x265 10bit EAC-3 2.0 - Frys) [TAoE]", "eac3"},
		{"not mp3 audio inside a word", "[Thund3r3mp3ror] Attack on Titan - 23.mp4", ""},
		{"2.0x2 audio", "Buttobi!! CPU - 02 (DVDRip 720x480p x265 HEVC AC3x2 2.0x2)(Dual Audio)[sxales].mkv", "2.0"},
		{"qaac2 audio", "[naiyas] Fate Stay Night - Unlimited Blade Works Movie [BD 1080P HEVC10 QAACx2 Dual Audio]", "aac"},
		{"2.0x5.1 audio", "Sakura Wars the Movie (2001) (BDRip 1920x1036p x265 HEVC FLACx2, AC3 2.0+5.1x2)(Dual Audio)[sxales].mkv", "2.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
		})
	}

	for _, tc := range []struct {
		name     string
		ttitle   string
		audio    string
		episodes []int
	}{
		{"5.1x2.0 audio", "Macross ~ Do You Remember Love (1984) (BDRip 1920x1036p x265 HEVC DTS-HD MA, FLAC, AC3x2 5.1+2.0x3)(Dual Audio)[sxales].mkv", "2.0", nil},
		{"5.1x2+2.0x3 audio", "Escaflowne (2000) (BDRip 1896x1048p x265 HEVC TrueHD, FLACx3, AC3 5.1x2+2.0x3)(Triple Audio)[sxales].mkv", "2.0", nil},
		{"FLAC2.0x2 audio", "[SAD] Inuyasha - The Movie 4 - Fire on the Mystic Island [BD 1920x1036 HEVC10 FLAC2.0x2] [84E9A4A1].mkv", "flac", nil},
		{"FLACx2 2.0x3 audio", "Outlaw Star - 23 (BDRip 1440x1080p x265 HEVC AC3, FLACx2 2.0x3)(Dual Audio)[sxales].mkv", "2.0", []int{23}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
		})
	}

	for _, tc := range []struct {
		name   string
		ttitle string
		audio  string
	}{
		{"7.1 Atmos audio", "Spider-Man.No.Way.Home.2021.2160p.BluRay.REMUX.HEVC.TrueHD.7.1.Atmos-FraMeSToR", "7.1 Atmos"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.audio, result.Audio)
		})
	}
}
