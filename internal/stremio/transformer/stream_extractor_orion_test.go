package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamExtractorOrionTorrent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:     "🪐 ORION 📺 4K",
				Title:    "Deadpool & Wolverine [2024] 2160p UHD BDRip DV HDR10 x265 TrueHD Atmos 7 1 Kira [SEV] mkv\n💾26.6 GB 👤0 🎥h265 🔊7.1\n👂EN ☁️torlock",
				InfoHash: "f0b4ba9bf31960b8920e9335ab07037f295bbf67",
			}, StreamExtractorResult{
				Hash:   "f0b4ba9bf31960b8920e9335ab07037f295bbf67",
				TTitle: "Deadpool & Wolverine [2024] 2160p UHD BDRip DV HDR10 x265 TrueHD Atmos 7 1 Kira [SEV] mkv",
				Result: &ptt.Result{
					Channels:   []string{"7.1"},
					Codec:      "HEVC",
					Languages:  []string{"EN"},
					Resolution: "4K",
					Site:       "torlock",
					Size:       "26.6 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "ORION",
				},
				File: StreamExtractorResultFile{
					Idx: 0,
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorOrion.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}

func TestStreamExtractorOrionDebrid(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:  "🚀 ORION\n[Offcloud]",
				Title: "Deadpool.&.Wolverine.(2024).(2160p.BluRay.x265.10bit.DV.HDR.TrueHD.Atmos.7.1.English. French. Spanish.r00t) [QxR]\n📺4K 💾22.8 GB 🎥h265 🔊7.1\n👂EN FR ES ☁️1337x",
				URL:   "https://orionoid.com/stream/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				BehaviorHints: &stremio.StreamBehaviorHints{
					NotWebReady: true,
				},
			}, StreamExtractorResult{
				TTitle: "Deadpool.&.Wolverine.(2024).(2160p.BluRay.x265.10bit.DV.HDR.TrueHD.Atmos.7.1.English. French. Spanish.r00t) [QxR]",
				Result: &ptt.Result{
					Channels:   []string{"7.1"},
					Codec:      "HEVC",
					Languages:  []string{"EN", "FR", "ES"},
					Resolution: "4K",
					Site:       "1337x",
					Size:       "22.8 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "ORION",
				},
				File: StreamExtractorResultFile{
					Idx: -1,
				},
				Store: StreamExtractorResultStore{
					Name:     "offcloud",
					Code:     "OC",
					IsCached: true,
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi",
			"series", stremio.Stream{
				Name:  "🚀 ORION\n [Offcloud]",
				Title: "Reacher S03E07 L A Story 2160p AMZN WEB-DL DDP5 1 Atmos DV HDR H 265-FLUX mkv\n📺4K 💾5.31 GB 🎥h265 🔊5.1\n👂EN ☁️torrentscsv",
				URL:   "https://orionoid.com/stream/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup:  "orion-4K-Offcloud",
					NotWebReady: true,
				},
			}, StreamExtractorResult{
				TTitle: "Reacher S03E07 L A Story 2160p AMZN WEB-DL DDP5 1 Atmos DV HDR H 265-FLUX mkv",
				Result: &ptt.Result{
					Channels:   []string{"5.1"},
					Codec:      "HEVC",
					Languages:  []string{"EN"},
					Resolution: "4k",
					Site:       "torrentscsv",
					Size:       "5.31 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "ORION",
				},
				File: StreamExtractorResultFile{
					Idx: -1,
				},
				Store: StreamExtractorResultStore{
					Name:     "offcloud",
					Code:     "OC",
					IsCached: true,
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorOrion.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}
