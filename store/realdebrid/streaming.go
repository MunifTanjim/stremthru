package realdebrid

import (
	"cmp"
	"slices"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/util"
)

func compareMediaInfoStream(a, b string) int {
	a1, a2, _ := strings.Cut(a, ":")
	b1, b2, _ := strings.Cut(b, ":")
	if c := cmp.Compare(util.SafeParseInt(a1, 0), util.SafeParseInt(b1, 0)); c != 0 {
		return c
	}
	return cmp.Compare(util.SafeParseInt(a2, 0), util.SafeParseInt(b2, 0))
}

type MediaInfoVideo struct {
	Stream     string `json:"stream"`
	Lang       string `json:"lang"`
	LangIso    string `json:"lang_iso"`
	Codec      string `json:"codec"`
	Colorspace string `json:"colorspace"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type MediaInfoAudio struct {
	Stream   string  `json:"stream"`
	Lang     string  `json:"lang"`
	LangISO  string  `json:"lang_iso"`
	Codec    string  `json:"codec"`
	Sampling int     `json:"sampling"`
	Channels float64 `json:"channels"`
}

type MediaInfoSubtitle struct {
	Stream  string `json:"stream"`
	Lang    string `json:"lang"`
	LangIso string `json:"lang_iso"`
	Type    string `json:"type"`
}

type GetMediaInfoData struct {
	*ResponseError
	Filename string  `json:"filename"`
	Hoster   string  `json:"hoster"`   // rd
	Link     string  `json:"link"`     // locked link
	Type     string  `json:"type"`     // show
	Season   string  `json:"season"`   // 00
	Episode  string  `json:"episode"`  // 00
	Year     string  `json:"year"`     // null
	Duration float64 `json:"duration"` // 3700.179
	Bitrate  int     `json:"bitrate"`  // 1784482
	Size     int64   `json:"size"`     // 825362868
	Details  struct {
		Video     util.MapOrEmptyArray[string, MediaInfoVideo]    `json:"video"`     // und1
		Audio     util.MapOrEmptyArray[string, MediaInfoAudio]    `json:"audio"`     // eng1
		Subtitles util.MapOrEmptyArray[string, MediaInfoSubtitle] `json:"subtitles"` // eng1 eng2 ara1 cze1 dan1 ger1 gre1 spa1 spa2 fin1 fre2 heb1 hin1 hun1 ind1 ita1 jpn1 kor1 may1 nob1 dut1 pol1 por1 por2 rum1 swe1 tha1 tur1 chi1 chi2
	} `json:"details"`
	PosterPath   string `json:"poster_path"`   // tmdb.jpg
	BackdropPath string `json:"backdrop_path"` // tmdb.jpg
	BaseURL      string `json:"baseUrl"`
	AudioImage   string `json:"audio_image"`
	Host         string `json:"host"`
}

func (d *GetMediaInfoData) GetVideos() []MediaInfoVideo {
	videos := make([]MediaInfoVideo, 0, len(d.Details.Video))
	for _, v := range d.Details.Video {
		videos = append(videos, v)
	}
	slices.SortFunc(videos, func(a, b MediaInfoVideo) int {
		return compareMediaInfoStream(a.Stream, b.Stream)
	})
	return videos
}

func (d *GetMediaInfoData) GetAudios() []MediaInfoAudio {
	audios := make([]MediaInfoAudio, 0, len(d.Details.Audio))
	for _, a := range d.Details.Audio {
		audios = append(audios, a)
	}
	slices.SortFunc(audios, func(a, b MediaInfoAudio) int {
		return compareMediaInfoStream(a.Stream, b.Stream)
	})
	return audios
}

func (d *GetMediaInfoData) GetSubtitles() []MediaInfoSubtitle {
	subs := make([]MediaInfoSubtitle, 0, len(d.Details.Subtitles))
	for _, s := range d.Details.Subtitles {
		subs = append(subs, s)
	}
	slices.SortFunc(subs, func(a, b MediaInfoSubtitle) int {
		return compareMediaInfoStream(a.Stream, b.Stream)
	})
	return subs
}

type GetMediaInfoParams struct {
	Ctx
	ID string
}

func (c APIClient) GetMediaInfo(params *GetMediaInfoParams) (APIResponse[GetMediaInfoData], error) {
	response := &GetMediaInfoData{}
	res, err := c.Request("GET", "/rest/1.0/streaming/mediaInfos/"+params.ID, params, response)
	return newAPIResponse(res, *response), err
}
