package stremio_transformer

import (
	"strings"

	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func GetMediaInfoStreamLangs[T media_info.MediaInfoStreamLanger](streams []T) []string {
	langs := make([]string, 0, len(streams))
	if len(streams) == 0 {
		return langs
	}
	seen := util.NewSet[string]()
	for i := range streams {
		if code, ok := language_to_code[strings.ToLower(streams[i].Lang())]; ok && code != "" && !seen.Has(code) {
			langs = append(langs, code)
			seen.Add(code)
		}
	}
	return langs
}
