package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/stretchr/testify/assert"
)

func TestFallbackStreamExtractor(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  *StreamExtractorResult
		result *StreamExtractorResult
	}{
		{
			"skcz-torrents",
			&StreamExtractorResult{
				Raw: StreamExtractorResultRaw{
					Name:        "SK/CZ Torrents\n1080p",
					Description: "Hodina zmizení / Weapons (2025)(CZ)[1080p] = CSFD 76%\n👤 76 💾 2.3 GB 🗣 SK/CZ",
				},
				Result: &ptt.Result{},
			},
			&StreamExtractorResult{
				Result: &ptt.Result{
					Resolution: "1080p",
					Size:       "2.3 GB",
					Languages:  []string{"sk", "cz"},
				},
				Seeders: 76,
				TTitle:  "Hodina zmizení / Weapons (2025)(CZ)[1080p] = CSFD 76%",
			},
		},
		{
			"thepiratebay-plus",
			&StreamExtractorResult{
				Raw: StreamExtractorResultRaw{
					Name:        "TPB+",
					Description: "Interstellar.2014.UHD.BluRay.2160p.DTS-HD.MA.5.1.HEVC.REMUX-FraMeSToR\n📺 4k BluRay REMUX\n👤 99 💾 65.74 GB",
				},
				Result: &ptt.Result{},
			},
			&StreamExtractorResult{
				Result: &ptt.Result{
					Codec:      "HEVC",
					Quality:    "BluRay",
					Resolution: "2160p",
					Size:       "65.74 GB",
				},
				Seeders: 99,
				TTitle:  "Interstellar.2014.UHD.BluRay.2160p.DTS-HD.MA.5.1.HEVC.REMUX-FraMeSToR",
			},
		},
		{
			"brazuca-torrents",
			&StreamExtractorResult{
				Raw: StreamExtractorResultRaw{
					Name:        "Brazuca\n4k",
					Description: "Interstelar.2014.1080p.60fps.BluRay.H264.AC3.5.1.DUAL-RICKSZ\n👤 0 💾 24.42 GB ⚙️ OndeBaixa\nDual Audio / 🇺🇸 / 🇧🇷",
				},
				Result: &ptt.Result{},
			},
			&StreamExtractorResult{
				Result: &ptt.Result{
					Codec:      "H264",
					Languages:  []string{"en", "pt"},
					Quality:    "BluRay",
					Resolution: "4k",
					Size:       "24.42 GB",
				},
				TTitle: "Interstelar.2014.1080p.60fps.BluRay.H264.AC3.5.1.DUAL-RICKSZ",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := fallbackStreamExtractor(tc.input)
			tc.result.Raw = result.Raw
			assert.Equal(t, tc.result, result)
		})
	}
}
