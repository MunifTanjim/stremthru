# WebDAV NZB File Surfacing Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform NZB archive display in WebDAV to show archive parts as files and extracted contents in extension-stripped folders.

**Architecture:** Create a transformation layer that converts NZBContentFile trees into WebDAV-friendly ContentEntry structures. The filesystem uses transformed entries instead of raw NZBContentFiles.

**Tech Stack:** Go, golang.org/x/net/webdav

---

## File Structure

| File            | Responsibility                                                                                    |
| --------------- | ------------------------------------------------------------------------------------------------- |
| `transform.go`  | **New** - ContentEntry type, TransformContentFiles(), stripArchiveExtension(), findContentEntry() |
| `file_info.go`  | Add contentEntryInfo implementing os.FileInfo for ContentEntry                                    |
| `filesystem.go` | Use transformed tree in Stat, open, openNZBDir, openContentDir, openContentFile                   |
| `util.go`       | Keep splitPath, statusDownloaded; remove isDirectory, findContentFile (moved to transform.go)     |

---

### Task 1: Create transform.go with ContentEntry type and stripArchiveExtension

**Files:**

- Create: `internal/usenet/webdav/transform.go`

- [ ] **Step 1: Create transform.go with ContentEntry type and stripArchiveExtension**

```go
package usenet_webdav

import (
	"regexp"
	"strings"
	"time"

	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
)

// ContentEntry represents a WebDAV-friendly view of NZB content
type ContentEntry struct {
	Name        string                       // display name (stripped extension for virtual folders)
	IsDir       bool                         // true for virtual folders containing extracted files
	Size        int64                        // file size (0 for directories)
	ModTime     time.Time                    // modification time
	Source      *usenet_pool.NZBContentFile  // original file reference (for streaming)
	ContentPath string                       // path for streaming (using original names, "/" separated)
	Children    []ContentEntry               // nested entries (for directories)
}

// archiveExtensions are extensions to strip when creating virtual folders
var archiveExtensions = []string{".rar", ".zip", ".7z", ".tar.gz", ".tgz", ".tar", ".gz"}

// multiPartPattern matches .partN, .rNN, .NNN patterns before archive extension
var multiPartPattern = regexp.MustCompile(`(?i)(\.(part\d+|r\d{2,}|\d{3,}))$`)

// stripArchiveExtension removes archive extension and multi-part suffixes from name
// e.g., "movie.part1.rar" -> "movie", "movie.rar" -> "movie", "movie.r00" -> "movie"
func stripArchiveExtension(name string) string {
	result := name
	lower := strings.ToLower(result)

	// Strip archive extension first
	for _, ext := range archiveExtensions {
		if strings.HasSuffix(lower, ext) {
			result = result[:len(result)-len(ext)]
			lower = strings.ToLower(result)
			break
		}
	}

	// Strip multi-part suffix (e.g., .part1, .r00, .001)
	if loc := multiPartPattern.FindStringIndex(result); loc != nil {
		result = result[:loc[0]]
	}

	// If result is empty, use original name
	if result == "" {
		return name
	}

	return result
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 2: Add TransformContentFiles function

**Files:**

- Modify: `internal/usenet/webdav/transform.go`

- [ ] **Step 1: Add TransformContentFiles function**

Append to transform.go:

```go
// TransformContentFiles converts NZBContentFile tree to WebDAV-friendly ContentEntry tree.
// Archives with Parts become individual files; archives with Files become virtual folders
// with stripped extension names.
func TransformContentFiles(files []usenet_pool.NZBContentFile, modTime time.Time, parentPath string) []ContentEntry {
	var entries []ContentEntry

	for i := range files {
		cf := &files[i]
		cfName := cf.Name
		if cf.Alias != "" {
			cfName = cf.Alias
		}

		// Build content path for this file
		contentPath := cfName
		if parentPath != "" {
			contentPath = parentPath + "/" + cfName
		}

		isArchive := cf.Type == usenet_pool.NZBContentFileTypeArchive
		hasParts := len(cf.Parts) > 0
		hasFiles := len(cf.Files) > 0

		if isArchive && (hasParts || hasFiles) {
			// Archive with Parts: emit each part as a file
			if hasParts {
				for j := range cf.Parts {
					part := &cf.Parts[j]
					partName := part.Name
					if part.Alias != "" {
						partName = part.Alias
					}
					partContentPath := cfName + "/" + partName
					if parentPath != "" {
						partContentPath = parentPath + "/" + partContentPath
					}
					entries = append(entries, ContentEntry{
						Name:        partName,
						IsDir:       false,
						Size:        part.Size,
						ModTime:     modTime,
						Source:      part,
						ContentPath: partContentPath,
					})
				}
			}

			// Archive with Files: emit virtual folder with stripped name
			if hasFiles {
				folderName := stripArchiveExtension(cfName)
				children := TransformContentFiles(cf.Files, modTime, contentPath)
				entries = append(entries, ContentEntry{
					Name:        folderName,
					IsDir:       true,
					Size:        0,
					ModTime:     modTime,
					Source:      cf,
					ContentPath: contentPath,
					Children:    children,
				})
			}
		} else {
			// Regular file or archive without Parts/Files
			entries = append(entries, ContentEntry{
				Name:        cfName,
				IsDir:       false,
				Size:        cf.Size,
				ModTime:     modTime,
				Source:      cf,
				ContentPath: contentPath,
			})
		}
	}

	return entries
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 3: Add findContentEntry function

**Files:**

- Modify: `internal/usenet/webdav/transform.go`

- [ ] **Step 1: Add findContentEntry function**

Append to transform.go:

```go
// findContentEntry finds a ContentEntry by path in the transformed tree.
// Path components are matched case-insensitively.
func findContentEntry(entries []ContentEntry, filePath string) *ContentEntry {
	parts := strings.SplitN(filePath, "/", 2)
	name := parts[0]

	for i := range entries {
		entry := &entries[i]
		if strings.EqualFold(entry.Name, name) {
			if len(parts) == 1 {
				return entry
			}
			if entry.IsDir {
				return findContentEntry(entry.Children, parts[1])
			}
			return nil
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 4: Add contentEntryInfo to file_info.go

**Files:**

- Modify: `internal/usenet/webdav/file_info.go`

- [ ] **Step 1: Add contentEntryInfo type**

Append to file_info.go (after contentFileInfo):

```go
type contentEntryInfo struct {
	entry *ContentEntry
}

func (e *contentEntryInfo) Name() string { return e.entry.Name }
func (e *contentEntryInfo) Size() int64  { return e.entry.Size }
func (e *contentEntryInfo) Mode() os.FileMode {
	if e.entry.IsDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (e *contentEntryInfo) ModTime() time.Time { return e.entry.ModTime }
func (e *contentEntryInfo) IsDir() bool        { return e.entry.IsDir }
func (e *contentEntryInfo) Sys() any           { return nil }
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 5: Update filesystem.go Stat method

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Update Stat method to use transformed tree**

Replace the Stat method (lines 41-77) with:

```go
func (fs *FileSystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean("/" + name)

	if name == "/" {
		return &rootDirInfo{}, nil
	}

	parts := splitPath(name)
	if len(parts) == 0 {
		return nil, os.ErrNotExist
	}

	nzbName := parts[0]
	info, err := fs.findNZBInfoByName(nzbName)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, os.ErrNotExist
	}

	if len(parts) == 1 {
		return &nzbDirInfo{info: info}, nil
	}

	// Transform and find entry
	entries := TransformContentFiles(info.ContentFiles.Data, info.CAt.Time, "")
	filePath := strings.Join(parts[1:], "/")
	entry := findContentEntry(entries, filePath)
	if entry == nil {
		return nil, os.ErrNotExist
	}

	return &contentEntryInfo{entry: entry}, nil
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 6: Update filesystem.go open method

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Update open method to use transformed tree**

Replace the open method (lines 79-115) with:

```go
func (fs *FileSystem) open(ctx context.Context, name string) (webdav.File, error) {
	name = path.Clean("/" + name)

	if name == "/" {
		return fs.openRootDir(ctx)
	}

	parts := splitPath(name)
	if len(parts) == 0 {
		return nil, os.ErrNotExist
	}

	nzbName := parts[0]
	info, err := fs.findNZBInfoByName(nzbName)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, os.ErrNotExist
	}

	if len(parts) == 1 {
		return fs.openNZBDir(ctx, info)
	}

	// Transform and find entry
	entries := TransformContentFiles(info.ContentFiles.Data, info.CAt.Time, "")
	filePath := strings.Join(parts[1:], "/")
	entry := findContentEntry(entries, filePath)
	if entry == nil {
		return nil, os.ErrNotExist
	}

	if entry.IsDir {
		return fs.openContentDir(ctx, info, entry)
	}

	return fs.openContentFile(ctx, info, entry)
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: May fail - openNZBDir/openContentDir/openContentFile signatures need updating

---

### Task 7: Update openNZBDir method

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Update openNZBDir to use transformed entries**

Replace openNZBDir method (lines 131-149) with:

```go
func (fs *FileSystem) openNZBDir(ctx context.Context, info *nzb_info.NZBInfo) (webdav.File, error) {
	entries := TransformContentFiles(info.ContentFiles.Data, info.CAt.Time, "")

	fileInfos := make([]os.FileInfo, 0, len(entries))
	for i := range entries {
		fileInfos = append(fileInfos, &contentEntryInfo{entry: &entries[i]})
	}

	return &virtualDir{
		info:    &nzbDirInfo{info: info},
		entries: fileInfos,
	}, nil
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors (or still needs openContentDir/openContentFile)

---

### Task 8: Update openContentDir method

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Update openContentDir to use ContentEntry**

Replace openContentDir method (lines 151-169) with:

```go
func (fs *FileSystem) openContentDir(ctx context.Context, info *nzb_info.NZBInfo, entry *ContentEntry) (webdav.File, error) {
	fileInfos := make([]os.FileInfo, 0, len(entry.Children))
	for i := range entry.Children {
		fileInfos = append(fileInfos, &contentEntryInfo{entry: &entry.Children[i]})
	}

	return &virtualDir{
		info:    &contentEntryInfo{entry: entry},
		entries: fileInfos,
	}, nil
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: May still fail due to openContentFile

---

### Task 9: Update openContentFile method

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Update openContentFile to use ContentEntry**

Replace openContentFile method (lines 171-182) with:

```go
func (fs *FileSystem) openContentFile(ctx context.Context, info *nzb_info.NZBInfo, entry *ContentEntry) (webdav.File, error) {
	return &lazyStreamFile{
		info:        &contentEntryInfo{entry: entry},
		nzbInfo:     info,
		contentPath: entry.ContentPath,
		ctx:         ctx,
	}, nil
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 10: Clean up util.go

**Files:**

- Modify: `internal/usenet/webdav/util.go`

- [ ] **Step 1: Remove isDirectory and findContentFile from util.go**

Update util.go to only contain:

```go
package usenet_webdav

import (
	"strings"
)

const statusDownloaded = "downloaded"

func splitPath(name string) []string {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil
	}
	return strings.Split(name, "/")
}
```

- [ ] **Step 2: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 11: Clean up unused types in file_info.go

**Files:**

- Modify: `internal/usenet/webdav/file_info.go`

- [ ] **Step 1: Remove contentDirInfo and contentFileInfo (no longer needed)**

Update file_info.go to contain only:

```go
package usenet_webdav

import (
	"os"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
)

type rootDirInfo struct{}

func (d *rootDirInfo) Name() string       { return "/" }
func (d *rootDirInfo) Size() int64        { return 0 }
func (d *rootDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (d *rootDirInfo) ModTime() time.Time { return time.Now() }
func (d *rootDirInfo) IsDir() bool        { return true }
func (d *rootDirInfo) Sys() any           { return nil }

type nzbDirInfo struct {
	info *nzb_info.NZBInfo
}

func (d *nzbDirInfo) Name() string       { return d.info.Name }
func (d *nzbDirInfo) Size() int64        { return 0 }
func (d *nzbDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (d *nzbDirInfo) ModTime() time.Time { return d.info.CAt.Time }
func (d *nzbDirInfo) IsDir() bool        { return true }
func (d *nzbDirInfo) Sys() any           { return nil }

type contentEntryInfo struct {
	entry *ContentEntry
}

func (e *contentEntryInfo) Name() string { return e.entry.Name }
func (e *contentEntryInfo) Size() int64  { return e.entry.Size }
func (e *contentEntryInfo) Mode() os.FileMode {
	if e.entry.IsDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (e *contentEntryInfo) ModTime() time.Time { return e.entry.ModTime }
func (e *contentEntryInfo) IsDir() bool        { return e.entry.IsDir }
func (e *contentEntryInfo) Sys() any           { return nil }
```

- [ ] **Step 2: Remove unused import**

Remove `usenet_pool` import from file_info.go if present.

- [ ] **Step 3: Verify file compiles**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

---

### Task 12: Final build and verification

**Files:**

- All files in `internal/usenet/webdav/`

- [ ] **Step 1: Run package build**

Run: `go build ./internal/usenet/webdav/`
Expected: No errors

- [ ] **Step 2: Run full project build**

Run: `go build ./...`
Expected: No errors (ignore go module cache warnings)

- [ ] **Step 3: Verify file structure**

Run: `ls internal/usenet/webdav/*.go`
Expected: file.go, file_info.go, filesystem.go, transform.go, util.go, webdav.go

- [ ] **Step 4: Verify line counts**

Run: `wc -l internal/usenet/webdav/*.go`
Expected: transform.go should be largest (~100-120 lines), others smaller
