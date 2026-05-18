# AniDB Review Resolution - Hybrid Preview Flow

## Context

AniDB mappings are more complex than IMDB - a single torrent can have multiple mapping rows with different `(tid, hash, s_type, s, ep_start, ep_end)` combinations. The current review resolution only accepts a `mapping_id` (anidb_id), which is insufficient.

When a reviewer resolves an AniDB review request, they need to:
1. See what mappings the system would generate for the corrected anidb_id
2. Review and optionally modify those mappings
3. Confirm before saving

## Goals

- Reviewer can preview mapper output before committing
- Reviewer can edit/add/remove mapping rows
- Reviewer can reject mapping entirely (mark unmappable)
- IMDB resolution unchanged (simple mapping_id flow)

## Non-Goals

- Changing how review requests are submitted (already captures corrected ID)
- Modifying the mapping algorithm itself

## Data Model

### AniDB Torrent Mapping Row
```
tid         - AniDB ID (e.g., "12345")
hash        - torrent hash
s_type      - season type: "abs" | "tv" | "ani"
s           - season number
ep_start    - episode start
ep_end      - episode end
eps         - comma-separated episode list (auto-derived, not user-editable)
```

Note: `eps` is computed from `ep_start` and `ep_end` on save, not exposed in UI.

### Review Resolution States
- `pending` - awaiting resolution
- `resolved` - mappings applied
- `rejected` - marked as unmappable

## API Changes

### 1. Preview Mapping Endpoint

**Request:** `POST /review-requests/:id/preview`
```json
{
  "anidb_id": "12345"
}
```

**Response:**
```json
{
  "mappings": [
    {"tid": "12345", "s_type": "abs", "s": 1, "ep_start": 1, "ep_end": 24},
    {"tid": "12345", "s_type": "ani", "s": 1, "ep_start": 1, "ep_end": 24}
  ],
  "anidb_titles": ["Anime Title (Main)", "Anime Title (English)"]
}
```

**Behavior:**
- Requires review status === `pending`, else 400 error
- Fetches torrent info by hash from review request
- Runs `MapTorrentToAniDB` logic with forced anidb_id
- Returns proposed mappings WITHOUT saving
- If mapper returns empty, response has empty `mappings` array

### 2. Resolve Endpoint Changes

**Current:** `POST /review-requests/:id/resolve`
```json
{
  "mapping_id": "tt1234567"  // for IMDB
}
```

**New for AniDB:** `POST /review-requests/:id/resolve`
```json
{
  "target": "anidb",
  "mappings": [
    {"tid": "12345", "s_type": "abs", "s": 1, "ep_start": 1, "ep_end": 24},
    {"tid": "12345", "s_type": "ani", "s": 1, "ep_start": 1, "ep_end": 24}
  ]
}
```

**Behavior:**
- Requires review status === `pending`, else 400 error
- Delete existing mappings by `(prev_id, hash)` for target
- Insert provided mappings
- Update review status to `resolved`

### 3. Reject Endpoint (both IMDB and AniDB)

**Request:** `POST /review-requests/:id/reject`
```json
{
  "reason": "unmappable"  // or "fake_torrent", "duplicate"
}
```

**Behavior:**
- Requires review status === `pending`, else 400 error
- Delete existing mappings by `(prev_id, hash)` for target
- Do NOT insert new mappings
- Update review status to `rejected`

## Frontend Changes

### TorrentReviewRequestDetailSheet

**Current flow:**
```
[AniDB Search] → [Resolve button]
```

**New flow:**
```
[AniDB Search] → [Preview button]
                      ↓
              ┌──────────────────────────────────┐
              │ Mapping Preview                  │
              │                                  │
              │ ┌──────┬───┬─────────┬─────────┐ │
              │ │s_type│ s │ep_start │ ep_end  │ │
              │ ├──────┼───┼─────────┼─────────┤ │
              │ │ abs  │ 1 │    1    │   24    │ │
              │ │ ani  │ 1 │    1    │   24    │ │
              │ └──────┴───┴─────────┴─────────┘ │
              │                                  │
              │ [+ Add Row]                      │
              │                                  │
              │ [Cancel] [Reject] [Apply]        │
              └──────────────────────────────────┘
```

**Editable fields per row:**
- `s_type` - dropdown: abs/tv/ani
- `s` - number input
- `ep_start` - number input
- `ep_end` - number input
- [Delete] button per row

**Buttons:**
- **Preview** - calls preview endpoint, shows table
- **+ Add Row** - adds empty row for manual entry
- **Cancel** - closes preview, returns to search
- **Reject** - calls reject endpoint, marks as unmappable
- **Apply** - calls resolve endpoint with current mappings

### State Management

```typescript
type PreviewMapping = {
  tid: string;
  s_type: "abs" | "tv" | "ani";
  s: number;
  ep_start: number;
  ep_end: number;
};

// Component state
const [previewMappings, setPreviewMappings] = useState<PreviewMapping[] | null>(null);
const [isPreviewMode, setIsPreviewMode] = useState(false);
```

### UI States

1. **Initial** - AniDB search visible, no preview
2. **Previewing** - Loading spinner while fetching preview
3. **Preview shown** - Editable table with mappings
4. **Empty preview** - "No mappings generated. Add manually or reject."
5. **Applying** - Loading while resolving

## Files to Modify

### Backend
- `internal/dash/api/torrent_review_requests.go` - add preview/reject endpoints, modify resolve
- `internal/torrent_mapping_review/db.go` - add rejected status support

### Frontend
- `apps/dash/src/api/torrent-review-requests.ts` - add preview/reject API calls
- `apps/dash/src/components/torrent-review-request-detail-sheet.tsx` - preview UI

## Edge Cases

1. **Mapper returns no mappings** - Show empty state, allow manual add or reject
2. **All rows deleted by reviewer** - Treat as reject, prompt confirmation
3. **Invalid episode range (start > end)** - Validation error, prevent apply
4. **Network error during preview** - Show error, allow retry
5. **Review already resolved** - Disable all actions, show read-only

## Verification

1. Create AniDB review request for a torrent with multiple season mappings
2. Open review detail sheet, search for corrected anidb_id
3. Click Preview - verify mappings appear in table
4. Edit a row - change episode range
5. Add a row - enter manual mapping
6. Delete a row
7. Click Apply - verify old mappings deleted, new ones inserted
8. Test Reject flow - verify mappings deleted, status updated
9. Test empty preview - verify manual add works