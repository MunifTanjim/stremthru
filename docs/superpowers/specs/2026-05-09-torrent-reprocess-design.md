# Torrent Reprocess Endpoint Design

## Context

Need dashboard endpoint to re-process specific torrents: re-parse titles and re-map to IMDB/AniDB. Useful when parser improves or mappings need correction.

## Requirements

- **Scope**: Re-parse (go-ptt) + re-map (IMDB/AniDB)
- **Selection**: Hash-based API + UI buttons on existing mapping lists
- **Targets**: User-selectable (IMDB, AniDB, or both)
- **Execution**: Parse always sync; mapping sync for ≤10 hashes, async for >10
- **Existing mappings**: Clear before re-mapping

## API Design

**Endpoint**: `POST /api/dash/torrents/reprocess`

**Request**:
```json
{
  "hashes": ["abc123", "def456"],
  "targets": ["imdb", "anidb"]  // optional, defaults to both
}
```

**Response (sync, ≤10 hashes)**:
```json
{
  "mode": "sync",
  "processed": 5,
  "parsed": 5,
  "mapped": {"imdb": 3, "anidb": 4}
}
```

**Response (async, >10 hashes)**:
```json
{
  "mode": "async",
  "queued": 50
}
```

## Backend Design

**File**: `internal/dash/api/torrent_reprocess.go`

**Flow**:
1. Parse request, validate hashes exist
2. Delete existing mappings from `imdb_torrent` / `anidb_torrent` (based on targets)
3. **Always sync**: Call `ForceParse()` on all hashes
4. If ≤10 hashes → sync mapping:
   - Run IMDB/AniDB mapping logic inline
   - Return counts
5. If >10 hashes → async mapping:
   - Return immediately after parse completes
   - Workers pick up unmapped torrents on next run

**Reused Functions**:
- `internal/torrent_info/db.go`: `ForceParse()`, `GetByHashes()`
- `internal/imdb_torrent/db.go`: mapping logic from `internal/worker/map_imdb_torrent.go`
- `internal/anidb/torrent.db.go`: mapping logic from `internal/worker/map_anidb_torrent.go`

**New Functions**:
- `imdb_torrent.DeleteByHashes(hashes []string)` - delete IMDB mappings
- `anidb_torrent.DeleteByHashes(hashes []string)` - delete AniDB mappings
- Extract mapping logic from workers into reusable functions

## Frontend Design

**API Hook**: `apps/dash/src/api/torrent-info.ts`
- Add `useReprocessTorrents()` mutation hook
- Accepts `{hashes: string[], targets: ("imdb"|"anidb")[]}`
- Invalidates mapping queries on success

**UI Changes**: `apps/dash/src/routes/dash/torrent/mapping.tsx`
- Add "Reprocess" button per row in IMDB/AniDB mapping lists
- Add bulk "Reprocess Selected" with checkbox selection
- Target selector (IMDB/AniDB/Both) - default based on current tab
- Toast notification on success/error

**UX Flow**:
1. User selects torrent(s) or clicks row action
2. Confirms targets (pre-selected based on context)
3. Button shows loading state
4. Toast shows result ("Reprocessed 5 torrents" or "Queued 50 for reprocessing")
5. List refreshes

## Verification

**Manual Testing**:
1. Start dev server, open dashboard
2. Navigate to torrent mapping page
3. Test single reprocess: click row action → verify toast + data refresh
4. Test bulk reprocess: select multiple → verify queued response for >10
5. Test target selection: try IMDB only, AniDB only, both

**Build Verification**:
- Backend: `go build ./...`
- Frontend: `cd apps/dash && bun run build`

## Files to Modify

### Backend
- `internal/dash/router.go` - add route
- `internal/dash/api/torrent_reprocess.go` - new file, endpoint handler
- `internal/imdb_torrent/db.go` - add `DeleteByHashes()`
- `internal/anidb/torrent.db.go` - add `DeleteByHashes()`
- `internal/worker/map_imdb_torrent.go` - extract reusable mapping function
- `internal/worker/map_anidb_torrent.go` - extract reusable mapping function

### Frontend
- `apps/dash/src/api/torrent-info.ts` - add mutation hook
- `apps/dash/src/routes/dash/torrent/mapping.tsx` - add UI controls
