package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamExtractorTorrentioTorrent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:      "Torrentio\n4k DV",
				Title:     "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR\n👤 47 💾 40.33 GB ⚙️ TorrentGalaxy",
				InfoHash:  "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				FileIndex: 0,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|4k|BluRay REMUX|hevc|DV",
					Filename:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				},
			}, StreamExtractorResult{
				Hash:   "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				TTitle: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR",
				Result: &ptt.Result{
					Codec:      "HEVC",
					HDR:        []string{"DV"},
					Quality:    "BluRay REMUX",
					Resolution: "4k",
					Site:       "TorrentGalaxy",
					Size:       "40.33 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"single - no resolution",
			"movie", stremio.Stream{
				Name:      "Torrentio\nDVDRip",
				Title:     "A Simple Favor 2018 DVDRip x264 ESub [MW]\n👤 5 💾 864.57 MB ⚙️ ThePirateBay",
				InfoHash:  "387ccd318d583405bbadcec55b9b05029645dd1d",
				FileIndex: 0,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|DVDRip|x264",
					Filename:   "A Simple Favor 2018 DVDRip x264 ESub [MW].mkv",
				},
			}, StreamExtractorResult{
				Hash:   "387ccd318d583405bbadcec55b9b05029645dd1d",
				TTitle: "A Simple Favor 2018 DVDRip x264 ESub [MW]",
				Result: &ptt.Result{
					Codec:   "AVC",
					Quality: "DVDRip",
					Site:    "ThePirateBay",
					Size:    "864.57 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "A Simple Favor 2018 DVDRip x264 ESub [MW].mkv",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi w/ behaviorHints.filename",
			"movie", stremio.Stream{
				Name:      "Torrentio\n1080p",
				Title:     "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				InfoHash:  "a6a80257d62e53e55c877a7067ea5055129b462c",
				FileIndex: 89,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
					Filename:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
			}, StreamExtractorResult{
				Hash:   "a6a80257d62e53e55c877a7067ea5055129b462c",
				TTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"msub", "en", "ru", "fi"},
					Resolution: "1080p",
					Site:       "TorrentGalaxy",
					Size:       "3.65 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  89,
					Name: "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi w/o behaviorHints.filename",
			"movie", stremio.Stream{
				Name:      "Torrentio\n1080p",
				Title:     "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				InfoHash:  "a6a80257d62e53e55c877a7067ea5055129b462c",
				FileIndex: 89,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
				},
			}, StreamExtractorResult{
				Hash:   "a6a80257d62e53e55c877a7067ea5055129b462c",
				TTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"msub", "en", "ru", "fi"},
					Resolution: "1080p",
					Site:       "TorrentGalaxy",
					Size:       "3.65 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  89,
					Name: "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"missing filename",
			"movie", stremio.Stream{
				Name:     "Torrentio\n720p",
				Title:    "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub\n👤 3 💾 934.8 MB ⚙️ 1337x\nDual Audio / 🇬🇧 / 🇮🇳",
				InfoHash: "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|720p|BluRay|x264",
				},
			}, StreamExtractorResult{
				Hash:   "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				TTitle: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"daud", "en", "hi"},
					Quality:    "BluRay",
					Resolution: "720p",
					Site:       "1337x",
					Size:       "934.8 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorTorrentio.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}

func TestStreamExtractorTorrentioDebrid(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:  "[RD+] Torrentio\n4k DV | HDR",
				Title: "Deadpool [2016] 2160p Hybrid UHD BDRip DV HDR10 x265 TrueHD Atmos 7.1 Kira [SEV]\n👤 16 💾 22.42 GB ⚙️ 1337x",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/c35ec8ad9f613d73782a898bece969efd6b98e0c/null/0/Deadpool%20%5B2016%5D%202160p%20Hybrid%20UHD%20BDRip%20DV%20HDR10%20x265%20TrueHD%20Atmos%207.1%20Kira%20%5BSEV%5D.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|4k|BDRip|x265|10bit|DV|HDR",
					Filename:   "Deadpool [2016] 2160p Hybrid UHD BDRip DV HDR10 x265 TrueHD Atmos 7.1 Kira [SEV].mkv",
				},
			}, StreamExtractorResult{
				Hash:   "c35ec8ad9f613d73782a898bece969efd6b98e0c",
				TTitle: "Deadpool [2016] 2160p Hybrid UHD BDRip DV HDR10 x265 TrueHD Atmos 7.1 Kira [SEV]",
				Result: &ptt.Result{
					BitDepth:   "10bit",
					Codec:      "HEVC",
					HDR:        []string{"DV", "HDR"},
					Quality:    "BDRip",
					Resolution: "4k",
					Site:       "1337x",
					Size:       "22.42 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Deadpool [2016] 2160p Hybrid UHD BDRip DV HDR10 x265 TrueHD Atmos 7.1 Kira [SEV].mkv",
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
			"multi",
			"movie", stremio.Stream{
				Name:  "[RD+] Torrentio\n1080p",
				Title: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/a6a80257d62e53e55c877a7067ea5055129b462c/null/89/08%20Deadpool%20-%20Action%202016%20Eng%20Rus%20Multi-Subs%201080p%20%5BH264-mp4%5D.mp4",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
					Filename:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
			}, StreamExtractorResult{
				Hash:   "a6a80257d62e53e55c877a7067ea5055129b462c",
				TTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"msub", "en", "ru", "fi"},
					Resolution: "1080p",
					Site:       "TorrentGalaxy",
					Size:       "3.65 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx:  89,
					Name: "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
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
			"missing filename",
			"movie", stremio.Stream{
				Name:  "[RD download] Torrentio\n720p",
				Title: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub\n👤 3 💾 934.8 MB ⚙️ 1337x\nDual Audio / 🇬🇧 / 🇮🇳",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/f5d0ab292f5a244a4b38efac9ae1f8d311179588/null/undefined/Deadpool%20(2016)%20720p%20BluRay%20x264%20%5BDual%20Audio%5D%20%5BHindi%20(Line%20Audio)%20-%20English%5D%20ESubs%20-%20Downloadhub",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|720p|BluRay|x264",
				},
			}, StreamExtractorResult{
				Hash:   "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				TTitle: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub",
				Result: &ptt.Result{
					Codec:      "AVC",
					Languages:  []string{"daud", "en", "hi"},
					Quality:    "BluRay",
					Resolution: "720p",
					Site:       "1337x",
					Size:       "934.8 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Torrentio",
				},
				File: StreamExtractorResultFile{
					Idx: -1,
				},
				Store: StreamExtractorResultStore{
					Name: "realdebrid",
					Code: "RD",
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorTorrentio.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}
