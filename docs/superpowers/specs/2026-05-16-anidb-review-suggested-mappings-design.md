# AniDB Review Request - Suggested Mappings

## Context

When submitting a review request for an AniDB torrent, users can currently only provide a corrected AniDB ID. They cannot specify the exact mapping details (season type, season, episode range) they believe are correct. This forces reviewers to figure out mappings from scratch.

## Goal

Allow users to suggest specific mapping rows when submitting AniDB review requests, making the reviewer's job easier and submissions more actionable.

## Data Model

### New Type
```typescript
type SuggestedMapping = {
  s_type: "abs" | "tv" | "ani";
  s: number;
  ep_start: number;
  ep_end: number;
};
```

### Updated MappingReviewRequest
```typescript
type MappingReviewRequest = {
  // existing fields...
  suggested_mappings?: SuggestedMapping[];  // NEW
};
```

### Backend Storage

Add `suggested_mappings` column to `torrent_mapping_review` table (JSON text, nullable).

Go struct:
```go
type SuggestedMapping struct {
    SType   string `json:"s_type"`
    S       int    `json:"s"`
    EpStart int    `json:"ep_start"`
    EpEnd   int    `json:"ep_end"`
}
```

## UI Changes

### TorrentMappingReviewSheet

When `target === "anidb"` and user has selected a corrected ID:

1. Show mapping input section below the AniDB search
2. Reuse `MappingPreviewTable` component from detail sheet (already built)
3. "+ Add Mapping" button to add rows
4. Each row: s_type (select), season (input), ep_start (input), ep_end (input), delete button

### Validation

- If corrected AniDB ID is provided, at least one mapping row is required
- Each mapping: ep_start <= ep_end, all values non-negative
- Show toast error on validation failure

## API Changes

### POST /torrent/review-requests

Request body adds optional field:
```json
{
  "hash": "...",
  "target": "anidb",
  "reason": "wrong_mapping",
  "prev_id": "12345",
  "mapping_id": "67890",
  "suggested_mappings": [
    {"s_type": "tv", "s": 1, "ep_start": 1, "ep_end": 12}
  ]
}
```

### GET /torrent/review-requests (list)

Response includes suggested_mappings when present.

## Files to Modify

### Backend
- `migrations/` - Add suggested_mappings column
- `internal/torrent_mapping_review/db.go` - Add field to struct, update insert/scan
- `internal/dash/api/torrent_review_requests.go` - Accept suggested_mappings in create handler

### Frontend
- `apps/dash/src/api/torrent-review-requests.ts` - Add SuggestedMapping type, update MappingReviewRequest
- `apps/dash/src/components/torrent-mapping-review-sheet.tsx` - Add mapping table UI for AniDB
- `apps/dash/src/components/torrent-review-request-detail-sheet.tsx` - Display suggested_mappings in read-only view

## Verification

1. Submit AniDB review with suggested mappings
2. Verify mappings stored in database
3. Verify mappings display in review detail view
4. Verify validation prevents empty mappings when ID provided
5. Verify IMDB flow unchanged
