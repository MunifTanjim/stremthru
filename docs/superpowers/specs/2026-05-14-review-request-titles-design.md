# Review Request Titles Display

## Context

The review request page currently shows torrent hashes and IMDB/AniDB IDs as raw identifiers without human-readable titles. Users must manually look up what each hash or ID represents. Adding titles improves usability when reviewing mapping requests.

## Requirements

- Display torrent name for each hash (from `torrent_info.TorrentTitle`)
- Display title(s) for `prev_id` and `mapping_id` (from `imdb_title` or `anidb` tables based on `target`)
- Show titles in both list table and detail sheet
- Handle AniDB having multiple titles per ID (different languages/romanizations)
- Fallback to showing ID/hash only when title unavailable

## Data Model

### Backend Response Type

```go
// internal/dash/api/torrent_review_requests.go

type MappingReviewWithTitles struct {
    torrent_mapping_review.MappingReview
    HashTitle       string   `json:"hash_title,omitempty"`
    PrevIdTitles    []string `json:"prev_id_titles,omitempty"`
    MappingIdTitles []string `json:"mapping_id_titles,omitempty"`
}
```

### Frontend Type

```typescript
// apps/dash/src/api/torrent-review-requests.ts

export type ReviewRequest = {
  id: number;
  hash: string;
  target: MappingTarget;
  reason: ReviewReason;
  prev_id: string;
  mapping_id: string;
  files?: FileCorrection[];
  comment?: string;
  status: ReviewStatus;
  created_at: string;
  resolved_at?: string;
  // New fields
  hash_title?: string;
  prev_id_titles?: string[];
  mapping_id_titles?: string[];
};
```

## Backend Implementation

### Modified Handler: `handleListTorrentReviewRequests`

Location: `internal/dash/api/torrent_review_requests.go`

Flow:
1. Fetch reviews via `torrent_mapping_review.List(params)`
2. Collect unique hashes from all reviews
3. Collect unique IDs separated by target (IMDB vs AniDB)
4. Batch fetch:
   - `torrent_info.GetByHashes(hashes)` → `map[hash]TorrentInfo`
   - `imdb_title.ListByIds(imdbIds)` → `[]IMDBTitle`
   - `anidb.GetTitlesByIds(anidbIds)` → `AniDBTitles`
5. Build lookup maps:
   - IMDB: `map[tid]string` (single title)
   - AniDB: `map[tid][]string` (multiple titles)
6. Merge titles into response items

### Helper Functions

```go
func collectUniqueHashes(items []MappingReview) []string
func collectIdsByTarget(items []MappingReview) (imdbIds, anidbIds []string)
func buildIMDBTitleMap(titles []IMDBTitle) map[string]string
func buildAniDBTitleMap(titles AniDBTitles) map[string][]string
func getTitlesForId(target MappingTarget, id string, imdbMap map[string]string, anidbMap map[string][]string) []string
```

### Error Handling

- Title lookup errors logged but don't fail request
- Missing titles result in empty/nil fields
- Frontend displays ID/hash only when titles unavailable

## Frontend Implementation

### List Table (`apps/dash/src/routes/dash/torrent/review-requests.tsx`)

- Hash column: truncated hash with torrent title below or in tooltip
- Suggested ID column: ID with first title in parentheses
- Prev ID column: ID with first title in parentheses
- Tooltip on ID columns shows all titles (for AniDB with multiple)

### Detail Sheet (`apps/dash/src/components/torrent-review-request-detail-sheet.tsx`)

- Hash section: full hash + full torrent title
- Prev ID section: ID + all titles listed vertically
- Mapping ID section: ID + all titles listed vertically
- Maintain existing monospace styling for IDs/hashes
- Normal text styling for titles

## Files to Modify

1. `internal/dash/api/torrent_review_requests.go` - Add title resolution logic
2. `apps/dash/src/api/torrent-review-requests.ts` - Extend ReviewRequest type
3. `apps/dash/src/routes/dash/torrent/review-requests.tsx` - Display titles in table
4. `apps/dash/src/components/torrent-review-request-detail-sheet.tsx` - Display titles in detail view

## Verification

### Manual Testing

1. Load review requests page with mix of IMDB and AniDB targets
2. Verify titles appear in list table columns
3. Open detail sheet, verify all titles display correctly
4. Test with torrent hash not in torrent_info (no hash_title shown)
5. Test with unknown IMDB/AniDB IDs (empty titles, ID still shown)

### Edge Cases

- Review with empty prev_id (new mapping request)
- Review with empty mapping_id (rejection/deletion request)
- AniDB entry with 5+ title variants (all should display in detail, first in list)
- Hash not in torrent_info table (fallback to hash only)
