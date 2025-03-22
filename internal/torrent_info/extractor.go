package torrent_info

import (
	"regexp"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/stremio"
)

var torrentioStreamHashRegex = regexp.MustCompile(`(?i)\/([a-f0-9]{40})\/[^/]+\/\d+\/`)
var torrentioStreamSizeRegex = regexp.MustCompile(`💾 (?:([\d.]+ [^ ]+)|.+?)`)

func extractInputFromTorrentioStream(data *TorrentInfoInsertData, stream *stremio.Stream) *TorrentInfoInsertData {
	data.TorrentTitle, _, _ = strings.Cut(stream.Title, "\n")
	data.Hash = stream.InfoHash
	if data.Hash == "" {
		if match := torrentioStreamHashRegex.FindStringSubmatch(stream.URL); len(match) > 0 {
			data.Hash = match[1]
		}
	}
	data.Size = int64(-1)
	if match := torrentioStreamSizeRegex.FindStringSubmatch(stream.Title); len(match) > 0 {
		data.Size = shared.ToBytes(match[1])
	}
	return data
}

func ExtractCreateDataFromStream(hostname string, stream *stremio.Stream) *TorrentInfoInsertData {
	data := &TorrentInfoInsertData{}
	switch hostname {
	case "torrentio.strem.fun":
		data.Source = TorrentInfoSourceTorrentio
		return extractInputFromTorrentioStream(data, stream)
	default:
		return nil
	}
}
