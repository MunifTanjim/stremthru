package util

import "regexp"

var torrentInfoHashRegex = regexp.MustCompile(`^(?:[a-f0-9]{40}|[A-F0-9]{40})$`)

func IsTorrentInfoHash(v string) bool {
	return torrentInfoHashRegex.MatchString(v)
}
