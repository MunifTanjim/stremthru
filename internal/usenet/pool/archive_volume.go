package usenet_pool

import (
	"cmp"
	"slices"
	"strings"
)

type archiveVolume struct {
	n    int
	name string
}

type archiveVolumeGroup[T any] struct {
	BaseName  string   // e.g., "video" for video.part01.rar, video.part02.rar
	FileType  FileType // RAR or 7z
	Files     []T
	TotalSize int64
}

func getArchiveBaseName(filename string) (baseName string, fileType FileType) {
	lower := strings.ToLower(filename)

	if matches := rarPartNumberRegex.FindStringSubmatch(lower); len(matches) > 0 {
		return filename[:len(filename)-len(matches[0])], FileTypeRAR
	}

	if matches := rarRNumberRegex.FindStringSubmatch(lower); len(matches) > 0 {
		return filename[:len(filename)-len(matches[0])], FileTypeRAR
	}

	if matches := rarFirstPartRegex.FindStringSubmatch(lower); len(matches) > 0 {
		return filename[:len(filename)-len(matches[0])], FileTypeRAR
	}

	if matches := sevenzipPartNumberRegex.FindStringSubmatch(lower); len(matches) > 0 {
		return filename[:len(filename)-len(matches[0])], FileType7z
	}

	if matches := sevenzipFirstPartRegex.FindStringSubmatch(lower); len(matches) > 0 {
		return filename[:len(filename)-len(matches[0])], FileType7z
	}

	return "", FileTypePlain
}

type simpleFile interface {
	Name() string
	Size() int64
}

func groupArchiveVolumes[T simpleFile](
	files []T,
) []archiveVolumeGroup[T] {
	groups := make(map[string]*archiveVolumeGroup[T])

	for _, f := range files {
		baseName, fileType := getArchiveBaseName(f.Name())
		if fileType == FileTypePlain {
			continue
		}

		key := baseName + ":" + fileType.String()
		if g, ok := groups[key]; ok {
			g.Files = append(g.Files, f)
			g.TotalSize += f.Size()
		} else {
			groups[key] = &archiveVolumeGroup[T]{
				BaseName:  baseName,
				FileType:  fileType,
				Files:     []T{f},
				TotalSize: f.Size(),
			}
		}
	}

	result := make([]archiveVolumeGroup[T], 0, len(groups))
	for _, group := range groups {
		slices.SortStableFunc(group.Files, func(a, b T) int {
			var volA, volB int
			switch group.FileType {
			case FileTypeRAR:
				volA = GetRARVolumeNumber(a.Name())
				volB = GetRARVolumeNumber(b.Name())
			case FileType7z:
				volA = Get7zVolumeNumber(a.Name())
				volB = Get7zVolumeNumber(b.Name())
			}
			return volA - volB
		})
		result = append(result, *group)
	}

	slices.SortStableFunc(result, func(a, b archiveVolumeGroup[T]) int {
		return cmp.Compare(b.TotalSize, a.TotalSize)
	})

	return result
}
