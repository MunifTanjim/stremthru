package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestEpisodeCode(t *testing.T) {

	for _, tc := range []struct {
		ttitle      string
		episodeCode string
	}{
		{"[Golumpa] Fairy Tail - 214 [FuniDub 720p x264 AAC] [5E46AC39].mkv", "5E46AC39"},
		{"[Exiled-Destiny]_Tokyo_Underground_Ep02v2_(41858470).mkv", "41858470"},
		{"[ACX]El_Cazador_de_la_Bruja_-_19_-_A_Man_Who_Protects_[SSJ_Saiyan_Elite]_[9E199846].mkv", "9E199846"},
		{"[CBM]_Medaka_Box_-_11_-_This_Is_the_End!!_[720p]_[436E0E90]", "436E0E90"},
		{"Gankutsuou.-.The.Count.Of.Monte.Cristo[2005].-.04.-.[720p.BD.HEVC.x265].[FLAC].[Jd].[DHD].[b6e6e648].mkv", "B6E6E648"},
		{"[D0ugyB0y] Nanatsu no Taizai Fundo no Shinpan - 01 (1080p WEB NF x264 AAC[9CC04E06]).mkv", "9CC04E06"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.episodeCode, result.EpisodeCode)
		})
	}

	for _, tc := range []struct {
		name        string
		ttitle      string
		episodeCode string
	}{

		{"not episode code not at the end", "Lost.[Perdidos].6x05.HDTV.XviD.[www.DivxTotaL.com].avi", ""},
		{"not episode code when it's a word", "Lost - Stagioni 01-06 (2004-2010) [COMPLETA] SD x264 AAC ITA SUB ITA", ""},
		{"not episode code when it's only numbers", "The voice of Holland S05E08 [20141017]  NL Battles 1.mp4", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.episodeCode, result.EpisodeCode)
		})
	}
}
