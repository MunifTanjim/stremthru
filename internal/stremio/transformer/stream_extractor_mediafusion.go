package stremio_transformer

import "strings"

var StreamExtractorMediaFusion = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^(?<addon_name>\w+(?: \| [^ ]+)?) 🧲 (?:P2P|(?<store_code>[A-Z]{2,3})) (?:⏳|(?<store_is_cached>⚡️)) (?:N\/A|(?<resolution>\d+[kp]))?

description
(?i)(?:🎨 (?<hdr>[^| ]+(?:(?<hdr_sep>\|)[^| ]+)*) )?(?:📺 (?<quality>` + qualityPattern + `) )?(?:🎞️ (?<codec>[^- ]+) )?(?:🎵 .+ )?(?: 🔊 .+)?\n(?:📦 (?:(?<file_size>.+?) \/ )?(?<size>.+?) )?(?:👤 (?<seeders>\d+))?\n(?:🌐 (?<language>[^\+]+(?:(?<language_sep>\+)[^\+]+)*))?\n🔗 (?<site>.+?)(?: 🧑‍💻 .+$|$)

url
\/(?:stream|playback)\/[^\/]+\/(?<hash>[a-f0-9]{40})(?:\/(?<season>\d+)\/(?<episode>\d+)\/?)?
`)).MustParse()
