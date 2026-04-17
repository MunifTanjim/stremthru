package stremio_transformer

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/stretchr/testify/assert"
)

func TestStreamTemplateDefault(t *testing.T) {
	newzDate := time.Now().Add(-3 * time.Hour)

	makeData := func() *StreamExtractorResult {
		return &StreamExtractorResult{
			Result: &ptt.Result{
				Resolution: "1080p",
				Quality:    "BluRay",
				Codec:      "x265",
				HDR:        []string{"HDR10", "DV"},
				Audio:      []string{"DDP"},
				Channels:   []string{"5.1"},
				Size:       "2.5 GB",
				Group:      "GROUP",
				Site:       "example.com",
				Languages:  []string{"en", "ja"},
			},
			Addon: StreamExtractorResultAddon{
				Name: "Addon",
			},
			File: StreamExtractorResultFile{
				Name: "movie.mkv",
				Size: "2.4 GB",
			},
			Indexer: StreamExtractorResultIndexer{
				Name: "Indexer",
			},
			BitRate:   12_000_000,
			IsPrivate: true,
			Kind:      StreamExtractorResultKindTorz,
			Seeders:   42,
			Store: StreamExtractorResultStore{
				Code:      "RD",
				IsCached:  true,
				IsProxied: true,
			},
			Subtitles: []string{"en"},
		}
	}

	for _, tc := range []struct {
		name                string
		prepare             func(*StreamExtractorResult)
		expectedName        string
		expectedDescription func(data *StreamExtractorResult) string
	}{
		{
			name:         "torz w/ full data",
			expectedName: "✨ ⚡️ [RD] 🔑\nAddon\n1080p",
			expectedDescription: func(*StreamExtractorResult) string {
				return strings.TrimSpace(`
💿 BluRay 🎞️ x265
📺 HDR10 DV 🎧 DDP | 5.1
💾 2.4 GB 📦 2.5 GB 〽️ 1.5 MB/s 👤 42
🎙️ 🇬🇧 🇯🇵
💬 🇬🇧
⚙️ GROUP 🔍 Indexer
📄 movie.mkv
`)
			},
		},
		{
			name: "newz w/ minimal data",
			prepare: func(d *StreamExtractorResult) {
				d.Result = &ptt.Result{
					Resolution: "720p",
					Size:       "500 MB",
					Site:       "usenet.example",
				}
				d.BitRate = 0
				d.File = StreamExtractorResultFile{}
				d.Indexer = StreamExtractorResultIndexer{}
				d.IsPrivate = false
				d.Seeders = 0
				d.Store = StreamExtractorResultStore{}
				d.Subtitles = nil
				d.Date = newzDate
				d.Kind = StreamExtractorResultKindNewz
				d.TTitle = "The Movie Title"
			},
			expectedName: "Addon\n720p",
			expectedDescription: func(data *StreamExtractorResult) string {
				return strings.TrimSpace(fmt.Sprintf(`
📦 500 MB ⏱️ %s
🔗 usenet.example
📁 The Movie Title
`, data.Age()))
			},
		},
		{
			name: "torz w/o group, site, indexer",
			prepare: func(d *StreamExtractorResult) {
				d.Result.Group = ""
				d.Result.Site = ""
				d.Indexer = StreamExtractorResultIndexer{}
			},
			expectedName: "✨ ⚡️ [RD] 🔑\nAddon\n1080p",
			expectedDescription: func(*StreamExtractorResult) string {
				return strings.TrimSpace(`
💿 BluRay 🎞️ x265
📺 HDR10 DV 🎧 DDP | 5.1
💾 2.4 GB 📦 2.5 GB 〽️ 1.5 MB/s 👤 42
🎙️ 🇬🇧 🇯🇵
💬 🇬🇧
📄 movie.mkv
`)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := makeData()
			if tc.prepare != nil {
				tc.prepare(data)
			}
			stream, err := StreamTemplateDefault.Execute(&stremio.Stream{}, data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedName, stream.Name)
			assert.Equal(t, tc.expectedDescription(data), stream.Description)
		})
	}
}
