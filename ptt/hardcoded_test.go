package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHardcoded(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		dubbed bool
	}{
		{"is hardcoded", "Ghost In The Shell 2017 1080p HC HDRip X264 AC3-EVO", true},
		{"not hardcoded", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.dubbed, result.Hardcoded)
		})
	}
}
