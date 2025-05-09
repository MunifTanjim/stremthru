package stremio_transformer

import "strings"

var StreamExtractorTorrentio = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^(?:\[(?<store_code>\w+?)(?:(?<store_is_cached>\+?)| download)\] )?(?<addon_name>\w+)(?:\n(?:(?<resolution>\d+[kp])? ?)?(?:(?<quality>` + qualityPattern + `)? ?)?(?:(?:3D(?: SBS)) ?)?(?<hdr>[^| ]+(?:(?<hdr_sep> \| )[^| ]+)*)?)?

bingeGroup
(?i)(?<codec>` + codecPattern + `)
(?i)(?<bitdepth>\d+bit)
(?i)(?<quality>` + qualityPattern + `)

filename
(?i)(?<codec>` + codecPattern + `)

description
^(?<t_title>.+)\n(?:(?<file_name>[^👤].+)\n)?👤.+ 💾 (?<size>.+) ⚙️ (?<site>\w+)(?:\n(?<language>[^\/]+(?:(?<language_sep>\/)[^\/]+)*))?$
(?i)(?<quality>` + qualityPattern + `)

url
(?i)\/(?<hash>[a-f0-9]{40})\/[^/]+\/(?:(?<file_idx>\d+)|null|undefined)\/
`)).MustParse()
