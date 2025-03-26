package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestExtended(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ttitle   string
		extended bool
	}{
		{"is extended", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", true},
		{"extended in the title when separated", "Ghostbusters - Extended (2016) 1080p H265 BluRay Rip ita eng AC3 5.1", true},
		{"not extended", "Better.Call.Saul.S03E04.CONVERT.720p.WEB.h264-TBS", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.extended, result.Extended)
		})
	}
}
