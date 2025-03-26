package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestSite(t *testing.T) {
	for _, tc := range []struct {
		ttitle string
		site   string
	}{
		{"The.Expanse.S05E02.1080p.AMZN.WEB.DDP5.1.x264-NTb[eztv.re].mp4", "eztv.re"},
		{"www.1TamilBlasters.lat - Thuritham (2023) [Tamil - 2K QHD AVC UNTOUCHED - x264 - AAC - 3.4GB - ESub].mkv", "www.1TamilBlasters.lat"},
		{"www.1TamilMV.world - Raja Vikramarka (2024) Tamil HQ HDRip - 400MB - x264 - AAC - ESub.mkv", "www.1TamilMV.world"},
		{"Anatomia De Grey - Temporada 19 [HDTV][Cap.1905][Castellano][www.AtomoHD.nu].avi", "www.AtomoHD.nu"},
		{"[HD-ELITE.NET] -  The.Art.Of.The.Steal.2014.DVDRip.XviD.Dual.Aud", "HD-ELITE.NET"},
		{"[ Torrent9.cz ] The.InBetween.S01E10.FiNAL.HDTV.XviD-EXTREME.avi", "Torrent9.cz"},
		{"Jurassic.World.Dominion.CUSTOM.EXTENDED.2022.2160p.MULTi.VF2.UHD.Blu-ray.REMUX.HDR.DoVi.HEVC.DTS-X.DTS-HDHRA.7.1-MOONLY.mkv", ""},
		{"Last.Call.for.Istanbul.2023.1080p.NF.WEB-DL.DDP5.1.H.264.MKV.torrent", ""},
		{"[Naruto-Kun.Hu] Naruto - 061 [1080p].mkv", "Naruto-Kun.Hu"},
		{"www 1TamilMV ms - The Electric State (2025) HQ HDRip - x264 - [Tam + Tel + Hin] - AAC - 450MB - ESub mkv", "www 1TamilMV ms"},
		{"www 1TamilBlasters rodeo - The Electric State (2025) [1080p HQ HD AVC - x264 - [Tam + Tel + Hin + Eng(ATMOS)] - DDP5 1(640Kbps) - 6 6GB - ESub] mkv", "www 1tamilblasters rodeo"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.site, result.Site)
		})
	}
}
