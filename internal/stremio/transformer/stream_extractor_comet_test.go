package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamExtractorCometTorrent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:        "[TORRENT🧲] Comet 1080p",
				Description: "Deadpool 2016 BluRay 1080p DTS-ES AC3 x264-3Li.mkv\n💿 BluRay|avc|DTS Lossy|AC3|3Li\n💾 7.29 GB 🔎 DMM\n🇪🇸",
				InfoHash:    "c359566eed1264fbe0482aae479cbe51c966d468",
				FileIndex:   0,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "comet|c359566eed1264fbe0482aae479cbe51c966d468",
					VideoSize:  7826122416,
					Filename:   "Deadpool 2016 BluRay 1080p DTS-ES AC3 x264-3Li.mkv",
				},
			}, StreamExtractorResult{
				Hash:   "c359566eed1264fbe0482aae479cbe51c966d468",
				TTitle: "Deadpool 2016 BluRay 1080p DTS-ES AC3 x264-3Li.mkv",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"es"},
					Quality:    "BluRay",
					Resolution: "1080p",
					Site:       "DMM",
					Size:       "7.29 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Comet",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Deadpool 2016 BluRay 1080p DTS-ES AC3 x264-3Li.mkv",
					Size: "7.3 GB",
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorComet.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Description
			assert.Equal(t, &tc.result, data)
		})
	}
}

func TestStreamExtractorCometDebrid(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:        "[PM⚡] Comet 2160p",
				Description: "Deadpool (2016) 2160p [4K] BluRay SDR [HINDI-ENG-5.1] 10bit HEVC - PeruGuy.mkv\n💿 BluRay|SDR|hevc|AC3|5.1|10bit|PeruGuy\n💾 8.14 GB 🔎 MediaFusion|Knightcrawler\n🇬🇧/🇮🇳",
				URL:         "https://comet.elfhosted.com/xxxxxxx/playback/74315dd5d8a0a4e2b229914ad729887acedc396f/0/deadpool/n/n/Deadpool (2016) 2160p [4K] BluRay SDR [HINDI-ENG-5.1] 10bit HEVC - PeruGuy.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "comet|74315dd5d8a0a4e2b229914ad729887acedc396f",
					VideoSize:  8734893087,
					Filename:   "Deadpool (2016) 2160p [4K] BluRay SDR [HINDI-ENG-5.1] 10bit HEVC - PeruGuy.mkv",
				},
			}, StreamExtractorResult{
				Hash:   "74315dd5d8a0a4e2b229914ad729887acedc396f",
				TTitle: "Deadpool (2016) 2160p [4K] BluRay SDR [HINDI-ENG-5.1] 10bit HEVC - PeruGuy.mkv",
				Result: &ptt.Result{
					Codec:      "HEVC",
					Languages:  []string{"en", "hi"},
					Quality:    "BluRay",
					Resolution: "4K",
					Site:       "MediaFusion|Knightcrawler",
					Size:       "8.14 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Comet",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Deadpool (2016) 2160p [4K] BluRay SDR [HINDI-ENG-5.1] 10bit HEVC - PeruGuy.mkv",
					Size: "8.1 GB",
				},
				Store: StreamExtractorResultStore{
					Name:     "premiumize",
					Code:     "PM",
					IsCached: true,
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi",
			"series", stremio.Stream{
				Name:        "[PM⬇️] Comet 2160p",
				Description: "Black Snow S01 MULTi HDR 2160p WEB H265-BraD\n💿 WEB|HDR|hevc|BraD\n👤 0 💾 27.93 GB 🔎 MediaFusion|Zilean DMM",
				URL:         "https://comet.elfhosted.com/xxxxxxx/playback/e3cd01e301c4bd3bc7888b189ba6a3b8e0ac152d/n/black snow/1/2/Black Snow S01 MULTi HDR 2160p WEB H265-BraD",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "comet|e3cd01e301c4bd3bc7888b189ba6a3b8e0ac152d",
					VideoSize:  29991714406,
					Filename:   "Black Snow S01 MULTi HDR 2160p WEB H265-BraD",
				},
			}, StreamExtractorResult{
				Hash:   "e3cd01e301c4bd3bc7888b189ba6a3b8e0ac152d",
				TTitle: "Black Snow S01 MULTi HDR 2160p WEB H265-BraD",
				Result: &ptt.Result{
					Codec:      "HEVC",
					Episodes:   []int{2},
					Quality:    "WEB",
					Resolution: "4k",
					Seasons:    []int{1},
					Site:       "MediaFusion|Zilean DMM",
					Size:       "27.93 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Comet",
				},
				File: StreamExtractorResultFile{
					Idx:  -1,
					Name: "Black Snow S01 MULTi HDR 2160p WEB H265-BraD",
					Size: "28 GB",
				},
				Store: StreamExtractorResultStore{
					Name: "premiumize",
					Code: "PM",
				},
				Episode: 2,
				Season:  1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorComet.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Description
			assert.Equal(t, &tc.result, data)
		})
	}
}
