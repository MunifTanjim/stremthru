package usenet_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteRange(t *testing.T) {
	t.Run("Count", func(t *testing.T) {
		r := ByteRange{Start: 0, End: 100}
		assert.Equal(t, int64(100), r.Count())

		r = ByteRange{Start: 50, End: 150}
		assert.Equal(t, int64(100), r.Count())

		r = ByteRange{Start: 0, End: 0}
		assert.Equal(t, int64(0), r.Count())
	})

	t.Run("Contains", func(t *testing.T) {
		r := ByteRange{Start: 10, End: 20}

		// Inside
		assert.True(t, r.Contains(10))
		assert.True(t, r.Contains(15))
		assert.True(t, r.Contains(19))

		// Outside
		assert.False(t, r.Contains(9))
		assert.False(t, r.Contains(20)) // End is exclusive
		assert.False(t, r.Contains(21))
	})

	t.Run("ContainsRange", func(t *testing.T) {
		r := ByteRange{Start: 0, End: 100}

		// Fully contained
		assert.True(t, r.ContainsRange(ByteRange{Start: 0, End: 100}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 10, End: 90}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 0, End: 50}))
		assert.True(t, r.ContainsRange(ByteRange{Start: 50, End: 100}))

		// Not contained
		assert.False(t, r.ContainsRange(ByteRange{Start: -10, End: 50}))
		assert.False(t, r.ContainsRange(ByteRange{Start: 50, End: 150}))
		assert.False(t, r.ContainsRange(ByteRange{Start: -10, End: 150}))
	})

	t.Run("NewByteRangeFromSize", func(t *testing.T) {
		r := NewByteRangeFromSize(0, 100)
		assert.Equal(t, int64(0), r.Start)
		assert.Equal(t, int64(100), r.End)

		r = NewByteRangeFromSize(50, 100)
		assert.Equal(t, int64(50), r.Start)
		assert.Equal(t, int64(150), r.End)
	})
}
