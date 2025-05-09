package stremio_transformer

import (
	"testing"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamExtractorPeerflixTorrent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:      "Peerflix 🇪🇸 4K",
				Title:     "Deadpool [4K UHD 2160p HEVC X265][DTS 5.1 Castellano DTS 5.1-Ingles+Subs][ES-EN]\n  👤 1 💾 20.38 GB 🌐 Peerflix",
				InfoHash:  "31e585c52409430634139d915aeb4ea0f74be287",
				FileIndex: 0,
			}, StreamExtractorResult{
				Hash:   "31e585c52409430634139d915aeb4ea0f74be287",
				TTitle: "Deadpool [4K UHD 2160p HEVC X265][DTS 5.1 Castellano DTS 5.1-Ingles+Subs][ES-EN]",
				Result: &ptt.Result{
					Resolution: "4K",
					Site:       "Peerflix",
					Size:       "20.38 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Peerflix",
				},
				File: StreamExtractorResultFile{
					Idx: 0,
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi",
			"series", stremio.Stream{
				Name:      "Peerflix 🇬🇧 720p",
				Title:     "Black Snow 2023 S01-S02 720p WEB-DL HEVC x265 BONE\nBlack Snow 2023 S01E02 720p WEB-DL HEVC x265 BONE.mkv\n  👤 89 💾 410.26 MB 🌐 Peerflix",
				InfoHash:  "fa7dfc675f81e573f5c99e776974df44e887ade5",
				FileIndex: 5,
			}, StreamExtractorResult{
				Hash:   "fa7dfc675f81e573f5c99e776974df44e887ade5",
				TTitle: "Black Snow 2023 S01-S02 720p WEB-DL HEVC x265 BONE",
				Result: &ptt.Result{
					Resolution: "720p",
					Site:       "Peerflix",
					Size:       "410.26 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Peerflix",
				},
				File: StreamExtractorResultFile{
					Idx:  5,
					Name: "Black Snow 2023 S01E02 720p WEB-DL HEVC x265 BONE.mkv",
				},
				Episode: -1,
				Season:  -1,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorPeerflix.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}

func TestStreamExtractorPeerflixDebrid(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sType  string
		stream stremio.Stream
		result StreamExtractorResult
	}{
		{
			"single",
			"movie", stremio.Stream{
				Name:  "[RD download] Peerflix 🇪🇸 4K",
				Title: "Deadpool [4K UHD 2160p HEVC X265][DTS 5.1 Castellano DTS 5.1-Ingles+Subs][ES-EN]\n  👤 1 💾 20.38 GB 🌐 Peerflix",
				URL:   "https://peerflix-addon.onrender.com/realdebrid/xxxxxxx/31e585c52409430634139d915aeb4ea0f74be287/null/0/Deadpool%20%5B4K%20UHD%202160p%20HEVC%20X265%5D%5BDTS%205.1%20Castellano%20DTS%205.1-Ingles%2BSubs%5D%5BES-EN%5D",
			}, StreamExtractorResult{
				Hash:   "31e585c52409430634139d915aeb4ea0f74be287",
				TTitle: "Deadpool [4K UHD 2160p HEVC X265][DTS 5.1 Castellano DTS 5.1-Ingles+Subs][ES-EN]",
				Result: &ptt.Result{
					Resolution: "4k",
					Site:       "Peerflix",
					Size:       "20.38 GB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Peerflix",
				},
				File: StreamExtractorResultFile{
					Idx: 0,
				},
				Store: StreamExtractorResultStore{
					Name: "realdebrid",
					Code: "RD",
				},
				Episode: -1,
				Season:  -1,
			},
		},
		{
			"multi",
			"series", stremio.Stream{
				Name:  "[RD+] Peerflix 🇬🇧 720p",
				Title: "Black Snow 2023 S01-S02 720p WEB-DL HEVC x265 BONE\nBlack Snow 2023 S01E01 720p WEB-DL HEVC x265 BONE.mkv\n  👤 89 💾 381.53 MB 🌐 Peerflix",
				URL:   "https://peerflix-addon.onrender.com/realdebrid/xxxxxxx/fa7dfc675f81e573f5c99e776974df44e887ade5/null/0/Black%20Snow%202023%20S01E01%20720p%20WEB-DL%20HEVC%20x265%20BONE.mkv",
			}, StreamExtractorResult{
				Hash:   "fa7dfc675f81e573f5c99e776974df44e887ade5",
				TTitle: "Black Snow 2023 S01-S02 720p WEB-DL HEVC x265 BONE",
				Result: &ptt.Result{
					Resolution: "720p",
					Site:       "Peerflix",
					Size:       "381.53 MB",
				},
				Addon: StreamExtractorResultAddon{
					Name: "Peerflix",
				},
				File: StreamExtractorResultFile{
					Idx:  0,
					Name: "Black Snow 2023 S01E01 720p WEB-DL HEVC x265 BONE.mkv",
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := StreamExtractorPeerflix.Parse(&tc.stream, tc.sType)
			tc.result.Category = tc.sType
			tc.result.Result.Normalize()
			tc.result.Raw.Name = tc.stream.Name
			tc.result.Raw.Description = tc.stream.Title
			assert.Equal(t, &tc.result, data)
		})
	}
}
