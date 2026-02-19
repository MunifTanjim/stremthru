package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	for _, tt := range []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m 30s"},
		{"hours only", 2 * time.Hour, "2h"},
		{"mixed with zero gaps", 1*time.Hour + 5*time.Second, "1h 5s"},
		{"days", 3*24*time.Hour + 12*time.Hour, "3d 12h"},
		{"weeks", 2*7*24*time.Hour + 3*24*time.Hour, "2wk 3d"},
		{"months", 2*30*24*time.Hour + 1*7*24*time.Hour + 3*24*time.Hour, "2mo 1wk 3d"},
		{"years", 1*365*24*time.Hour + 1*30*24*time.Hour + 5*24*time.Hour + 5*time.Hour, "1yr 1mo 5d 5h"},
		{"large duration",
			2*365*24*time.Hour + 3*30*24*time.Hour + 2*7*24*time.Hour + 4*24*time.Hour + 23*time.Hour + 59*time.Minute + 59*time.Second,
			"2yr 3mo 2wk 4d 23h 59m 59s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatDuration(tt.duration, 7))
		})
	}
}

func TestParseDuration(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"empty string", "", 0},
		{"single day", "1d", 24 * time.Hour},
		{"multiple days", "2d", 48 * time.Hour},
		{"days and hours", "2d12h", 60 * time.Hour},
		{"days and minutes", "1d30m", 24*time.Hour + 30*time.Minute},
		{"days hours minutes", "1d2h30m", 26*time.Hour + 30*time.Minute},
		{"minutes only", "30m", 30 * time.Minute},
		{"hours and minutes", "2h30m", 2*time.Hour + 30*time.Minute},
		{"hours and seconds", "1h30s", 1*time.Hour + 30*time.Second},
		{"fractional days", "0.5d", 12 * time.Hour},
		{"negative days", "-1d", -24 * time.Hour},
		{"negative days and hours", "-2d12h", -60 * time.Hour},
		{"positive sign", "+1d", 24 * time.Hour},
		{"negative fractional days", "-0.5d", -12 * time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dur, err := ParseDuration(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, dur)
		})
	}
}
