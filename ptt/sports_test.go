package ptt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSports(t *testing.T) {
	for _, tc := range []struct {
		ttitle string
		result Result
	}{
		{"UFC.239.PPV.Jones.Vs.Santos.HDTV.x264-PUNCH[TGx]", Result{
			Title:   "UFC 239 Jones Vs Santos",
			Quality: "HDTV",
			Codec:   "x264",
			Group:   "PUNCH",
		}},
		{"UFC.Fight.Night.158.Cowboy.vs.Gaethje.WEB.x264-PUNCH[TGx]", Result{
			Title:   "UFC Fight Night 158 Cowboy vs Gaethje",
			Quality: "WEB",
			Codec:   "x264",
			Group:   "PUNCH",
		}},
		{"UFC 226 PPV Miocic vs Cormier HDTV x264-Ebi [TJET]", Result{
			Title:   "UFC 226 Miocic vs Cormier",
			Quality: "HDTV",
			Codec:   "x264",
		}},
	} {
		t.Run(tc.ttitle, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, &tc.result, result)
		})
	}

}
