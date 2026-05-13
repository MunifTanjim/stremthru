# Usenet WebDAV Package Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize `internal/usenet/webdav` to match codebase conventions.

**Architecture:** Extract FileInfo types and utility functions from filesystem.go into separate files. Pure code movement, no logic changes.

**Tech Stack:** Go

---

## File Structure

| File            | Responsibility                                                                |
| --------------- | ----------------------------------------------------------------------------- |
| `webdav.go`     | Handler, routing, auth (unchanged)                                            |
| `filesystem.go` | FileSystem struct + methods                                                   |
| `file.go`       | virtualDir + lazyStreamFile (unchanged)                                       |
| `file_info.go`  | All FileInfo types (rootDirInfo, nzbDirInfo, contentDirInfo, contentFileInfo) |
| `util.go`       | Helper functions (splitPath, findContentFile, isDirectory) + constants        |

---

### Task 1: Create util.go

**Files:**

- Create: `internal/usenet/webdav/util.go`

- [ ] **Step 1: Create util.go with helper functions**

```go
package usenet_webdav

import (
	"strings"

	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
)

const statusDownloaded = "downloaded"

func splitPath(name string) []string {
	name = strings.Trim(name, "/")
	if name == "" {
		return nil
	}
	return strings.Split(name, "/")
}

func findContentFile(files []usenet_pool.NZBContentFile, filePath string) *usenet_pool.NZBContentFile {
	parts := strings.SplitN(filePath, "/", 2)
	name := parts[0]

	for i := range files {
		cf := &files[i]
		cfName := cf.Name
		if cf.Alias != "" {
			cfName = cf.Alias
		}
		if strings.EqualFold(cfName, name) {
			if len(parts) == 1 {
				return cf
			}
			subFiles := cf.Files
			if len(subFiles) == 0 {
				subFiles = cf.Parts
			}
			return findContentFile(subFiles, parts[1])
		}
	}
	return nil
}

func isDirectory(cf *usenet_pool.NZBContentFile) bool {
	return cf.Type == usenet_pool.NZBContentFileTypeArchive && (len(cf.Files) > 0 || len(cf.Parts) > 0)
}
```

- [ ] **Step 2: Verify file created**

Run: `ls internal/usenet/webdav/util.go`
Expected: File exists

---

### Task 2: Create file_info.go

**Files:**

- Create: `internal/usenet/webdav/file_info.go`

- [ ] **Step 1: Create file_info.go with all FileInfo types**

```go
package usenet_webdav

import (
	"os"
	"time"

	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
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

type contentDirInfo struct {
	file *usenet_pool.NZBContentFile
}

func (d *contentDirInfo) Name() string {
	if d.file.Alias != "" {
		return d.file.Alias
	}
	return d.file.Name
}
func (d *contentDirInfo) Size() int64        { return 0 }
func (d *contentDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (d *contentDirInfo) ModTime() time.Time { return time.Now() }
func (d *contentDirInfo) IsDir() bool        { return true }
func (d *contentDirInfo) Sys() any           { return nil }

type contentFileInfo struct {
	file    *usenet_pool.NZBContentFile
	nzbInfo *nzb_info.NZBInfo
	size    int64
}

func (f *contentFileInfo) Name() string {
	if f.file.Alias != "" {
		return f.file.Alias
	}
	return f.file.Name
}

func (f *contentFileInfo) Size() int64 {
	if f.size > 0 {
		return f.size
	}
	return f.file.Size
}

func (f *contentFileInfo) Mode() os.FileMode  { return 0644 }
func (f *contentFileInfo) ModTime() time.Time { return f.nzbInfo.CAt.Time }
func (f *contentFileInfo) IsDir() bool        { return false }
func (f *contentFileInfo) Sys() any           { return nil }
```

- [ ] **Step 2: Verify file created**

Run: `ls internal/usenet/webdav/file_info.go`
Expected: File exists

---

### Task 3: Update filesystem.go

**Files:**

- Modify: `internal/usenet/webdav/filesystem.go`

- [ ] **Step 1: Remove extracted code from filesystem.go**

Remove these sections from filesystem.go:

- Line 118: `const statusDownloaded = "downloaded"` (keep reference, it's now in util.go)
- Lines 205-239: `splitPath`, `findContentFile`, `isDirectory` functions
- Lines 241-300: All four FileInfo types (`rootDirInfo`, `nzbDirInfo`, `contentDirInfo`, `contentFileInfo`)

The resulting filesystem.go should contain only:

- Package declaration and imports
- `FileSystem` struct and `NewFileSystem()`
- `Mkdir`, `RemoveAll`, `Rename`, `OpenFile`, `Stat` methods
- `open`, `findNZBInfoByName` methods
- `openRootDir`, `openNZBDir`, `openContentDir`, `openContentFile` methods

- [ ] **Step 2: Clean up imports in filesystem.go**

Remove unused imports after extraction:

- Remove `"time"` (used only by FileInfo types)

Keep these imports:

- `"context"`
- `"os"`
- `"path"`
- `"strings"`
- `"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"`
- `usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"`
- `"golang.org/x/net/webdav"`

---

### Task 4: Verify Build

**Files:**

- All files in `internal/usenet/webdav/`

- [ ] **Step 1: Run go build to verify compilation**

Run: `cd /Users/muniftanjim/Dev/github/MunifTanjim/stremthru && go build ./internal/usenet/webdav/`
Expected: No errors, clean build

- [ ] **Step 2: Run full project build**

Run: `cd /Users/muniftanjim/Dev/github/MunifTanjim/stremthru && go build ./...`
Expected: No errors, clean build

---

### Task 5: Final Verification

- [ ] **Step 1: Verify file count**

Run: `ls internal/usenet/webdav/*.go`
Expected: 5 files (webdav.go, filesystem.go, file.go, file_info.go, util.go)

- [ ] **Step 2: Verify line counts are reasonable**

Run: `wc -l internal/usenet/webdav/*.go`
Expected:

- filesystem.go: ~120 lines (down from 301)
- file_info.go: ~70 lines
- util.go: ~50 lines
