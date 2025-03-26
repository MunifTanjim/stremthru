package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ttitle  string
		convert bool
	}{
		{"is convert", "Better.Call.Saul.S03E04.CONVERT.720p.WEB.h264-TBS", true},
		{"not convert", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.convert, result.Convert)
		})
	}
}
