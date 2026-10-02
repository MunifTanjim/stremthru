package lzstring

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecompressFromEncodedUriComponent(t *testing.T) {
	content, err := os.ReadFile("testdata/15dc99e3-39d1-4f86-bfc7-fc3ef59300ae.txt")
	if err != nil {
		t.Fatal(err)
	}
	input := string(content)

	// decoded with the reference js implementation (lz-string@1)
	decoded := `{"title":"Test","torrents":[{"filename":"Sherlock.S04E01.The.Six.Thatchers.1080p.HEVC.x265-MeGusta.mkv","hash":"a96a231ba49d402737114499de94a67304ab8645","bytes":1006072466},{"filename":"Vera.S01E01-04.WEBDL.1080p.ITA","hash":"60830ed2f6ee04f6cda4b1a81b0c292b9db4b2e2","bytes":13633227982}]}`

	for _, tc := range []struct {
		name   string
		length int
		result string
		err    string
	}{
		{"full", 365, decoded, ""},
		{"without trailing padding", 364, decoded, ""},
		{"truncated end marker", 363, "", "Unexpected end of buffer reached."},
		{"truncated half", 182, "", "Unexpected end of buffer reached."},
		{"truncated to one char", 1, "", "Unexpected end of buffer reached."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DecompressFromEncodedUriComponent(input[:tc.length])
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.result, result)
		})
	}
}
