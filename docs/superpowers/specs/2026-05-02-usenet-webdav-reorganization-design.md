# Usenet WebDAV Package Reorganization

## Goal

Reorganize `internal/usenet/webdav` to match codebase conventions (one file per major type, snake_case names, util.go for helpers).

## Current Structure

| File          | Lines | Contents                                |
| ------------- | ----- | --------------------------------------- |
| webdav.go     | 50    | Handler, routing, auth middleware       |
| filesystem.go | 301   | FileSystem + 4 FileInfo types + helpers |
| file.go       | 151   | virtualDir + lazyStreamFile             |

Problem: `filesystem.go` mixes FileSystem logic, FileInfo types, and utility functions.

## Target Structure

| File          | Contents                                                  | Lines (approx) |
| ------------- | --------------------------------------------------------- | -------------- |
| webdav.go     | Handler, routing, auth middleware                         | ~50            |
| filesystem.go | FileSystem struct + methods                               | ~120           |
| file.go       | virtualDir + lazyStreamFile                               | ~150           |
| file_info.go  | rootDirInfo, nzbDirInfo, contentDirInfo, contentFileInfo  | ~70            |
| util.go       | splitPath, findContentFile, isDirectory, statusDownloaded | ~50            |

## Implementation Steps

1. Create `util.go` with:
   - `statusDownloaded` const
   - `splitPath(name string) []string`
   - `findContentFile(files []usenet_pool.NZBContentFile, filePath string) *usenet_pool.NZBContentFile`
   - `isDirectory(cf *usenet_pool.NZBContentFile) bool`

2. Create `file_info.go` with:
   - `rootDirInfo` struct + methods
   - `nzbDirInfo` struct + methods
   - `contentDirInfo` struct + methods
   - `contentFileInfo` struct + methods

3. Update `filesystem.go`:
   - Remove extracted types and functions
   - Keep FileSystem struct and all its methods

4. Verify build passes

## Constraints

- No logic changes - pure reorganization
- Maintain all existing functionality
- Follow existing codebase naming conventions
