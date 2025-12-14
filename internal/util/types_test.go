package util

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONTime_UnmarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input       string
		expectTime  time.Time
		expectRaw   string
		expectError bool
	}{
		{
			name:       "valid RFC3339 time",
			input:      `"2024-01-15T10:30:00Z"`,
			expectTime: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expectRaw:  `"2024-01-15T10:30:00Z"`,
		},
		{
			name:       "empty string",
			input:      `""`,
			expectTime: time.Time{},
			expectRaw:  `""`,
		},
		{
			name:       "null",
			input:      `null`,
			expectTime: time.Time{},
			expectRaw:  `null`,
		},
		{
			name:        "invalid time format",
			input:       `"not-a-time"`,
			expectError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var jt JSONTime
			err := json.Unmarshal([]byte(tc.input), &jt)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectTime, jt.Time)
			assert.Equal(t, tc.expectRaw, string(jt.raw))
		})
	}
}

func TestJSONTime_RoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{
			name:  "valid time",
			input: `"2024-01-15T10:30:00Z"`,
		},
		{
			name:  "null",
			input: `null`,
		},
		{
			name:  "empty string",
			input: `""`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var jt JSONTime
			err := json.Unmarshal([]byte(tc.input), &jt)
			require.NoError(t, err)

			data, err := json.Marshal(jt)
			require.NoError(t, err)
			assert.Equal(t, tc.input, string(data))
		})
	}
}
