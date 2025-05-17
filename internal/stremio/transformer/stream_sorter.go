package stremio_transformer

import (
	"sort"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type StreamSortableField string

const (
	StreamSortableFieldResolution StreamSortableField = "resolution"
	StreamSortableFieldQuality    StreamSortableField = "quality"
	StreamSortableFieldSize       StreamSortableField = "size"
)

type StreamSortable interface {
	GetQuality() string
	GetResolution() string
	GetSize() string
	IsSortable() bool
}

func getQualityRank(input string) int64 {
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

func getResolutionRank(input string) int64 {
	if strings.HasSuffix(input, "p") {
		if resolution, err := strconv.Atoi(input[:len(input)-1]); err == nil {
			return int64(resolution)
		}
	}
	if strings.HasSuffix(input, "k") {
		if resolution, err := strconv.Atoi(input[:len(input)-1]); err == nil {
			return int64(resolution * 1000)
		}
	}
	return 0
}

func getSizeRank(input string) int64 {
	return util.ToBytes(input)
}

func getFieldRank(str StreamSortable, field StreamSortableField) int64 {
	switch field {
	case StreamSortableFieldResolution:
		return getResolutionRank(str.GetResolution())
	case StreamSortableFieldQuality:
		return getQualityRank(str.GetQuality())
	case StreamSortableFieldSize:
		return getSizeRank(str.GetSize())
	default:
		panic("Unsupported field for sorting")
	}
}

type StreamSorterConfig struct {
	Field StreamSortableField
	Desc  bool
}

func parseSortConfig(config string) []StreamSorterConfig {
	sortConfigs := []StreamSorterConfig{}
	for part := range strings.SplitSeq(config, ",") {
		part = strings.TrimSpace(part)
		desc := strings.HasPrefix(part, "-")
		field := StreamSortableField(strings.TrimPrefix(part, "-"))
		switch field {
		case StreamSortableFieldResolution, StreamSortableFieldQuality, StreamSortableFieldSize:
			sortConfigs = append(sortConfigs, StreamSorterConfig{Field: field, Desc: desc})
		}
	}
	return sortConfigs
}

type streamSorter[T StreamSortable] struct {
	items  []T
	config []StreamSorterConfig
}

func (ss streamSorter[StreamSortable]) Len() int {
	return len(ss.items)
}
func (ss streamSorter[StreamSortable]) Swap(i, j int) {
	ss.items[i], ss.items[j] = ss.items[j], ss.items[i]
}

func (ss streamSorter[StreamSortable]) Less(a, b int) bool {
	aData, bData := ss.items[a], ss.items[b]
	if !bData.IsSortable() {
		return true
	} else if !aData.IsSortable() {
		return false
	}

	for _, config := range ss.config {
		va := getFieldRank(aData, config.Field)
		vb := getFieldRank(bData, config.Field)

		if va == vb {
			continue
		}

		if config.Desc {
			return va > vb
		}

		return va < vb
	}
	return false
}

const StreamDefaultSortConfig = "-resolution,-quality,-size"

func SortStreams[T StreamSortable](items []T, config string) {
	if config == "" {
		config = StreamDefaultSortConfig
	}

	sortConfigs := parseSortConfig(config)
	if len(sortConfigs) == 0 {
		return
	}
	sorter := streamSorter[T]{items: items, config: sortConfigs}
	sort.Sort(sorter)
}
