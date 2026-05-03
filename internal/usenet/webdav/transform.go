package usenet_webdav

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
)

func isExtensionAllowed(filename string, isArchive bool) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return false
	}
	if isArchive {
		switch usenet_pool.DetectArchiveFileTypeByExtension(filename) {
		case usenet_pool.FileType7z:
			ext = ".7z"
		case usenet_pool.FileTypeRAR:
			ext = ".rar"
		}
	}
	return config.WebDAVFileExtFilter.Has(ext)
}

// ContentEntry represents a WebDAV-friendly view of NZB content
type ContentEntry struct {
	Name        string                      // display name
	Size        int64                       // file size
	ModTime     time.Time                   // modification time
	Source      *usenet_pool.NZBContentFile // original file reference (for streaming)
	ContentPath string                      // path for streaming (using original names, "/" separated)
}

type seenEntry struct {
	index    int
	prefix   string
	prefixed bool
}

// TransformContentFiles converts NZBContentFile tree to flat WebDAV-friendly ContentEntry list.
func TransformContentFiles(files []usenet_pool.NZBContentFile, modTime time.Time) []ContentEntry {
	return flattenFiles(files, modTime, "", []ContentEntry{}, map[string]*seenEntry{})
}

func flattenFiles(files []usenet_pool.NZBContentFile, modTime time.Time, parentPath string, entries []ContentEntry, seen map[string]*seenEntry) []ContentEntry {
	archivePrefix := ""
	if parentPath != "" {
		archivePrefix, _ = usenet_pool.GetArchiveBaseName(filepath.Base(parentPath))
	}
	for i := range files {
		f := &files[i]

		if !f.Streamable {
			continue
		}

		hasParts := len(f.Parts) > 0
		hasFiles := len(f.Files) > 0

		if f.Size == 0 && !hasParts && !hasFiles {
			continue
		}

		fName := f.Name
		if f.Alias != "" {
			fName = f.Alias
		}
		fName = filepath.Base(fName)

		contentPath := fName
		if parentPath != "" {
			contentPath = parentPath + "/" + fName
		}

		isArchive := f.Type == usenet_pool.NZBContentFileTypeArchive

		if isArchive && (hasFiles || hasParts) {
			// Archive parts: emit each as file
			for j := range f.Parts {
				part := &f.Parts[j]
				partName := part.Name
				if part.Alias != "" {
					partName = part.Alias
				}
				if !isExtensionAllowed(partName, isArchive) {
					continue
				}
				partContentPath := partName
				if parentPath != "" {
					partContentPath = parentPath + "/" + partName
				}

				if first, exists := seen[strings.ToLower(partName)]; exists {
					if !first.prefixed && first.prefix != "" {
						entries[first.index].Name = first.prefix + "_" + entries[first.index].Name
						first.prefixed = true
					}
					if archivePrefix != "" {
						partName = archivePrefix + "_" + partName
					}
				} else {
					seen[strings.ToLower(partName)] = &seenEntry{index: len(entries), prefix: archivePrefix}
				}

				entries = append(entries, ContentEntry{
					Name:        partName,
					Size:        part.Size,
					ModTime:     modTime,
					Source:      part,
					ContentPath: partContentPath,
				})
			}

			if hasFiles {
				// Recurse into archive contents, flatten to root
				entries = flattenFiles(f.Files, modTime, contentPath, entries, seen)
			}
		} else {
			if !isExtensionAllowed(fName, isArchive) {
				continue
			}

			if first, exists := seen[strings.ToLower(fName)]; exists {
				if !first.prefixed && first.prefix != "" {
					entries[first.index].Name = first.prefix + "_" + entries[first.index].Name
					first.prefixed = true
				}
				if archivePrefix != "" {
					fName = archivePrefix + "_" + fName
				}
			} else {
				seen[strings.ToLower(fName)] = &seenEntry{index: len(entries), prefix: archivePrefix}
			}

			entries = append(entries, ContentEntry{
				Name:        fName,
				Size:        f.Size,
				ModTime:     modTime,
				Source:      f,
				ContentPath: contentPath,
			})
		}
	}
	return entries
}

// findContentEntry finds a ContentEntry by name (flat lookup, case-insensitive).
func findContentEntry(entries []ContentEntry, name string) *ContentEntry {
	for i := range entries {
		if strings.EqualFold(entries[i].Name, name) {
			return &entries[i]
		}
	}
	return nil
}
