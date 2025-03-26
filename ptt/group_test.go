package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		group  string
	}{
		{"HD2", "Nocturnal Animals 2016 VFF 1080p BluRay DTS HEVC-HD2", "HD2"},
		{"HDH", "Gold 2016 1080p BluRay DTS-HD MA 5 1 x264-HDH", "HDH"},
		{"YIFY", "Hercules (2014) 1080p BrRip H264 - YIFY", "YIFY"},
		{"before container file type", "The.Expanse.S05E02.720p.WEB.x264-Worldmkv.mkv", "Worldmkv"},
		{"with site source tag", "The.Expanse.S05E02.PROPER.720p.WEB.h264-KOGi[rartv]", "KOGi"},
		{"with site source tag before container file type", "The.Expanse.S05E02.1080p.AMZN.WEB.DDP5.1.x264-NTb[eztv.re].mp4", "NTb"},
		{"no group", "Western - L'homme qui n'a pas d'étoile-1955.Multi.DVD9", ""},
		{"no group with hyphen separator", "Power (2014) - S02E03.mp4", ""},
		{"no group with hyphen separator and no container", "Power (2014) - S02E03", ""},
		{"no group when it is episode", "3-Nen D-Gumi Glass no Kamen - 13", ""},
		{"no group when it is ep symbol", "3-Nen D-Gumi Glass no Kamen - Ep13", ""},
		{"anime group in brackets", "[AnimeRG] One Punch Man - 09 [720p].mkv", "AnimeRG"},
		{"anime group in brackets with underscores", "[Mazui]_Hyouka_-_03_[DF5E813A].mkv", "Mazui"},
		{"anime group in brackets with numbers", "[H3] Hunter x Hunter - 38 [1280x720] [x264]", "H3"},
		{"anime group in brackets with spaces", "[KNK E MMS Fansubs] Nisekoi - 20 Final [PT-BR].mkv", "KNK E MMS Fansubs"},
		{"anime group in brackets when bracket part exist at the end", "[ToonsHub] JUJUTSU KAISEN - S02E01 (Japanese 2160p x264 AAC) [Multi-Subs].mkv", "ToonsHub"},
		{"anime group in brackets with a link", "[HD-ELITE.NET] -  The.Art.Of.The.Steal.2014.DVDRip.XviD.Dual.Aud", ""},
		{"not detect brackets group when group is detected at the end of title", "[Russ]Lords.Of.London.2014.XviD.H264.AC3-BladeBDP", "BladeBDP"},
		{"group in parenthesis", "Jujutsu Kaisen S02E01 2160p WEB H.265 AAC -Tsundere-Raws (B-Global).mkv", "B-Global"},
		{"not detect brackets group when it contains other parsed parameters", "[DVD-RIP] Kaavalan (2011) Sruthi XVID [700Mb] [TCHellRaiser]", ""},
		{"not detect brackets group when it contains other parsed parameters for series", "[DvdMux - XviD - Ita Mp3 Eng Ac3 - Sub Ita Eng] Sanctuary S01e01", ""},
		{"not detect group from episode", "the-x-files-502.mkv", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.group, result.Group)
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle string
		group  string
	}{
		{"[ Torrent9.cz ] The.InBetween.S01E10.FiNAL.HDTV.XviD-EXTREME.avi", "EXTREME"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.group, result.Group)
		})
	}
}
