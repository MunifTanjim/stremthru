package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestUnrated(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ttitle  string
		unrated bool
	}{
		{"unrated", "Identity.Thief.2013.Vostfr.UNRATED.BluRay.720p.DTS.x264-Nenuko", true},
		{"uncensored", "Charlie.les.filles.lui.disent.merci.2007.UNCENSORED.TRUEFRENCH.DVDRiP.AC3.Libe", true},
		{"not unrated", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.unrated, result.Unrated)
		})
	}
}
