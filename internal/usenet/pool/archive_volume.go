package usenet_pool

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type archiveVolume struct {
	n    int
	name string
}

type archiveVolumeGroup[T any] struct {
	BaseName  string   // e.g., "video" for video.part01.rar, video.part02.rar
	Aliased   bool     // no standard archive extension
	FileType  FileType // RAR or 7z
	Files     []T
	Volumes   []int
	TotalSize int64
}

var trailingNumbersRegex = regexp.MustCompile(`\.\d+$`)

func stripTrailingNumbers(filename string) string {
	if loc := trailingNumbersRegex.FindStringIndex(filename); loc != nil {
		return filename[:loc[0]]
	}
	return filename
}

func normalizeRARPartNames[T simpleFile](files []T) map[string]string {
	type partInfo struct {
		index  int
		digits string
		prefix string
		suffix string
	}

	var parts []partInfo
	for i, f := range files {
		m := rarPartNumberRegex.FindStringSubmatchIndex(f.Name())
		if m == nil {
			continue
		}
		digitStr := f.Name()[m[2]:m[3]]
		prefix := f.Name()[:m[2]]
		suffix := f.Name()[m[3]:]
		parts = append(parts, partInfo{
			index:  i,
			digits: digitStr,
			prefix: prefix,
			suffix: suffix,
		})
	}

	if len(parts) == 0 {
		return nil
	}

	maxWidth := 0
	for _, p := range parts {
		maxWidth = max(len(p.digits), maxWidth)
	}

	allSame := true
	for _, p := range parts {
		if len(p.digits) != maxWidth {
			allSame = false
			break
		}
	}
	if allSame {
		return nil
	}

	aliases := make(map[string]string)
	for _, p := range parts {
		if len(p.digits) == maxWidth {
			continue
		}
		normalized := fmt.Sprintf("%s%0*s%s", p.prefix, maxWidth, p.digits, p.suffix)
		aliases[normalized] = files[p.index].Name()
	}

	return aliases
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

type typedArchiveFile interface {
	FileType() FileType
	Volume() int
}

func getFileVolume[T simpleFile](f T, fileType FileType) int {
	if tf, ok := any(f).(typedArchiveFile); ok {
		return tf.Volume()
	}
	switch fileType {
	case FileTypeRAR:
		return GetRARVolumeNumber(f.Name())
	case FileType7z:
		return Get7zVolumeNumber(f.Name())
	default:
		return -1
	}
}

func groupArchiveVolumes[T simpleFile](
	files []T,
) []archiveVolumeGroup[T] {
	groups := make(map[string]*archiveVolumeGroup[T])

	for _, f := range files {
		baseName, fileType := getArchiveBaseName(f.Name())
		aliased := false
		if fileType == FileTypePlain {
			if tf, ok := any(f).(typedArchiveFile); ok && tf.FileType() != FileTypePlain {
				fileType = tf.FileType()
				baseName = stripTrailingNumbers(f.Name())
				aliased = true
			} else {
				continue
			}
		}

		key := baseName + ":" + fileType.String()
		if g, ok := groups[key]; ok {
			g.Files = append(g.Files, f)
			g.TotalSize += f.Size()
		} else {
			groups[key] = &archiveVolumeGroup[T]{
				BaseName:  baseName,
				Aliased:   aliased,
				FileType:  fileType,
				Files:     []T{f},
				TotalSize: f.Size(),
			}
		}
	}

	// Post-processing: merge single-file aliased RAR groups if their volumes form a 0-based sequence.
	// This handles the case where RAR volumes have completely random extensionless names.
	type groupWithVol struct {
		group *archiveVolumeGroup[T]
		key   string
		vol   int
	}
	var singleFileAliasedRAR []groupWithVol
	for key, group := range groups {
		if group.FileType == FileTypeRAR && len(group.Files) == 1 {
			singleFileAliasedRAR = append(singleFileAliasedRAR, groupWithVol{
				group: group,
				key:   key,
				vol:   getFileVolume(group.Files[0], group.FileType),
			})
		}
	}

	if len(singleFileAliasedRAR) > 0 {
		slices.SortStableFunc(singleFileAliasedRAR, func(a, b groupWithVol) int {
			return cmp.Compare(a.vol, b.vol)
		})

		// Check if volumes form a strict 0-based contiguous sequence
		valid := true
		for i, gv := range singleFileAliasedRAR {
			if gv.vol != i {
				valid = false
				break
			}
		}

		if valid {
			baseName := singleFileAliasedRAR[0].group.BaseName

			mergedFiles := make([]T, len(singleFileAliasedRAR))
			mergedVolumes := make([]int, len(singleFileAliasedRAR))
			var totalSize int64
			for i, gv := range singleFileAliasedRAR {
				delete(groups, gv.key)
				mergedFiles[i] = gv.group.Files[0]
				mergedVolumes[i] = gv.vol
				totalSize += gv.group.TotalSize
			}

			mergedKey := baseName + ":" + FileTypeRAR.String()
			groups[mergedKey] = &archiveVolumeGroup[T]{
				BaseName:  baseName,
				Aliased:   true,
				FileType:  FileTypeRAR,
				Files:     mergedFiles,
				Volumes:   mergedVolumes,
				TotalSize: totalSize,
			}
		}
	}

	result := make([]archiveVolumeGroup[T], 0, len(groups))
	for _, group := range groups {
		if group.Volumes != nil {
			result = append(result, *group)
			continue
		}

		type indexedVolume struct {
			index  int
			volume int
		}
		ivs := make([]indexedVolume, len(group.Files))
		for i, f := range group.Files {
			ivs[i] = indexedVolume{index: i, volume: getFileVolume(f, group.FileType)}
		}
		slices.SortStableFunc(ivs, func(a, b indexedVolume) int {
			return cmp.Compare(a.volume, b.volume)
		})
		sorted := make([]T, len(group.Files))
		volumes := make([]int, len(group.Files))
		for i, iv := range ivs {
			sorted[i] = group.Files[iv.index]
			volumes[i] = iv.volume
		}
		group.Files = sorted
		group.Volumes = volumes
		result = append(result, *group)
	}

	slices.SortStableFunc(result, func(a, b archiveVolumeGroup[T]) int {
		return cmp.Compare(b.TotalSize, a.TotalSize)
	})

	return result
}
