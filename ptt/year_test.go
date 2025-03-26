package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestYear(t *testing.T) {
	// JS
	for _, tc := range []struct {
		name   string
		ttitle string
		year   string
	}{
		{"year", "Dawn.of.the.Planet.of.the.Apes.2014.HDRip.XViD-EVO", "2014"},
		{"year within braces", "Hercules (2014) 1080p BrRip H264 - YIFY", "2014"},
		{"year within brackets", "One Shot [2014] DVDRip XViD-ViCKY", "2014"},
		{"year but not the title if the title is a year", "2012 2009 1080p BluRay x264 REPACK-METiS", "2009"},
		{"year at the beginning if there is none", "2008 The Incredible Hulk Feature Film.mp4'", "2008"},
		{"year range", "Harry Potter All Movies Collection 2001-2011 720p Dual KartiKing'", "2001-2011"},
		{"year range with simplified end year", "Empty Nest Season 1 (1988 - 89) fiveofseven", "1988-1989"},
		{"not detect year from bitrate", "04. Practice Two (1324mb 1916x1080 50fps 1970kbps x265 deef).mkv", ""},
		{"not detect year spanish episode", "Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano][www.AtomoHD.nu].avi", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.year, result.Year)
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle string
		year   string
	}{
		{"Dawn.of.the.Planet.of.the.Apes.2014.HDRip.XViD-EVO", "2014"},
		{"Hercules (2014) 1080p BrRip H264 - YIFY", "2014"},
		{"One Shot [2014] DVDRip XViD-ViCKY", "2014"},
		{"2012 2009 1080p BluRay x264 REPACK-METiS", "2009"},
		{"2008 The Incredible Hulk Feature Film.mp4", "2008"},
		{"Harry Potter All Movies Collection 2001-2011 720p Dual KartiKing", "2001-2011"},
		{"Empty Nest Season 1 (1988 - 89) fiveofseven", "1988-1989"},
		{"04. Practice Two (1324mb 1916x1080 50fps 1970kbps x265 deef).mkv", ""},
		{"Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano][www.AtomoHD.nu].avi", ""},
		{"Wonder Woman 1984 (2020) [UHDRemux 2160p DoVi P8 Es-DTSHD AC3 En-AC3].mkv", "2020"},
		{"1923 S02E01 The Killing Season 1080p AMZN WEB-DL DDP5 1 H 264-FLUX[TGx]", ""},
		{"1883 - Season 1 (S01) (A Yellowstone Origin Story) [2160p NVEnc 10Bit HVEC][DDP 5.1Ch][WEBRip][English Subs]", ""},
		{"1883.S01E01.1883.2160p.WEB-DL.DDP5.1.H.265-NTb.mkv", ""},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.year, result.Year)
		})
	}
}
