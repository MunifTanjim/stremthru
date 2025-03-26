package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDubbed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		dubbed bool
	}{
		{"the english dubbed language correctly", "Yo-Kai Watch S01E71 DUBBED 720p HDTV x264-W4F", true},
		{"dub correctly", "[Golumpa] Kochoki - 11 (Kochoki - Wakaki Nobunaga) [English Dub] [FuniDub 720p x264 AAC] [MKV] [4FA0D898]", true},
		{"dub correctly", "[Golumpa] Kochoki - 11 (Kochoki - Wakaki Nobunaga) [English Dub] [FuniDub 720p x264 AAC] [MKV] [4FA0D898]", true},
		{"dubs correctly", "[Aomori-Raws] Juushinki Pandora (01-13) [Dubs & Subs]", true},
		{"dual audio correctly", "[LostYears] Tsuredure Children (WEB 720p Hi10 AAC) [Dual-Audio]", true},
		{"dual-audio correctly", "[DB] Gamers! [Dual Audio 10bit 720p][HEVC-x265]", true},
		{"multi-audio correctly", "[DragsterPS] Yu-Gi-Oh! S02 [480p] [Multi-Audio] [Multi-Subs]", true},
		{"dublado correctly", "A Freira (2018) Dublado HD-TS 720p", true},
		{"dual correctly", "Fame (1980) [DVDRip][Dual][Ac3][Eng-Spa]", true},
		{"dubbing correctly", "Vaiana Skarb oceanu / Moana (2016) [720p] [WEB-DL] [x264] [Dubbing]", true},
		{"not dual subs", "[Hakata Ramen] Hoshiai No Sora (Stars Align) 01 [1080p][HEVC][x265][10bit][Dual-Subs] HR-DR", false},
		{"not multi-dub", "[IceBlue] Naruto (Season 01) - [Multi-Dub][Multi-Sub][HEVC 10Bits] 800p BD", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.dubbed, result.Dubbed)
		})
	}
}
