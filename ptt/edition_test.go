package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEdition(t *testing.T) {
	for _, tc := range []struct {
		ttitle  string
		edition string
	}{
		{"Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", "Extended Edition"},
		{"Mary.Poppins.1964.50th.ANNIVERSARY.EDITION.REMUX.1080p.Bluray.AVC.DTS-HD.MA.5.1-LEGi0N", "Anniversary Edition"},
		{"The.Lord.of.the.Rings.The.Fellowship.of.the.Ring.2001.EXTENDED.2160p.UHD.BluRay.x265.10bit.HDR.TrueHD.7.1.Atmos-BOREDOR", "Extended Edition"},
		{"The.Lord.of.the.Rings.The.Motion.Picture.Trilogy.Extended.Editions.2001-2003.1080p.BluRay.x264.DTS-WiKi", "Extended Edition"},
		{"Better.Call.Saul.S03E04.CONVERT.720p.WEB.h264-TBS", ""},
		{"The Fifth Element 1997 REMASTERED MULTi 1080p BluRay HDLight AC3 x264 Zone80", "Remastered"},
		{"Predator 1987 REMASTER MULTi 1080p BluRay x264 FiDELiO", "Remastered"},
		{"Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", "Extended Edition"},
		{"Uncut.Gems.2019.1080p.NF.WEB-DL.DDP5.1.x264-NTG", ""},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.edition, result.Edition)
		})
	}
}
