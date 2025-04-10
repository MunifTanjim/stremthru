package stremio_wrap

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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

	if strings.Contains(quality, "mux") {
		return 100
	}

	if strings.Contains(quality, "bluray") {
		return 99
	}
	if strings.Contains(quality, "br") {
		return 97
	}
	if strings.Contains(quality, "bd") {
		return 95
	}
	if strings.Contains(quality, "uhd") {
		return 93
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

func (str StreamTransformerResult) GetFieldRank(field string) any {
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
	for _, part := range strings.Split(config, ",") {
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
	for _, config := range ss.config {
		va := ss.items[a].r.GetFieldRank(config.Field)
		vb := ss.items[b].r.GetFieldRank(config.Field)

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
