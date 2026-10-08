package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseByteRange_Normal(t *testing.T) {
	start, end, ok := parseByteRange("bytes=0-499")
	assert.True(t, ok)
	assert.Equal(t, int64(0), start)
	assert.Equal(t, int64(499), end)
}

func TestParseByteRange_OpenEnded(t *testing.T) {
	start, end, ok := parseByteRange("bytes=100-")
	assert.True(t, ok)
	assert.Equal(t, int64(100), start)
	assert.Equal(t, int64(-1), end)
}

func TestParseByteRange_LargeOffset(t *testing.T) {
	start, end, ok := parseByteRange("bytes=276134889-")
	assert.True(t, ok)
	assert.Equal(t, int64(276134889), start)
	assert.Equal(t, int64(-1), end)
}

func TestParseByteRange_SuffixRange(t *testing.T) {
	_, _, ok := parseByteRange("-500")
	assert.False(t, ok, "suffix ranges should return ok=false")
}

func TestParseByteRange_BytesSuffixRange(t *testing.T) {
	_, _, ok := parseByteRange("bytes=-500")
	assert.False(t, ok, "bytes= suffix ranges should return ok=false")
}

func TestParseByteRange_MultiRange(t *testing.T) {
	start, end, ok := parseByteRange("bytes=0-99,200-299")
	assert.True(t, ok, "multi-range should parse first range")
	assert.Equal(t, int64(0), start)
	assert.Equal(t, int64(99), end)
}

func TestParseByteRange_MissingPrefix(t *testing.T) {
	_, _, ok := parseByteRange("0-499")
	assert.False(t, ok)
}

func TestParseByteRange_EmptyString(t *testing.T) {
	_, _, ok := parseByteRange("")
	assert.False(t, ok)
}

func TestParseByteRange_NonNumericStart(t *testing.T) {
	_, _, ok := parseByteRange("bytes=abc-499")
	assert.False(t, ok)
}

func TestParseByteRange_NonNumericEnd(t *testing.T) {
	_, _, ok := parseByteRange("bytes=0-abc")
	assert.False(t, ok)
}

func TestParseByteRange_ZeroToZero(t *testing.T) {
	start, end, ok := parseByteRange("bytes=0-0")
	assert.True(t, ok)
	assert.Equal(t, int64(0), start)
	assert.Equal(t, int64(0), end)
}
