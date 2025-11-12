package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringNormalizer(t *testing.T) {
	n := NewStringNormalizer()

	for _, tc := range []struct {
		in, out string
	}{
		{"Hello World!", "hello world"},
		{"Café-au-lait", "cafe au lait"},
		{`"Quoted" 'String'`, "quoted string"},
		{"   Multiple    Spaces   ", "multiple spaces"},
		{"Special_characters-are_here.", "special characters are here"},
	} {
		assert.Equal(t, tc.out, n.Normalize(tc.in))
	}
}
