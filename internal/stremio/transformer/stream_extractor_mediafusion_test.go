package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamExtractorMediaFusionTorrent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"w/ tt - single",
			"movie", stremio.Stream{
				Name:        "MediaFusion | ElfHosted P2P 2160P ⏳",
				Description: "📂 Deadpool 2016 4K HDR DV 2160p BDRemux Ita Eng x265-NAHOM\n💾 35.09 GB\n🌐 English + Italian\n🔗 Torlock",
				InfoHash:    "a3d11f4d97121a79f3e94b18a43e5b3e2f1853e1",
				FileIndex:   2,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-🎨 DV|HDR 📺 BluRay REMUX 🎞️ hevc-2160P",
					Filename:   "Deadpool.2016.4K.HDR.DV.2160p.BDRemux Ita Eng x265-NAHOM.mkv",
					VideoSize:  37682583137,
				},
			}, StreamExtractorResult{
				Hash:   "a3d11f4d97121a79f3e94b18a43e5b3e2f1853e1",
				TTitle: "Deadpool 2016 4K HDR DV 2160p BDRemux Ita Eng x265-NAHOM",
				Result: &ptt.Result{
					Codec:      "HEVC",
					HDR:        []string{"DV", "HDR"},
					Quality:    "BluRay REMUX",
					Resolution: "4k",
					Site:       "Torlock",
					Size:       "35.09 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  2,
					Name: "Deadpool.2016.4K.HDR.DV.2160p.BDRemux Ita Eng x265-NAHOM.mkv",
					Size: "35 GB",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"w/ tt - invalid behaviorHints.filename",
			"movie", stremio.Stream{
				Name:        "MediaFusion | ElfHosted P2P 720P ⏳",
				Description: "📂 Deadpool 2016 x264 720p BluRay Eng Subs Dual Audio Hindi 5 1 English 5 1 Downloadhub\n💾 983.4 MB 👤 30\n🌐 English + Hindi\n🔗 TheRARBG",
				InfoHash:    "2decf5e42220711acf7a2515ed14ee78f13413fe",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-📺 BluRay 🎞️ avc-720P",
					Filename:   "Deadpool 2016 x264 720p BluRay Eng Subs Dual Audio Hindi 5 1 English 5 1 Downloadhub",
					VideoSize:  1031169664,
				},
			}, StreamExtractorResult{
				Hash:   "2decf5e42220711acf7a2515ed14ee78f13413fe",
				TTitle: "Deadpool 2016 x264 720p BluRay Eng Subs Dual Audio Hindi 5 1 English 5 1 Downloadhub",
				Result: &ptt.Result{
					Codec:      "AVC",
					Quality:    "BluRay",
					Resolution: "720p",
					Site:       "TheRARBG",
					Size:       "983.4 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Deadpool 2016 x264 720p BluRay Eng Subs Dual Audio Hindi 5 1 English 5 1 Downloadhub",
					Size: "983 MB",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"w/ tt - multi",
			"series", stremio.Stream{
				Name:        "MediaFusion | ElfHosted P2P 1080P ⏳",
				Description: "📂 Sherlock S01-S04 + Extras Complete BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas [Telly] ┈➤ Sherlock S01 E01 BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas mkv\n💾 2.43 GB / 💾 33.48 GB\n🌐 English\n🔗 Zilean DMM",
				InfoHash:    "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				FileIndex:   1,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-📺 BluRay 🎞️ avc 🎵 Dolby Digital-1080P",
					Filename:   "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					VideoSize:  2608894683,
				},
			}, StreamExtractorResult{
				Hash:   "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				TTitle: "Sherlock S01-S04 + Extras Complete BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas [Telly]",
				Result: &ptt.Result{
					Codec:      "AVC",
					Quality:    "BluRay",
					Resolution: "1080p",
					Site:       "Zilean DMM",
					Size:       "33.48 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  1,
					Name: "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					Size: "2.4 GB",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"w/o tt - multi",
			"series", stremio.Stream{
				Name:        "MediaFusion | ElfHosted P2P 1080P ⏳",
				Description: "📺 BluRay 🎞️ avc 🎵 Dolby Digital\n💾 2.43 GB / 💾 33.48 GB\n🌐 English\n🔗 Zilean DMM",
				InfoHash:    "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				FileIndex:   1,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-📺 BluRay 🎞️ avc 🎵 Dolby Digital-1080P",
					Filename:   "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					VideoSize:  2608894683,
				},
			}, StreamExtractorResult{
				Hash:   "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				TTitle: "",
				Result: &ptt.Result{
					Codec:      "AVC",
					Quality:    "BluRay",
					Resolution: "1080p",
					Site:       "Zilean DMM",
					Size:       "33.48 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  1,
					Name: "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					Size: "2.4 GB",
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorMediaFusion.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Description
			assert.Equal(t, &tc.result, data)
		})
	}
}

func TestStreamExtractorMediaFusionDebrid(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"w/ tt - single",
			"movie", stremio.Stream{
				Name:        "MediaFusion | ElfHosted RD 2160P ⚡️",
				Description: "📂 Deadpool 2016 4K HDR DV 2160p BDRemux Ita Eng x265-NAHOM\n💾 35.09 GB\n🌐 English + Italian\n🔗 Torlock",
				URL:         "https://mediafusion.elfhosted.com/streaming_provider/D-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/stream/a3d11f4d97121a79f3e94b18a43e5b3e2f1853e1/Deadpool.2016.4K.HDR.DV.2160p.BDRemux%20Ita%20Eng%20x265-NAHOM.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-🎨 DV|HDR 📺 BluRay REMUX 🎞️ hevc-2160P",
					Filename:   "Deadpool.2016.4K.HDR.DV.2160p.BDRemux Ita Eng x265-NAHOM.mkv",
					VideoSize:  37682583137,
				},
			}, StreamExtractorResult{
				Hash:   "a3d11f4d97121a79f3e94b18a43e5b3e2f1853e1",
				TTitle: "Deadpool 2016 4K HDR DV 2160p BDRemux Ita Eng x265-NAHOM",
				Result: &ptt.Result{
					Codec:      "HEVC",
					HDR:        []string{"DV", "HDR"},
					Quality:    "BluRay REMUX",
					Resolution: "4k",
					Site:       "Torlock",
					Size:       "35.09 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  -1,
					Name: "Deadpool.2016.4K.HDR.DV.2160p.BDRemux Ita Eng x265-NAHOM.mkv",
					Size: "35 GB",
				},
				Store: StreamExtractorResultStore{
					Name:     "realdebrid",
					Code:     "RD",
					IsCached: true,
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"w/ tt - multi",
			"series", stremio.Stream{
				Name:        "MediaFusion | ElfHosted RD 1080P ⚡️",
				Description: "📂 Sherlock S01-S04 + Extras Complete BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas [Telly] ┈➤ Sherlock S01 E01 BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas mkv\n💾 2.43 GB / 💾 33.48 GB\n🌐 English\n🔗 Zilean DMM",
				URL:         "https://mediafusion.elfhosted.com/streaming_provider/D-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/stream/ce146cce125215f5e6615d2375ffa6a881c8eedd/1/1/Sherlock%20S01%20E01%20BluRay%201080p%20English%20DD%205.1%20x264%20ESub%20-%20mkvCinemas.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-📺 BluRay 🎞️ avc 🎵 Dolby Digital-1080P",
					Filename:   "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					VideoSize:  2608894683,
				},
			}, StreamExtractorResult{
				Hash:   "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				TTitle: "Sherlock S01-S04 + Extras Complete BluRay 1080p English DD 5 1 x264 ESub - mkvCinemas [Telly]",
				Result: &ptt.Result{
					Codec:      "AVC",
					Episodes:   []int{1},
					Quality:    "BluRay",
					Resolution: "1080p",
					Seasons:    []int{1},
					Site:       "Zilean DMM",
					Size:       "33.48 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  -1,
					Name: "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					Size: "2.4 GB",
				},
				Store: StreamExtractorResultStore{
					Name:     "realdebrid",
					Code:     "RD",
					IsCached: true,
				},
				Episode: 1,
				Season:  1,
			},
		},
		{
			"w/o tt - multi",
			"series", stremio.Stream{
				Name:        "MediaFusion | ElfHosted RD 1080P ⚡️",
				Description: "📺 BluRay 🎞️ avc 🎵 Dolby Digital\n💾 2.43 GB / 💾 33.48 GB\n🌐 English\n🔗 Zilean DMM",
				URL:         "https://mediafusion.elfhosted.com/streaming_provider/D-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/stream/ce146cce125215f5e6615d2375ffa6a881c8eedd/1/1/Sherlock%20S01%20E01%20BluRay%201080p%20English%20DD%205.1%20x264%20ESub%20-%20mkvCinemas.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "MediaFusion-|-ElfHosted-📺 BluRay 🎞️ avc 🎵 Dolby Digital-1080P",
					Filename:   "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					VideoSize:  2608894683,
				},
			}, StreamExtractorResult{
				Hash:   "ce146cce125215f5e6615d2375ffa6a881c8eedd",
				TTitle: "",
				Result: &ptt.Result{
					Codec:      "AVC",
					Episodes:   []int{1},
					Quality:    "BluRay",
					Resolution: "1080p",
					Seasons:    []int{1},
					Site:       "Zilean DMM",
					Size:       "33.48 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "MediaFusion | ElfHosted",
				},
				File: StreamExtractorResultFile{
					Idx:  -1,
					Name: "Sherlock S01 E01 BluRay 1080p English DD 5.1 x264 ESub - mkvCinemas.mkv",
					Size: "2.4 GB",
				},
				Store: StreamExtractorResultStore{
					Name:     "realdebrid",
					Code:     "RD",
					IsCached: true,
				},
				Episode: 1,
				Season:  1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorMediaFusion.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Description
			assert.Equal(t, &tc.result, data)
		})
	}
}
