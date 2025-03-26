package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestProper(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		proper bool
	}{
		{"release is proper", "Into the Badlands S02E07 PROPER 720p HDTV x264-W4F", true},
		{"release is real proper", "Bossi-Reality-REAL PROPER-CDM-FLAC-1999-MAHOU", true},
		{"not proper", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.proper, result.Proper)
		})
	}
}
