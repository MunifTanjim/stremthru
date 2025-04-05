package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestSize(t *testing.T) {
	for _, tc := range []struct {
		ttitle string
		size   string
	}{
		{"www.1TamilBlasters.lat - Thuritham (2023) [Tamil - 2K QHD AVC UNTOUCHED - x264 - AAC - 3.4GB - ESub].mkv", "3.4GB"},
		{"www.1TamilMV.world - Raja Vikramarka (2024) Tamil HQ HDRip - 400MB - x264 - AAC - ESub.mkv", "400MB"},
		{"www.1TamilMV.cz - Maharaja (2024) TRUE WEB-DL - 1080p HQ - AVC - (DD+5.1 - 640Kbps) [Tam + Tel + Hin + Mal + Kan] - 8.4GB - ESub.mkv", "8.4GB"},
		{"The.Walking.Dead.S06E07.SUBFRENCH.HDTV.x264-AMB3R.mkv", ""},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.size, result.Size)
		})
	}
}
