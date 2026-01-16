package usenet_pool

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

type archiveVolume struct {
	n    int
	name string
}

type archiveVolumeGroup struct {
	BaseName  string   // e.g., "video" for video.part01.rar, video.part02.rar
	FileType  FileType // RAR or 7z
	Files     []ArchiveFile
	TotalSize int64
}

func (g *archiveVolumeGroup) GetFirstVolumeName() string {
	if len(g.Files) == 0 {
		return ""
	}
	return filepath.Base(g.Files[0].Name())
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

func groupArchiveVolumes(files []ArchiveFile) []archiveVolumeGroup {
	groups := make(map[string]*archiveVolumeGroup)

	for _, f := range files {
		baseName, fileType := getArchiveBaseName(f.Name())
		if fileType == FileTypePlain {
			continue
		}

		key := baseName + ":" + fileType.String()
		if g, ok := groups[key]; ok {
			g.Files = append(g.Files, f)
			g.TotalSize += f.UnPackedSize()
		} else {
			groups[key] = &archiveVolumeGroup{
				BaseName:  baseName,
				FileType:  fileType,
				Files:     []ArchiveFile{f},
				TotalSize: f.UnPackedSize(),
			}
		}
	}

	result := make([]archiveVolumeGroup, 0, len(groups))
	for _, group := range groups {
		slices.SortStableFunc(group.Files, func(a, b ArchiveFile) int {
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

	slices.SortStableFunc(result, func(a, b archiveVolumeGroup) int {
		return cmp.Compare(b.TotalSize, a.TotalSize)
	})

	return result
}
