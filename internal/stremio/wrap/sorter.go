package stremio_wrap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func getResolutionRank(input string) int {
	if strings.HasSuffix(input, "P") {
		if resolution, err := strconv.Atoi(input[:len(input)-1]); err == nil {
			return resolution
		}
	}
	if strings.HasSuffix(input, "K") {
		if resolution, err := strconv.Atoi(input[:len(input)-1]); err == nil {
			return resolution * 1000
		}
	}
	return 0
}

func getQualityRank(input string) int {
	quality := strings.ToLower(input)

	if strings.Contains(quality, "remux") {
		return 100
	}
	if strings.Contains(quality, "mux") {
		return 99
	}

	if strings.Contains(quality, "bluray") {
		return 98
	}
	if strings.Contains(quality, "br") {
		return 96
	}
	if strings.Contains(quality, "bd") {
		return 94
	}
	if strings.Contains(quality, "uhd") {
		return 92
	}

	if strings.Contains(quality, "web") {
		if strings.Contains(quality, "dl") {
			return 89
		}
		if strings.Contains(quality, "rip") {
			return 85
		}
		return 80
	}

	if strings.Contains(quality, "hd") {
		if strings.Contains(quality, "rip") {
			return 79
		}
		return 75
	}

	if strings.Contains(quality, "dvd") {
		return 60
	}

	if strings.Contains(quality, "sat") {
		return 59
	}
	if strings.Contains(quality, "tv") {
		return 57
	}
	if strings.Contains(quality, "ppv") {
		return 55
	}

	if strings.Contains(quality, "cam") {
		return 40
	}
	if strings.Contains(quality, "tele") {
		return 30
	}
	if strings.Contains(quality, "scr") {
		return 20
	}
	return 0
}

func getSizeRank(input string) int64 {
	return util.ToBytes(input)
}

func getFieldRank(str *stremio_transformer.StreamExtractorResult, field string) any {
	switch field {
	case "resolution":
		return getResolutionRank(str.Resolution)
	case "quality":
		return getQualityRank(str.Quality)
	case "size":
		return getSizeRank(str.Size)
	default:
		panic("Unsupported field for sorting")
	}
}

type StreamSortConfig struct {
	Field string
	Desc  bool
}

func parseSortConfig(config string) []StreamSortConfig {
	sortConfigs := []StreamSortConfig{}
	for part := range strings.SplitSeq(config, ",") {
		part = strings.TrimSpace(part)
		desc := strings.HasPrefix(part, "-")
		field := strings.TrimPrefix(part, "-")
		if field == "resolution" || field == "quality" || field == "size" {
			sortConfigs = append(sortConfigs, StreamSortConfig{Field: field, Desc: desc})
		}
	}
	return sortConfigs
}

type StreamSorter struct {
	items  []WrappedStream
	config []StreamSortConfig
}

func (ss StreamSorter) Len() int {
	return len(ss.items)
}
func (ss StreamSorter) Swap(i, j int) {
	ss.items[i], ss.items[j] = ss.items[j], ss.items[i]
}

func (ss StreamSorter) Less(a, b int) bool {
	aData, bData := ss.items[a].r, ss.items[b].r
	if bData == nil {
		return true
	} else if aData == nil {
		return false
	}

	for _, config := range ss.config {
		va := getFieldRank(aData, config.Field)
		vb := getFieldRank(bData, config.Field)

		if va == vb {
			continue
		}

		switch vaTyped := va.(type) {
		case int:
			vbTyped := vb.(int)
			if config.Desc {
				return vaTyped > vbTyped
			}
			return vaTyped < vbTyped
		case int64:
			vbTyped := vb.(int64)
			if config.Desc {
				return vaTyped > vbTyped
			}
			return vaTyped < vbTyped
		default:
			panic(fmt.Sprintf("Unsupported field rank type for sorting: %T", va))
		}
	}
	return false
}

func SortWrappedStreams(items []WrappedStream, config string) {
	if config == "" {
		config = "-resolution,-quality,-size"
	}

	sortConfigs := parseSortConfig(config)
	if len(sortConfigs) == 0 {
		return
	}
	sorter := StreamSorter{items: items, config: sortConfigs}
	sort.Sort(sorter)
}
