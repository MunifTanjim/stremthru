# TorBox WebDAV Newz Endpoint Design

## Summary

Add `/v0/webdav/torbox/newz/` endpoint to expose TorBox Usenet items via WebDAV. Similar to existing `/v0/webdav/newz/` but sources data from TorBox API instead of local StremThru pool.

## Endpoint

**Path:** `/v0/webdav/torbox/newz/`

**Authentication:**

- HTTP Basic Auth using `STREMTHRU_AUTH` credentials
- TorBox API token resolved from `STREMTHRU_STORE_AUTH` for authenticated user

## Directory Structure

```
/v0/webdav/torbox/newz/
└── {usenet-download-name}/
    └── {file-name}
```

## Data Flow

1. **Root listing:** Call `torbox.ListUsenetDownload()` to get all Usenet items
2. **Directory listing:** Return files from specific download, filtered by `STREMTHRU_WEBDAV_FILE_EXT_FILTER`
3. **File open:** Call `torbox.RequestUsenetDownloadLink()` to get direct URL, return HTTP 302 redirect

## Implementation

### Files to Create

| File                                         | Purpose                                |
| -------------------------------------------- | -------------------------------------- |
| `internal/store/torbox/webdav/router.go`     | Endpoint registration, auth middleware |
| `internal/store/torbox/webdav/filesystem.go` | WebDAV FileSystem implementation       |
| `internal/store/torbox/webdav/file.go`       | File and directory types               |

### Key Functions

**router.go:**

- `AddEndpoints(mux)` - Register `/v0/webdav/torbox/newz/` handler
- `withAuth(next)` - Validate STREMTHRU_AUTH, resolve TorBox token, inject into context

**filesystem.go:**

- `FileSystem.Stat(ctx, name)` - Return file info for path
- `FileSystem.OpenFile(ctx, name, flag, perm)` - Open file (write operations blocked)
- `FileSystem.Readdir(ctx, name, n)` - List directory contents

**file.go:**

- `webdavDir` - Directory type implementing `webdav.File`
- `webdavFile` - File type that redirects to TorBox URL on Read

### Dependencies

- `store/torbox` - TorBox API client
- `config.WebDAVFileExtFilter` - File extension filtering
- `golang.org/x/net/webdav` - WebDAV handler

## File Extension Filtering

Apply `STREMTHRU_WEBDAV_FILE_EXT_FILTER` to filter visible files, same as existing `/v0/webdav/newz/`.

## Error Handling

- No TorBox token for user → 401 Unauthorized with message
- TorBox API error → 502 Bad Gateway
- Download not found → 404 Not Found

## Documentation

Update `docs/api/newz.md` to add TorBox WebDAV endpoint section.
