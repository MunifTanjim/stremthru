package torrent_info

import (
	"strings"
)

func GetCategoryFromStremId(sid, sType string) TorrentInfoCategory {
	if sType == "movie" {
		return TorrentInfoCategoryMovie
	}
	if sType == "series" {
		return TorrentInfoCategorySeries
	}
	category := TorrentInfoCategoryUnknown
	if strings.HasPrefix(sid, "tt") {
		if sepCount := strings.Count(sid, ":"); sepCount == 2 {
			category = TorrentInfoCategorySeries
		} else if sepCount == 0 {
			category = TorrentInfoCategoryMovie
		}
	}
	return category
}
