package ptt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHDR(t *testing.T) {
	testCases := []struct {
		name   string
		ttitle string
		hdr    []string
	}{
		{"HDR", "The.Mandalorian.S01E06.4K.HDR.2160p 4.42GB", []string{"HDR"}},
		{"HDR10", "Spider-Man - Complete Movie Collection (2002-2022) 1080p.HEVC.HDR10.1920x800.x265. DTS-HD", []string{"HDR"}},
		{"HDR10Plus", "Bullet.Train.2022.2160p.AMZN.WEB-DL.x265.10bit.HDR10Plus.DDP5.1-SMURF", []string{"HDR10+"}},
		{"DV v1", "Belle (2021) 2160p 10bit 4KLight DOLBY VISION BluRay DDP 7.1 x265-QTZ", []string{"DV"}},
		{"DV v2", "Андор / Andor [01x01-03 из 12] (2022) WEB-DL-HEVC 2160p | 4K | Dolby Vision TV | NewComers, HDRezka Studio", []string{"DV"}},
		{"DV v3", "АBullet.Train.2022.2160p.WEB-DL.DDP5.1.DV.MKV.x265-NOGRP", []string{"DV"}},
		{"DV v4", "Bullet.Train.2022.2160p.WEB-DL.DoVi.DD5.1.HEVC-EVO[TGx]", []string{"DV"}},
		{"HDR/DV v1", "Спайдерхед / Spiderhead (2022) WEB-DL-HEVC 2160p | 4K | HDR | Dolby Vision Profile 8 | P | NewComers, Jaskier", []string{"DV", "HDR"}},
		{"HDR/DV v2", "House.of.the.Dragon.S01E07.2160p.10bit.HDR.DV.WEBRip.6CH.x265.HEVC-PSA", []string{"DV", "HDR"}},
		{"HDR/HDR10+/DV", "Флешбэк / Memory (2022) WEB-DL-HEVC 2160p | 4K | HDR | HDR10+ | Dolby Vision Profile 8 | Pazl Voice", []string{"DV", "HDR10+", "HDR"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.hdr, result.HDR)
			for _, hdr := range result.HDR {
				if strings.Contains(hdr, "10") {
					assert.Equal(t, "10bit", result.BitDepth)
				}
			}
		})
	}
}
