package torrent_info

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var torrentioStreamHashRegex = regexp.MustCompile(`(?i)\/([a-f0-9]{40})\/[^/]+\/(?:(\d+)|null|undefined)\/`)
var torrentioStreamSizeRegex = regexp.MustCompile(`💾 (?:([\d.]+ [^ ]+)|.+?)`)

func extractInputFromTorrentioStream(data *TorrentInfoInsertData, stream *stremio.Stream) *TorrentInfoInsertData {
	description := stream.Description
	if description == "" {
		description = stream.Title
	}
	torrentTitle, descriptionRest, _ := strings.Cut(description, "\n")
	data.TorrentTitle = torrentTitle
	if stream.BehaviorHints != nil && stream.BehaviorHints.Filename != "" {
		data.File.Name = stream.BehaviorHints.Filename
	} else if descriptionRest != "" && !strings.HasPrefix(descriptionRest, "👤") {
		data.File.Name, _, _ = strings.Cut(descriptionRest, "\n")
	}
	if stream.InfoHash == "" {
		if match := torrentioStreamHashRegex.FindStringSubmatch(stream.URL); len(match) > 0 {
			data.Hash = match[1]
			if len(match) > 2 {
				if idx, err := strconv.Atoi(match[2]); err == nil {
					data.File.Idx = idx
				}
			}
		}
	} else {
		data.Hash = stream.InfoHash
		data.File.Idx = stream.FileIndex
	}
	data.File.Size = int64(-1)
	if match := torrentioStreamSizeRegex.FindStringSubmatch(description); len(match) > 1 {
		data.File.Size = shared.ToBytes(match[1])
	}
	data.Size = -1
	return data
}

func ExtractCreateDataFromStream(hostname string, stream *stremio.Stream) *TorrentInfoInsertData {
	data := &TorrentInfoInsertData{}
	switch hostname {
	case "torrentio.strem.fun":
		data.Source = TorrentInfoSourceTorrentio
		data = extractInputFromTorrentioStream(data, stream)
	}
	if data.Hash == "" || data.TorrentTitle == "" || data.File.Name == "" {
		return nil
	}
	return data
}
