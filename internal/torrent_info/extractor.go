package torrent_info

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/stremio"
)

var torrentioStreamHashRegex = regexp.MustCompile(`(?i)\/([a-f0-9]{40})\/[^/]+\/(?:(\d+)|null|undefined)\/`)
var torrentioStreamSeedersRegex = regexp.MustCompile(`👤 (\d+)`)
var torrentioStreamSizeRegex = regexp.MustCompile(`💾 (?:([\d.]+ [^ ]+)|.+?)`)
var torrentioDebridTrustedFileIndexRegex = regexp.MustCompile(`\[(?:RD|DL)`)

func isTorrentioDebridFileIndexTrustable(name string) bool {
	return torrentioDebridTrustedFileIndexRegex.MatchString(name)
}

func extractInputFromTorrentioStream(data *TorrentInfoInsertData, sid string, stream *stremio.Stream) *TorrentInfoInsertData {
	description := stream.Description
	if description == "" {
		description = stream.Title
	}
	torrentTitle, descriptionRest, _ := strings.Cut(description, "\n")
	data.TorrentTitle = torrentTitle
	file := TorrentInfoInsertDataFile{
		Idx:  -1,
		Size: -1,
		SId:  sid,
	}

	if stream.BehaviorHints != nil && stream.BehaviorHints.Filename != "" {
		file.Name = stream.BehaviorHints.Filename
	} else if descriptionRest != "" && !strings.HasPrefix(descriptionRest, "👤") {
		file.Name, _, _ = strings.Cut(descriptionRest, "\n")
		file.Name = filepath.Base(file.Name)
	}
	if stream.InfoHash == "" {
		if match := torrentioStreamHashRegex.FindStringSubmatch(stream.URL); len(match) > 0 {
			data.Hash = match[1]
			if isTorrentioDebridFileIndexTrustable(stream.Name) && len(match) > 2 {
				if idx, err := strconv.Atoi(match[2]); err == nil {
					file.Idx = idx
				}
			}
		}
	} else {
		data.Hash = stream.InfoHash
		file.Idx = stream.FileIndex
	}
	if match := torrentioStreamSeedersRegex.FindStringSubmatch(description); len(match) > 1 {
		data.Seeders = util.SafeParseInt(match[1], -1)
	}
	if match := torrentioStreamSizeRegex.FindStringSubmatch(description); len(match) > 1 {
		file.Size = util.ToBytes(match[1])
	}
	if file.Name != "" {
		data.Files = append(data.Files, file)
	}
	data.Size = -1
	return data
}

var mediafusionStreamHashRegex = regexp.MustCompile(`(?i)\/stream\/([a-f0-9]{40})(?:\/|$)`)
var mediafusionStreamSeedersRegex = regexp.MustCompile(`👤 (\d+)`)
var mediafusionStreamSizeRegex = regexp.MustCompile(`💾 ([\d.]+ [A-Z]B)(?: \/ 💾 ([\d.]+ [A-Z]B))?`)

func extractInputFromMediaFusionStream(data *TorrentInfoInsertData, sid string, stream *stremio.Stream) *TorrentInfoInsertData {
	data.Size = -1
	file := TorrentInfoInsertDataFile{
		Idx:  -1,
		Size: -1,
		SId:  sid,
	}

	torrentTitle, descriptionRest, _ := strings.Cut(stream.Description, "\n")
	if folderName, ok := strings.CutPrefix(torrentTitle, "📂 "); ok {
		torrentTitle = folderName
		data.TorrentTitle, file.Name, _ = strings.Cut(torrentTitle, " ┈➤ ")
	}

	if stream.BehaviorHints != nil {
		if stream.BehaviorHints.Filename != "" && stream.BehaviorHints.Filename != data.TorrentTitle {
			file.Name = stream.BehaviorHints.Filename
		}
		if stream.BehaviorHints.VideoSize > 0 {
			file.Size = stream.BehaviorHints.VideoSize
		}
	}

	if stream.InfoHash == "" {
		if match := mediafusionStreamHashRegex.FindStringSubmatch(stream.URL); len(match) > 0 {
			data.Hash = match[1]
		}
	} else {
		data.Hash = stream.InfoHash
		file.Idx = stream.FileIndex
	}
	if match := mediafusionStreamSeedersRegex.FindStringSubmatch(descriptionRest); len(match) > 1 {
		data.Seeders = util.SafeParseInt(match[1], -1)
	}
	if match := mediafusionStreamSizeRegex.FindStringSubmatch(descriptionRest); len(match) > 0 {
		if file.Size == -1 {
			file.Size = util.ToBytes(match[1])
		}
		if len(match) > 2 {
			data.Size = util.ToBytes(match[2])
		}
	}
	if core.HasVideoExtension(file.Name) {
		data.Files = append(data.Files, file)
	}

	return data
}

func ExtractCreateDataFromStream(hostname string, sid string, stream *stremio.Stream) *TorrentInfoInsertData {
	data := &TorrentInfoInsertData{}
	switch hostname {
	case "torrentio.strem.fun":
		data.Source = TorrentInfoSourceTorrentio
		data = extractInputFromTorrentioStream(data, sid, stream)
	case "mediafusion.elfhosted.com":
		data.Source = TorrentInfoSourceMediaFusion
		data = extractInputFromMediaFusionStream(data, sid, stream)
	}
	if data.Hash == "" {
		return nil
	}
	return data
}
