package media_probe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"
)

type VideoInfo struct {
	Codec      string `json:"codec,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}

type AudioTrack struct {
	Codec    string `json:"codec,omitempty"`
	Channels int    `json:"channels,omitempty"`
	Layout   string `json:"layout,omitempty"`
	Language string `json:"language,omitempty"`
}

type SubtitleTrack struct {
	Codec    string `json:"codec,omitempty"`
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
}

type FormatInfo struct {
	Name     string  `json:"name,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Size     int64   `json:"size,omitempty"`
	BitRate  int64   `json:"bitrate,omitempty"`
}

type MediaInfo struct {
	Video    *VideoInfo      `json:"video,omitempty"`
	Audio    []AudioTrack    `json:"audio,omitempty"`
	Subtitle []SubtitleTrack `json:"subtitle,omitempty"`
	Format   *FormatInfo     `json:"format,omitempty"`
}

func deriveResolution(width, height int) string {
	switch {
	case height >= 2160 || width >= 3840:
		return "2160p"
	case height >= 1440 || width >= 2560:
		return "1440p"
	case height >= 1080 || width >= 1920:
		return "1080p"
	case height >= 720 || width >= 1280:
		return "720p"
	case height >= 480 || width >= 854:
		return "480p"
	default:
		return ""
	}
}

func getTag(stream *ffprobe.Stream, key string) string {
	val, _ := stream.TagList.GetString(key)
	return val
}

func Probe(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	data, err := ffprobe.ProbeURL(ctx, url)
	if err != nil {
		return "", fmt.Errorf("ffprobe: %w", err)
	}

	mi := MediaInfo{}

	for _, stream := range data.Streams {
		switch stream.CodecType {
		case "video":
			if mi.Video == nil {
				mi.Video = &VideoInfo{
					Codec:      stream.CodecName,
					Profile:    stream.Profile,
					Resolution: deriveResolution(stream.Width, stream.Height),
				}
			}
		case "audio":
			mi.Audio = append(mi.Audio, AudioTrack{
				Codec:    stream.CodecName,
				Channels: stream.Channels,
				Layout:   stream.ChannelLayout,
				Language: getTag(stream, "language"),
			})
		case "subtitle":
			mi.Subtitle = append(mi.Subtitle, SubtitleTrack{
				Codec:    stream.CodecName,
				Language: getTag(stream, "language"),
				Title:    getTag(stream, "title"),
			})
		}
	}

	if data.Format != nil {
		fi := &FormatInfo{
			Name:     data.Format.FormatName,
			Duration: data.Format.DurationSeconds,
		}
		if size, err := strconv.ParseInt(data.Format.Size, 10, 64); err == nil {
			fi.Size = size
		}
		if br, err := strconv.ParseInt(data.Format.BitRate, 10, 64); err == nil {
			fi.BitRate = br
		}
		mi.Format = fi
	}

	b, err := json.Marshal(mi)
	if err != nil {
		return "", fmt.Errorf("marshal media info: %w", err)
	}

	return string(b), nil
}
