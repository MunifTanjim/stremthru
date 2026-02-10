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
	Codec      string `json:"c,omitempty"`
	Resolution string `json:"r,omitempty"`
}

type AudioTrack struct {
	Codec    string `json:"c,omitempty"`
	Channels int    `json:"ch,omitempty"`
	Layout   string `json:"cl,omitempty"`
	Language string `json:"l,omitempty"`
}

type SubtitleTrack struct {
	Codec    string `json:"c,omitempty"`
	Language string `json:"l,omitempty"`
}

type FormatInfo struct {
	Name     string  `json:"n,omitempty"`
	Duration float64 `json:"d,omitempty"`
	Size     int64   `json:"s,omitempty"`
	BitRate  int64   `json:"br,omitempty"`
}

type MediaInfo struct {
	Video    *VideoInfo      `json:"v,omitempty"`
	Audio    []AudioTrack    `json:"a,omitempty"`
	Subtitle []SubtitleTrack `json:"s,omitempty"`
	Format   *FormatInfo     `json:"f,omitempty"`
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

func getLanguage(stream *ffprobe.Stream) string {
	lang, _ := stream.TagList.GetString("language")
	return lang
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
					Resolution: deriveResolution(stream.Width, stream.Height),
				}
			}
		case "audio":
			mi.Audio = append(mi.Audio, AudioTrack{
				Codec:    stream.CodecName,
				Channels: stream.Channels,
				Layout:   stream.ChannelLayout,
				Language: getLanguage(stream),
			})
		case "subtitle":
			mi.Subtitle = append(mi.Subtitle, SubtitleTrack{
				Codec:    stream.CodecName,
				Language: getLanguage(stream),
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
