package ptt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodec(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		codec  string
	}{
		{"hevc", "Nocturnal Animals 2016 VFF 1080p BluRay DTS HEVC-HD2", "hevc"},
		{"x264", "doctor_who_2005.8x12.death_in_heaven.720p_hdtv_x264-fov", "x264"},
		{"web-dl", "The Vet Life S02E01 Dunk-A-Doctor 1080p ANPL WEB-DL AAC2 0 H 264-RTN", "h264"},
		{"xvid", "Gotham S03E17 XviD-AFG", "xvid"},
		{"mpeg2", "Jimmy Kimmel 2017 05 03 720p HDTV DD5 1 MPEG2-CTL", "mpeg2"},
		{"hvec10bit", "[Anime Time] Re Zero kara Hajimeru Isekai Seikatsu (Season 2 Part 1) [1080p][HEVC10bit x265][Multi Sub]", "hevc"},
		{"hevc10", "[naiyas] Fate Stay Night - Unlimited Blade Works Movie [BD 1080P HEVC10 QAACx2 Dual Audio]", "hevc"},
		{"skip 264 from episode number", "[DB]_Bleach_264_[012073FE].avi", ""},
		{"skip 265 from episode number", "[DB]_Bleach_265_[B4A04EC9].avi", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.codec, result.Codec)
			if strings.Contains(tc.ttitle, "hevc10") {
				assert.Equal(t, "10bit", result.BitDepth)
			}
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle string
		codec  string
	}{
		{"Mad.Max.Fury.Road.2015.1080p.BluRay.DDP5.1.x265.10bit-GalaxyRG265[TGx]", "x265"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.codec, result.Codec)
			if strings.Contains(tc.ttitle, "10bit") {
				assert.Equal(t, "10bit", result.BitDepth)
			}
		})
	}
}
