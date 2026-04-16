package shared

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/torrent_stream/media_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
)

var trailingDigitsRegex = regexp.MustCompile(`\d+$`)

func rdLinkIdToDownloadId(linkId string) string {
	return trailingDigitsRegex.ReplaceAllLiteralString(linkId, "")
}

func rdProbeMediaInfo(apiKey, linkId string) (*media_info.MediaInfo, error) {
	params := &realdebrid.GetMediaInfoParams{ID: rdLinkIdToDownloadId(linkId)}
	params.APIKey = apiKey
	res, err := rdStore.GetClient().GetMediaInfo(params)
	if err != nil {
		return nil, fmt.Errorf("realdebrid media info: %w", err)
	}
	return rdFormatMediaInfo(&res.Data), nil
}

func rdFormatMediaInfo(data *realdebrid.GetMediaInfoData) *media_info.MediaInfo {
	mi := &media_info.MediaInfo{
		Source: string(store.StoreCodeRealDebrid),
	}

	for _, v := range data.GetVideos() {
		mi.Video = &media_info.MediaInfoVideo{
			Codec:  v.Codec,
			Width:  v.Width,
			Height: v.Height,
		}
		break
	}

	for _, a := range data.GetAudios() {
		aud := media_info.MediaInfoAudio{
			Codec:    a.Codec,
			Language: a.LangISO,
		}
		if a.Channels == 2.0 {
			aud.Channels = 2
			aud.ChannelLayout = "stereo"
		} else if ch := math.Ceil(a.Channels); ch == a.Channels {
			aud.Channels = int(ch)
		} else {
			aud.ChannelLayout = fmt.Sprintf("%g", a.Channels)
			x, y, _ := strings.Cut(aud.ChannelLayout, ".")
			aud.Channels = util.SafeParseInt(x, 0) + util.SafeParseInt(y, 0)
		}
		mi.Audio = append(mi.Audio, aud)
	}

	for _, s := range data.GetSubtitles() {
		mi.Subtitle = append(mi.Subtitle, media_info.MediaInfoSubtitle{
			Codec:    strings.ToLower(s.Type),
			Language: s.LangIso,
		})
	}

	mi.Format = &media_info.MediaInfoFormat{
		Duration: time.Duration(data.Duration * float64(time.Second)),
		Size:     data.Size,
		BitRate:  int64(data.Bitrate),
	}

	return mi
}

func init() {
	media_info.RegisterStoreProber(string(store.StoreCodeRealDebrid), rdProbeMediaInfo)
}
