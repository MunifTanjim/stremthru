package torrent_info

import "strings"

func GetCategoryFromStremId(sid string) TorrentInfoCategory {
	category := TorrentInfoCategoryUnknown
	if strings.HasPrefix(sid, "tt") {
		if strings.Contains(sid, ":") {
			category = TorrentInfoCategorySeries
		} else {
			category = TorrentInfoCategoryMovie
		}
	}
	return category
}
