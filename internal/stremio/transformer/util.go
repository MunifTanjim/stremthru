package stremio_transformer

import (
	"strings"

	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func ApplyMediaInfo(data *StreamExtractorResult, mi *media_info.MediaInfo) {
	if mi == nil {
		return
	}
	if len(mi.Audio) > 0 {
		data.Languages = GetMediaInfoStreamLangs(mi.Audio)
		if mi.Source == "" {
			data.Channels = mi.Channels()
		}
	}
	if len(mi.Subtitle) > 0 {
		data.Subtitles = GetMediaInfoStreamLangs(mi.Subtitle)
	}
	if mi.Video != nil {
		if mi.Video.Codec != "" {
			data.Codec = strings.ToUpper(mi.Video.Codec)
		}
		if len(mi.Video.HDR) > 0 {
			data.HDR = mi.Video.HDR
		}
	}
	if mi.Format != nil {
		if mi.Format.BitRate > 0 {
			data.BitRate = StreamExtractorResultBitRate(mi.Format.BitRate)
		}
	}
}

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
