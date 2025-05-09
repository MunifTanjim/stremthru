package stremio_transformer

import "strings"

var StreamExtractorDebridio = StreamExtractorBlob(strings.TrimSpace(`
name
(?i)^(?:\[(?<store_code>\w+?)(?<store_is_cached>\+?)\] \n)?(?<addon_name>\w+) (?:Other|(?<resolution>\d[^kp]*[kp]))

description
^(?<t_title>.+?) ?\n(?:(?<file_name>.+?) ?\n)?⚡? 📺 (?<resolution>[^ ]+) 💾 (?:Unknown|(?<size>[\d.]+ [^ ]+)|.+?) (?:👤 (?:Unknown|\d+))? ⚙️ (?<site>[^ ]+)(?: \n🌐 (?<language>[^|]+(?:(?<language_sep>\|)[^|]+)*))?

url
\/(?<hash>[a-f0-9]{40})(?:\/(?<season>\d+)\/(?<episode>\d+))?
`)).MustParse()
