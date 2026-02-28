package media_info

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"
)

type MediaInfoVideo struct {
	Codec  string   `json:"codec,omitempty"`
	HDR    []string `json:"hdr,omitempty"`
	Height int      `json:"h,omitempty"`
	Width  int      `json:"w,omitempty"`
}

type MediaInfoAudio struct {
	ChannelLayout string `json:"ch_layout,omitempty"`
	Channels      int    `json:"ch,omitempty"`
	Codec         string `json:"codec,omitempty"`
	Language      string `json:"lang,omitempty"`
	Profile       string `json:"profile,omitempty"`

	Commentary      bool `json:"commentary,omitempty"`
	Default         bool `json:"default,omitempty"`
	Dub             bool `json:"dub,omitempty"`
	HearingImpaired bool `json:"hearing_impaired,omitempty"`
	Original        bool `json:"original,omitempty"`
	VisualImpaired  bool `json:"visual_impaired,omitempty"`
}

type MediaInfoSubtitle struct {
	Codec    string `json:"codec,omitempty"`
	Language string `json:"lang,omitempty"`
	Title    string `json:"title,omitempty"`

	Default         bool `json:"default,omitempty"`
	Forced          bool `json:"forced,omitempty"`
	HearingImpaired bool `json:"hearing_impaired,omitempty"`
}

type MediaInfoFormat struct {
	Name     string        `json:"n,omitempty"`
	Duration time.Duration `json:"dur,omitempty"`
	Size     int64         `json:"s,omitempty"`
	BitRate  int64         `json:"br,omitempty"`
}

type MediaInfo struct {
	Video       *MediaInfoVideo     `json:"video,omitempty"`
	Audio       []MediaInfoAudio    `json:"audio,omitempty"`
	Subtitle    []MediaInfoSubtitle `json:"subtitle,omitempty"`
	Format      *MediaInfoFormat    `json:"format,omitempty"`
	HasChapters bool                `json:"has_chapters,omitempty"`
}

func detectHDR(stream *ffprobe.Stream) []string {
	var hdr []string

	switch stream.CodecTagString {
	case "dvhe", "dvh1", "dvav", "dva1", "dav1":
		hdr = append(hdr, "DV")
	default:
		switch stream.CodecName {
		case "dvhe", "dvh1", "dvav", "dva1", "dav1":
			hdr = append(hdr, "DV")
		}
	}

	for _, sd := range stream.SideDataList {
		switch sd.Type {
		case "DOVI configuration record":
			if !slices.Contains(hdr, "DV") {
				hdr = append(hdr, "DV")
			}
		case "HDR10+ Dynamic Metadata (SMPTE 2094-40)":
			hdr = append(hdr, "HDR10+")
		}
	}

	switch stream.ColorTransfer {
	case "smpte2084":
		if !slices.Contains(hdr, "HDR10+") {
			hdr = append(hdr, "HDR10")
		}
	case "arib-std-b67":
		hdr = append(hdr, "HLG")
	}

	return hdr
}

func getTagString(stream *ffprobe.Stream, key string) string {
	val, _ := stream.TagList.GetString(key)
	return val
}

func Probe(ctx context.Context, url string) (*MediaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	data, err := ffprobe.ProbeURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	mi := MediaInfo{}

	for _, stream := range data.StreamType(ffprobe.StreamVideo) {
		if mi.Video != nil {
			continue
		}
		video := &MediaInfoVideo{
			Codec:  stream.CodecName,
			HDR:    detectHDR(&stream),
			Height: stream.Height,
			Width:  stream.Width,
		}
		mi.Video = video
	}
	for _, stream := range data.StreamType(ffprobe.StreamAudio) {
		mi.Audio = append(mi.Audio, MediaInfoAudio{
			ChannelLayout: stream.ChannelLayout,
			Channels:      stream.Channels,
			Codec:         stream.CodecName,
			Language:      getTagString(&stream, "language"),
			Profile:       stream.Profile,

			Commentary:      stream.Disposition.Comment == 1,
			Default:         stream.Disposition.Default == 1,
			Dub:             stream.Disposition.Dub == 1,
			HearingImpaired: stream.Disposition.HearingImpaired == 1,
			Original:        stream.Disposition.Original == 1,
			VisualImpaired:  stream.Disposition.VisualImpaired == 1,
		})
	}
	for _, stream := range data.StreamType(ffprobe.StreamSubtitle) {
		mi.Subtitle = append(mi.Subtitle, MediaInfoSubtitle{
			Codec:    stream.CodecName,
			Language: getTagString(&stream, "language"),
			Title:    getTagString(&stream, "title"),

			Default:         stream.Disposition.Default == 1,
			Forced:          stream.Disposition.Forced == 1,
			HearingImpaired: stream.Disposition.HearingImpaired == 1,
		})
	}

	if data.Format != nil {
		fi := &MediaInfoFormat{
			Name:     data.Format.FormatName,
			Duration: data.Format.Duration(),
		}
		if size, err := strconv.ParseInt(data.Format.Size, 10, 64); err == nil {
			fi.Size = size
		}
		if br, err := strconv.ParseInt(data.Format.BitRate, 10, 64); err == nil {
			fi.BitRate = br
		}
		mi.Format = fi
	}

	mi.HasChapters = len(data.Chapters) > 0

	return &mi, nil
}
