package stremio_transformer

import "strings"

var StreamExtractorPeerflix = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^(?:\[(?<store_code>\w+?)(?:(?<store_is_cached>\+?)|\s[^\]]+)\] )?(?<addon_name>\w+) \S+ (?:\w+-)?(?<resolution>\d+[kp])?

description
^(?<t_title>[^\n]+)\n(?:(?<file_name>.+)\n)?.+👤 \d+ (?:💾 (?<size>[\d.]+ \w[bB]) )?🌐 (?<site>\w+)$

url
(?i)\/(?<hash>[a-f0-9]{40})\/[^/]+\/(?:(?<file_idx>\d+)|null|undefined)\/
`)).MustParse()
