package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestRemastered(t *testing.T) {
	testCases := []struct {
		name       string
		ttitle     string
		remastered bool
	}{
		{"remastered", "The Fifth Element 1997 REMASTERED MULTi 1080p BluRay HDLight AC3 x264 Zone80", true},
		{"remaster (without 'ed')", "Predator 1987 REMASTER MULTi 1080p BluRay x264 FiDELiO", true},
		{"polish rekonstrukcija", "Gra 1968 [REKONSTRUKCJA] [1080p.WEB-DL.H264.AC3-FT] [Napisy PL] [Film Polski]", true},
		{"not remastered", "Have I Got News For You S53E02 EXTENDED 720p HDTV x264-QPEL", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.remastered, result.Remastered)
		})
	}
}
