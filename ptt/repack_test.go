package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestRepack(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		repack bool
	}{
		{"release is repack", "Silicon Valley S04E03 REPACK HDTV x264-SVA", true},
		{"release is rerip", "Expedition Unknown S03E14 Corsicas Nazi Treasure RERIP 720p HDTV x264-W4F", true},
		{"not repack", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.repack, result.Repack)
		})
	}
}
