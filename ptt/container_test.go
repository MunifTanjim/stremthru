package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestContainer(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ttitle    string
		container string
	}{
		{"mkv", "Kevin Hart What Now (2016) 1080p BluRay x265 6ch -Dtech mkv", "mkv"},
		{"mp4", "The Gorburger Show S01E05 AAC MP4-Mobile", "mp4"},
		{"avi", "[req]Night of the Lepus (1972) DVDRip XviD avi", "avi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.container, result.Container)
		})
	}
}
