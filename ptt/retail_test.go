package ptt

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestRetail(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ttitle string
		retail bool
	}{
		{"release is retail", "MONSTER HIGH: ELECTRIFIED (2017) Retail PAL DVD9 [EAGLE]", true},
		{"not retail", "Have I Got News For You S53E02 EXTENDED 720pjHDTV x264-QPEL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.ttitle)
			assert.Equal(t, tc.retail, result.Retail)
		})
	}
}
