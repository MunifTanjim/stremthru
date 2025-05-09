package stremio_transformer

import "strings"

var StreamExtractorOrion = StreamExtractorBlob(strings.TrimSpace(`
name
(?:🪐 (?<addon_name>\w+) 📺 (?<resolution>\w+))|(?:(?<store_is_cached>🚀) (?<addon_name>\w+)\n.*\[(?<store_name>[^\]]+)\])

description
(?<t_title>.+)\n(?:📺(?<resolution>.+?) )?💾(?<size>[0-9.]+ [^ ]+) (?:👤\d+ )?🎥(?<codec>\w+) 🔊(?:(?<channel>\d\.\d)|.+)\n👂(?<language>[A-Z]+(?:(?<language_sep> )[A-Z]+)*) ☁️(?<site>.+)
`)).MustParse()
