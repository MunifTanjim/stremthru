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

func ParseDuration(input string) (time.Duration, error) {
	if input == "" {
		return 0, nil
	}

	neg := false
	start := 0
	if input[0] == '-' || input[0] == '+' {
		neg = input[0] == '-'
		start = 1
	}

	var n int64
	var hasDot bool
	var dIdx int
	for dIdx = start; dIdx < len(input); dIdx++ {
		c := input[dIdx]
		if c >= '0' && c <= '9' {
			if !hasDot {
				n = n*10 + int64(c-'0')
			}
			continue
		}
		if c == '.' && !hasDot {
			hasDot = true
			continue
		}
		break
	}

	if dIdx == start || dIdx >= len(input) || input[dIdx] != 'd' {
		return time.ParseDuration(input)
	}

	var result time.Duration
	if hasDot {
		totalDays, err := strconv.ParseFloat(input[start:dIdx], 64)
		if err != nil {
			return 0, err
		}
		result = time.Duration(totalDays * 24 * float64(time.Hour))
	} else {
		result = time.Duration(n) * 24 * time.Hour
	}

	if dIdx+1 < len(input) {
		d, err := time.ParseDuration(input[dIdx+1:])
		if err != nil {
			return 0, err
		}
		result += d
	}

	if neg {
		result = -result
	}

	return result, nil
}
