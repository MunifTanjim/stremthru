package util

import (
	"math/rand"
	"time"
)

func GetRandomDuration(minValue, maxValue time.Duration) time.Duration {
	if minValue >= maxValue {
		return minValue
	}
	offset := time.Duration(rand.Int63n(int64(maxValue-minValue) + 1))
	return minValue + offset
}

func IsToday(t time.Time) bool {
	now := time.Now()
	y, m, d := now.Date()
	ty, tm, td := t.Date()
	return y == ty && m == tm && d == td
}

func HasDurationPassedSince(t time.Time, dur time.Duration) bool {
	return time.Since(t) >= dur
}

func MustParseTime(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return t
}
