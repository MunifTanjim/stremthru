package torrent_info

import (
	"testing"

	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestExtractorTorrentioTorrent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hostname string
		sid      string
		stream   stremio.Stream
		data     TorrentInfoInsertData
	}{
		{
			"single",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:      "Torrentio\n4k DV",
				Title:     "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR\n👤 47 💾 40.33 GB ⚙️ TorrentGalaxy",
				InfoHash:  "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				FileIndex: 0,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|4k|BluRay REMUX|hevc|DV",
					Filename:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				},
			}, TorrentInfoInsertData{
				Hash:         "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				TorrentTitle: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR",
				Seeders:      47,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
						Idx:    0,
						Size:   43304007761,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"multi w/ behaviorHints.filename",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:      "Torrentio\n1080p",
				Title:     "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				InfoHash:  "a6a80257d62e53e55c877a7067ea5055129b462c",
				FileIndex: 89,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
					Filename:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
			}, TorrentInfoInsertData{
				Hash:         "a6a80257d62e53e55c877a7067ea5055129b462c",
				TorrentTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Seeders:      20,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
						Idx:    89,
						Size:   3919157657,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"multi w/o behaviorHints.filename",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:      "Torrentio\n1080p",
				Title:     "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				InfoHash:  "a6a80257d62e53e55c877a7067ea5055129b462c",
				FileIndex: 89,
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
				},
			}, TorrentInfoInsertData{
				Hash:         "a6a80257d62e53e55c877a7067ea5055129b462c",
				TorrentTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Seeders:      20,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
						Idx:    89,
						Size:   3919157657,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"missing filename",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:     "Torrentio\n720p",
				Title:    "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub\n👤 3 💾 934.8 MB ⚙️ 1337x\nDual Audio / 🇬🇧 / 🇮🇳",
				InfoHash: "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|720p|BluRay|x264",
				},
			}, TorrentInfoInsertData{
				Hash:         "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				TorrentTitle: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub",
				Seeders:      3,
				Size:         -1,
				Source:       "tio",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := ExtractCreateDataFromStream(tc.hostname, tc.sid, &tc.stream)
			assert.Equal(t, &tc.data, data)
		})
	}
}

func TestExtractorTorrentioDebrid(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hostname string
		sid      string
		stream   stremio.Stream
		data     TorrentInfoInsertData
	}{
		{
			"single",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:  "[RD+] Torrentio\n4k DV",
				Title: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR\n👤 47 💾 40.33 GB ⚙️ TorrentGalaxy",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72/null/0/Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|4k|BluRay REMUX|hevc|DV",
					Filename:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				},
			}, TorrentInfoInsertData{
				Hash:         "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				TorrentTitle: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR",
				Seeders:      47,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
						Idx:    0,
						Size:   43304007761,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"single - untrusted fileidx",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:  "[AD download] Torrentio\n4k DV",
				Title: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR\n👤 47 💾 40.33 GB ⚙️ TorrentGalaxy",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72/null/0/Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|4k|BluRay REMUX|hevc|DV",
					Filename:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
				},
			}, TorrentInfoInsertData{
				Hash:         "e4f5d7a2f3dd6b7b1826bd77e316b6b5ba31eb72",
				TorrentTitle: "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR",
				Seeders:      47,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "Deadpool.2016.UHD.BluRay.2160p.TrueHD.Atmos.7.1.DV.HEVC.HYBRiD.REMUX-FraMeSToR.mkv",
						Idx:    -1,
						Size:   43304007761,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"multi",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:  "[RD+] Torrentio\n1080p",
				Title: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]\nX-Men Complete 13 Movie Collection 2000-2020/08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4\n👤 20 💾 3.65 GB ⚙️ TorrentGalaxy\nMulti Subs / 🇬🇧 / 🇷🇺 / 🇫🇮",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/a6a80257d62e53e55c877a7067ea5055129b462c/null/89/08%20Deadpool%20-%20Action%202016%20Eng%20Rus%20Multi-Subs%201080p%20%5BH264-mp4%5D.mp4",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|1080p|h264",
					Filename:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
				},
			}, TorrentInfoInsertData{
				Hash:         "a6a80257d62e53e55c877a7067ea5055129b462c",
				TorrentTitle: "X-Men Complete 13 Movie Collection Sci-Fi 2000 - 2020 Eng Rus Multi-Subs 1080p [H264-mp4]",
				Seeders:      20,
				Size:         -1,
				Source:       "tio",
				Files: []TorrentInfoInsertDataFile{
					{
						Name:   "08 Deadpool - Action 2016 Eng Rus Multi-Subs 1080p [H264-mp4].mp4",
						Idx:    89,
						Size:   3919157657,
						SId:    "tt1431045",
						Source: "",
					},
				},
			},
		},
		{
			"missing filename",
			"torrentio.strem.fun", "tt1431045", stremio.Stream{
				Name:  "[RD download] Torrentio\n720p",
				Title: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub\n👤 3 💾 934.8 MB ⚙️ 1337x\nDual Audio / 🇬🇧 / 🇮🇳",
				URL:   "https://torrentio.strem.fun/realdebrid/xxxxxxxxxxxxxxxx/f5d0ab292f5a244a4b38efac9ae1f8d311179588/null/undefined/Deadpool%20(2016)%20720p%20BluRay%20x264%20%5BDual%20Audio%5D%20%5BHindi%20(Line%20Audio)%20-%20English%5D%20ESubs%20-%20Downloadhub",
				BehaviorHints: &stremio.StreamBehaviorHints{
					BingeGroup: "torrentio|720p|BluRay|x264",
				},
			}, TorrentInfoInsertData{
				Hash:         "f5d0ab292f5a244a4b38efac9ae1f8d311179588",
				TorrentTitle: "Deadpool (2016) 720p BluRay x264 [Dual Audio] [Hindi (Line Audio) - English] ESubs - Downloadhub",
				Seeders:      3,
				Size:         -1,
				Source:       "tio",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := ExtractCreateDataFromStream(tc.hostname, tc.sid, &tc.stream)
			assert.Equal(t, &tc.data, data)
		})
	}
}
