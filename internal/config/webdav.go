package config

import (
	"strings"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type extFilter struct {
	*util.Set[string]
	raw []string
}

func parseWebDAVExtFilter(value string) *extFilter {
	result := extFilter{
		Set: util.NewSet[string](),
		raw: []string{},
	}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch part {
		case ":video:":
			result.raw = append(result.raw, part)

			for ext := range util.FileExtVideo.Seq() {
				result.Add(ext)
			}
		case ":subtitle:":
			result.raw = append(result.raw, part)

			for ext := range util.FileExtSubtitle.Seq() {
				result.Add(ext)
			}
		default:
			result.raw = append(result.raw, part)

			exclude := strings.HasPrefix(part, "-")
			if exclude {
				part = part[1:]
			}

			ext := part
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if exclude {
				result.Del(strings.ToLower(ext))
			} else {
				result.Add(strings.ToLower(ext))
			}
		}
	}
	return &result
}

var WebDAVFileExtFilter = parseWebDAVExtFilter(getEnv("STREMTHRU_WEBDAV_FILE_EXT_FILTER"))
