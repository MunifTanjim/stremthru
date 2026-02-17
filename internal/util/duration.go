package util

import (
	"strconv"
	"strings"
	"time"
)

var durationFormatUnits = []struct {
	seconds int64
	suffix  string
}{
	{365 * 24 * 3600, "yr"},
	{30 * 24 * 3600, "mo"},
	{7 * 24 * 3600, "wk"},
	{24 * 3600, "d"},
	{3600, "h"},
	{60, "m"},
	{1, "s"},
}

func FormatDuration(d time.Duration, maxParts int) string {
	totalSeconds := int64(d.Seconds())
	if totalSeconds == 0 {
		return "0s"
	}

	parts := make([]string, 0, maxParts)
	remaining := totalSeconds
	for _, u := range durationFormatUnits {
		if val := remaining / u.seconds; val > 0 {
			parts = append(parts, strconv.FormatInt(val, 10)+u.suffix)
			if len(parts) >= maxParts {
				break
			}
			remaining %= u.seconds
		}
	}

	return strings.Join(parts, " ")
}
