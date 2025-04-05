package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		region string
	}{
		{"the R5 region", "Welcome to New York 2014 R5 XviD AC3-SUPERFAST", "R5"},
		{"not region in the title", "[Coalgirls]_Code_Geass_R2_06_(1920x1080_Blu-ray_FLAC)_[F8C7FE25].mkv", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.region, result.Region)
		})
	}

	for _, tc := range []struct {
		ttitle string
		region string
	}{
		{"[JySzE] Naruto [v2] [R2J] [VFR] [Dual Audio] [Complete] [Extras] [x264]", "R2J"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.region, result.Region)
		})
	}
}
