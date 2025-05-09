package util

import (
	"regexp"
	"strings"

	"github.com/dustin/go-humanize"
)

var normalizeSize = func() func(size string) string {
	re := regexp.MustCompile("(?i)(.+)([^i])b$")
	return func(size string) string {
		return re.ReplaceAllString(strings.TrimSpace(size), "${1}${2}iB")
	}
}()

func ToBytes(size string) int64 {
	bytes, err := humanize.ParseBytes(normalizeSize(size))
	if err != nil {
		return -1
	}
	return int64(bytes)
}

func ToSize(bytes int64) string {
	return strings.Replace(humanize.IBytes(uint64(bytes)), "iB", "B", 1)
}
