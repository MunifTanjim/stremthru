package util

import "time"

func IsToday(t time.Time) bool {
	now := time.Now()
	y, m, d := now.Date()
	ty, tm, td := t.Date()
	return y == ty && m == tm && d == td
}

func HasDurationPassedSince(t time.Time, dur time.Duration) bool {
	return time.Since(t) >= dur
}
