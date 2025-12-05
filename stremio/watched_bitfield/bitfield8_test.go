package stremio_watched_bitfield

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeSerialize(t *testing.T) {
	json_value := `"eJyTZwAAAEAAIA=="`

	t.Run("basic deserialization and serialization", func(t *testing.T) {
		expected := &BitField8{
			Length: 16,
			values: []byte{31, 0},
		}

		var actual_from_json BitField8
		err := json.Unmarshal([]byte(json_value), &actual_from_json)
		assert.NoError(t, err, "Should deserialize")

		assert.Equal(t, expected.Length, actual_from_json.Length)
		assert.Equal(t, expected.values, actual_from_json.values)

		actual_to_json, err := json.Marshal(&actual_from_json)
		assert.NoError(t, err, "Should serialize")

		var roundtrip_from_json BitField8
		err = json.Unmarshal(actual_to_json, &roundtrip_from_json)
		assert.NoError(t, err, "Should deserialize round-trip")

		assert.Equal(t, expected.Length, roundtrip_from_json.Length)
		assert.Equal(t, expected.values, roundtrip_from_json.values)
	})

	t.Run("with custom length", func(t *testing.T) {
		expected := &BitField8{
			Length: 9,
			values: []byte{31, 0},
		}

		var actual_from_json BitField8
		err := json.Unmarshal([]byte(json_value), &actual_from_json)
		assert.NoError(t, err, "Should deserialize")

		// The fact that we have custom length is the reason these two values will not be the same
		assert.NotEqual(t, expected.Length, actual_from_json.Length)
		assert.Equal(t, 16, actual_from_json.Length)
		assert.Equal(t, expected.values, actual_from_json.values)

		actual_to_json, err := json.Marshal(&expected)
		assert.NoError(t, err, "Should serialize")

		var roundtrip_from_json BitField8
		err = json.Unmarshal(actual_to_json, &roundtrip_from_json)
		assert.NoError(t, err, "Should deserialize round-trip")

		assert.Equal(t, expected.values, roundtrip_from_json.values)
		assert.Equal(t, 16, roundtrip_from_json.Length)
	})
}

func TestParseLength(t *testing.T) {
	watched := "eJyTZwAAAEAAIA=="

	t.Run("with explicit length", func(t *testing.T) {
		bf, err := NewBitField8FromString(watched, 9)
		assert.NoError(t, err)
		assert.Equal(t, 9, bf.Length)
	})

	t.Run("without length (rounded to next byte)", func(t *testing.T) {
		bf, err := NewBitField8FromString(watched, 0)
		assert.NoError(t, err)
		assert.Equal(t, 16, bf.Length)
	})
}
