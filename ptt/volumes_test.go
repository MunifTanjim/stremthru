package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestVolumes(t *testing.T) {
	for _, tc := range []struct {
		ttitle  string
		volumes []int
	}{
		{"[MTBB] Sword Art Onlineː Alicization - Volume 2 (BD 1080p)", []int{2}},
		{"[Neutrinome] Sword Art Online Alicization Vol.2 - VOSTFR [1080p BDRemux] + DDL", []int{2}},
		{"[Mr. Kimiko] Oh My Goddess! - Vol. 7 [Kobo][2048px][CBZ]", []int{7}},
		{"[MTBB] Cross Game - Volume 1-3 (WEB 720p)", []int{1, 2, 3}},
		{"PIXAR SHORT FILMS COLLECTION - VOLS. 1 & 2 + - BDrip 1080p", []int{1, 2}},
		{"Altair - A Record of Battles Vol. 01-08 (Digital) (danke-Empire)", []int{1, 2, 3, 4, 5, 6, 7, 8}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.volumes, result.Volumes)
		})
	}

	for _, tc := range []struct {
		ttitle  string
		title   string
		volumes []int
	}{
		{"Guardians of the Galaxy Vol. 2 (2017) 720p HDTC x264 MKVTV", "Guardians of the Galaxy Vol. 2", nil},
		{"Kill Bill: Vol. 1 (2003) BluRay 1080p 5.1CH x264 Ganool", "Kill Bill: Vol. 1", nil},
		{"[Valenciano] Aquarion EVOL - 22 [1080p][AV1 10bit][FLAC][Eng sub].mkv", "Aquarion EVOL", nil},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.title, result.Title)
			assert.Equal(t, tc.volumes, result.Volumes)
		})
	}
}
