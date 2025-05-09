package stremio_transformer

import "strings"

var StreamExtractorComet = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^\[(?:TORRENT🧲|(?<store_code>\w+)(?:(?<store_is_cached>⚡)|⬇️)?)\] (?<addon_name>.+) (?:unknown|(?<resolution>\d[^kp]*[kp]))

description
^(?<t_title>.+)\n(?:💿 .+\n)?(?:👤 \d+ )?💾 (?:(?<size>[\d.]+ [^ ]+)|.+?) 🔎 (?<site>.+)(?:\n(?<language>[^/]+(?:(?<language_sep>\/)[^/]+)*))?
(?i)💿 (?:.+\|)?(?<quality>` + qualityPattern + `)
(?i)💿 (?:.+\|)?(?<codec>` + codecPattern + `)

url
\/playback\/(?<hash>[a-f0-9]{40})\/(?:n|(?<file_idx>\d+))\/[^/]+\/(?:n|(?<season>\d+))\/(?:n|(?<episode>\d+))\/(?<file_name>.+)
`)).MustParse()
