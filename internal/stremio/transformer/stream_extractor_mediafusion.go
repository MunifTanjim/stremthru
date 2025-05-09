package stremio_transformer

import "strings"

var StreamExtractorMediaFusion = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^(?<addon_name>\w+(?: \| [^ ]+)?) (?:P2P|(?<store_code>[A-Z]{2,3})) (?:N\/A|(?<resolution>[^kp]+[kp])) (?<store_is_cached>⚡️)?

description
(?i)(?:📂 (?<t_title>.+?)(?: ┈➤ (?<file_name>.+))?\n)?(?:(?:📺 .+)?(?: 🎞️ .+)?(?: 🎵 .+)?\n)?💾 (?:(?<file_size>.+?) \/ 💾 )?(?<size>.+?)(?: 👤 \d+)?\n(?:.+\n)?🔗 (?<site>.+?)(?: 🧑‍💻 |$)

bingeGroup
(?i)-(?:🎨 (?<hdr>[^| ]+(?:(?<hdr_sep>\|)[^| ]+)*) )?📺 (?<quality>` + qualityPattern + `)(?: ?🎞️ (?<codec>[^- ]+))?(?: ?🎵 .+)?-(?:N\/A|(?:\d+[kp]))

filename
(?i)(?<quality>` + qualityPattern + `)
(?i)(?<codec>` + codecPattern + `)

url
\/stream\/(?<hash>[a-f0-9]{40})(?:\/(?<season>\d+)\/(?<episode>\d+)\/?)?
`)).MustParse()
