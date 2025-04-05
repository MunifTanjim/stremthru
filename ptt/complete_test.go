package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComplete(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ttitle   string
		complete bool
	}{
		{"complete series with full seasons", "[Furi] Avatar - The Last Airbender [720p] (Full 3 Seasons + Extr", true},
		{"complete collection", "Harry.Potter.Complete.Collection.2001-2011.1080p.BluRay.DTS-ETRG", true},
		{"complete collection with all seasons", "Game of Thrones All 7 Seasons 1080p ~∞~ .HakunaMaKoko", true},
		{"complete collection with full series", "Avatar: The Last Airbender Full Series 720p", true},
		{"complete collection with ultimate collection", "Dora the Explorer - Ultimate Collection", true},
		{"complete collection with complete pack", "Mr Bean Complete Pack (Animated, Tv series, 2 Movies) DVDRIP (WA", true},
		{"complete collection with complete set", "American Pie - Complete set (8 movies) 720p mkv - YIFY", true},
		{"complete collection with complete filmography", "Charlie Chaplin - Complete Filmography (87 movies)", true},
		{"complete collection with movies complete", "Monster High Movies Complete 2014", true},
		{"complete collection all movies", "Harry Potter All Movies Collection 2001-2011 720p Dual KartiKing", true},
		{"complete movie collection", "The Clint Eastwood Movie Collection", true},
		{"complete collection movies", "Clint Eastwood Collection - 15 HD Movies", true},
		{"complete movies collection", "Official  IMDb  Top  250  Movies  Collection  6/17/2011", true},
		{"collection", "The Texas Chainsaw Massacre Collection (1974-2017) BDRip 1080p", true},
		{"duology", "Snabba.Cash.I-II.Duology.2010-2012.1080p.BluRay.x264.anoXmous", true},
		{"trilogy", "Star Wars Original Trilogy 1977-1983 Despecialized 720p", true},
		{"quadrology", "The.Wong.Kar-Wai.Quadrology.1990-2004.1080p.BluRay.x264.AAC.5.1-", true},
		{"quadrilogy", "Lethal.Weapon.Quadrilogy.1987-1992.1080p.BluRay.x264.anoXmous", true},
		{"tetralogy", "X-Men.Tetralogy.BRRip.XviD.AC3.RoSubbed-playXD", true},
		{"pentalogy", "Mission.Impossible.Pentalogy.1996-2015.1080p.BluRay.x264.AAC.5.1", true},
		{"hexalogy", "Mission.Impossible.Hexalogy.1996-2018.SweSub.1080p.x264-Justiso", true},
		{"hexalogy", "American.Pie.Heptalogy.SWESUB.DVDRip.XviD-BaZZe", true},
		{"anthalogy", "The Exorcist 1, 2, 3, 4, 5 - Complete Horror Anthology 1973-2005", true},
		{"saga", "Harry.Potter.Complete.Saga. I - VIII .1080p.Bluray.x264.anoXmous", true},
		{"italian complete", "Inganno - Miniserie (2024) [COMPLETA] SD H264 ITA AAC-UBi", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.complete, result.Complete)
		})
	}

	for _, tc := range []struct {
		name     string
		ttitle   string
		complete bool
		title    string
	}{
		{"not remove collection from title", "[Erai-raws] Ninja Collection - 05 [720p][Multiple Subtitle].mkv", false, "Ninja Collection"},
		{"not remove kolekcja from title", "Kolekcja Halloween (1978-2022) [720p] [BRRip] [XviD] [AC3-ELiTE] [Lektor PL]", true, "Kolekcja Halloween"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.complete, result.Complete)
			assert.Equal(t, tc.title, result.Title)
		})
	}

	// PY
	for _, tc := range []struct {
		ttitle   string
		complete bool
		title    string
	}{
		{"Furiosa - A Mad Max Saga (2024) 2160p H265 HDR10 D V iTA EnG AC3 5 1 Sub iTA EnG NUiTA NUEnG AsPiDe-MIRCrew mkv", true, "Furiosa - A Mad Max Saga"},
		{"[Judas] Vinland Saga (Season 2) [1080p][HEVC x265 10bit][Multi-Subs]", true, "Vinland Saga"},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.complete, result.Complete)
			assert.Equal(t, tc.title, result.Title)
		})
	}
}
