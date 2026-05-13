# WebDAV Config Section Design

## Summary

Create a dedicated WebDAV section in startup logs and dashboard UI for WebDAV-related configuration.

## Config Rename

| Before                                  | After                              |
| --------------------------------------- | ---------------------------------- |
| `STREMTHRU_WEBDAV_NEWZ_FILE_EXT_FILTER` | `STREMTHRU_WEBDAV_FILE_EXT_FILTER` |
| `WebDAVNewzFileExtFilter`               | `WebDAVFileExtFilter`              |

## Files to Modify

1. `internal/config/config.go` - Rename env var key, add PrintConfig section
2. `internal/config/webdav.go` - Rename variable
3. `internal/config/config_display.go` - New struct, remove from Newz, add to ConfigDisplay
4. `apps/dash/src/api/config.ts` - Add WebDAV type, remove from ConfigNewz
5. `apps/dash/src/routes/dash/settings/config.tsx` - Add WebDAVSection component

## Backend Changes

### config_display.go

New struct:

```go
type ConfigDisplayWebDAV struct {
    FileExtFilter []string `json:"file_ext_filter"`
}
```

Add to ConfigDisplay:

```go
type ConfigDisplay struct {
    // ... existing fields ...
    Torz   ConfigDisplayTorz   `json:"torz"`
    WebDAV ConfigDisplayWebDAV `json:"webdav"`  // after Torz
}
```

Remove `WebDAVFileExtFilter` from `ConfigDisplayNewz`.

### config.go

Rename default env key from `STREMTHRU_WEBDAV_NEWZ_FILE_EXT_FILTER` to `STREMTHRU_WEBDAV_FILE_EXT_FILTER`.

Add PrintConfig section after Torz:

```go
l.Println(" WebDAV:")
l.Println("      file ext filter: " + strings.Join(data.WebDAV.FileExtFilter, ","))
l.Println()
```

Remove WebDAV line from Newz section.

### webdav.go

Rename `WebDAVNewzFileExtFilter` to `WebDAVFileExtFilter`.

## Frontend Changes

### config.ts

Add type:

```typescript
type ConfigWebDAV = {
  file_ext_filter: string[];
};
```

Add to ConfigData:

```typescript
webdav: ConfigWebDAV;
```

Remove `webdav_file_ext_filter` from ConfigNewz.

### config.tsx

Add WebDAVSection component:

```tsx
function WebDAVSection({ webdav }: { webdav: ConfigData["webdav"] }) {
  return (
    <ConfigSection
      gradient="from-violet-500 to-purple-500"
      icon="W"
      title="WebDAV"
    >
      <ConfigEntry
        label="File Extension Filter"
        value={
          <div className="flex flex-wrap gap-1">
            {webdav.file_ext_filter.map((ext) => (
              <Badge key={ext} variant="secondary">
                {ext}
              </Badge>
            ))}
          </div>
        }
      />
    </ConfigSection>
  );
}
```

Render after TorzSection in RouteComponent.

Remove WebDAV display from NewzSection.

## Placement

- Startup logs: After Torz section
- Dashboard UI: After TorzSection component
