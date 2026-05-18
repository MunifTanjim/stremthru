# Review Request Titles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Display human-readable titles for torrent hashes and IMDB/AniDB IDs on the review request page.

**Architecture:** Batch-resolve titles in the backend API handler. Collect unique hashes/IDs from review list, query torrent_info/imdb_title/anidb tables in bulk, merge titles into response. Frontend displays titles alongside IDs.

**Tech Stack:** Go backend, React/TypeScript frontend, TanStack Query

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/dash/api/torrent_review_requests.go` | Add `MappingReviewWithTitles` type, helper functions, update handler |
| `apps/dash/src/api/torrent-review-requests.ts` | Extend `ReviewRequest` type with title fields |
| `apps/dash/src/routes/dash/torrent/review-requests.tsx` | Display titles in list table columns |
| `apps/dash/src/components/torrent-review-request-detail-sheet.tsx` | Display titles in detail view |

---

### Task 1: Backend - Add Response Type and Helpers

**Files:**
- Modify: `internal/dash/api/torrent_review_requests.go`

- [ ] **Step 1: Add new imports**

Add these imports at the top of the file:

```go
"github.com/MunifTanjim/stremthru/internal/anidb"
"github.com/MunifTanjim/stremthru/internal/imdb_title"
```

Note: `anidb` and `imdb_torrent` are already imported. Add `imdb_title` if not present.

- [ ] **Step 2: Add MappingReviewWithTitles type**

Add after the existing `ListTorrentReviewRequestsResponse` struct:

```go
type MappingReviewWithTitles struct {
	torrent_mapping_review.MappingReview
	HashTitle       string   `json:"hash_title,omitempty"`
	PrevIdTitles    []string `json:"prev_id_titles,omitempty"`
	MappingIdTitles []string `json:"mapping_id_titles,omitempty"`
}
```

- [ ] **Step 3: Update response type**

Change `ListTorrentReviewRequestsResponse` from:

```go
type ListTorrentReviewRequestsResponse struct {
	Items      []torrent_mapping_review.MappingReview `json:"items"`
	NextCursor string                                 `json:"next_cursor"`
}
```

To:

```go
type ListTorrentReviewRequestsResponse struct {
	Items      []MappingReviewWithTitles `json:"items"`
	NextCursor string                    `json:"next_cursor"`
}
```

- [ ] **Step 4: Add helper functions**

Add before `handleListTorrentReviewRequests`:

```go
func collectReviewHashes(items []torrent_mapping_review.MappingReview) []string {
	seen := make(map[string]struct{})
	hashes := []string{}
	for _, item := range items {
		if item.Hash != "" {
			if _, ok := seen[item.Hash]; !ok {
				seen[item.Hash] = struct{}{}
				hashes = append(hashes, item.Hash)
			}
		}
	}
	return hashes
}

func collectReviewIdsByTarget(items []torrent_mapping_review.MappingReview) (imdbIds, anidbIds []string) {
	seenImdb := make(map[string]struct{})
	seenAnidb := make(map[string]struct{})

	for _, item := range items {
		ids := []string{item.PrevId, item.MappingId}
		for _, id := range ids {
			if id == "" {
				continue
			}
			switch item.Target {
			case torrent_mapping_review.MappingTargetIMDB:
				if _, ok := seenImdb[id]; !ok {
					seenImdb[id] = struct{}{}
					imdbIds = append(imdbIds, id)
				}
			case torrent_mapping_review.MappingTargetAniDB:
				if _, ok := seenAnidb[id]; !ok {
					seenAnidb[id] = struct{}{}
					anidbIds = append(anidbIds, id)
				}
			}
		}
	}
	return imdbIds, anidbIds
}

func buildIMDBTitleMap(titles []imdb_title.IMDBTitle) map[string]string {
	m := make(map[string]string)
	for _, t := range titles {
		m[t.TId] = t.Title
	}
	return m
}

func buildAniDBTitleMap(titles anidb.AniDBTitles) map[string][]string {
	m := make(map[string][]string)
	for _, t := range titles {
		m[t.TId] = append(m[t.TId], t.Value)
	}
	return m
}

func getTitlesForId(target torrent_mapping_review.MappingTarget, id string, imdbMap map[string]string, anidbMap map[string][]string) []string {
	if id == "" {
		return nil
	}
	switch target {
	case torrent_mapping_review.MappingTargetIMDB:
		if title, ok := imdbMap[id]; ok {
			return []string{title}
		}
	case torrent_mapping_review.MappingTargetAniDB:
		if titles, ok := anidbMap[id]; ok {
			return titles
		}
	}
	return nil
}
```

- [ ] **Step 5: Verify build**

Run: `go build ./internal/dash/api/...`
Expected: Build succeeds (handlers not updated yet, but types compile)

---

### Task 2: Backend - Update Handler to Resolve Titles

**Files:**
- Modify: `internal/dash/api/torrent_review_requests.go`

- [ ] **Step 1: Update handleListTorrentReviewRequests**

Replace the function body after fetching results. Change from:

```go
	items := result.Items
	if items == nil {
		items = []torrent_mapping_review.MappingReview{}
	}

	SendData(w, r, 200, ListTorrentReviewRequestsResponse{
		Items:      items,
		NextCursor: result.NextCursor,
	})
```

To:

```go
	rawItems := result.Items
	if rawItems == nil {
		rawItems = []torrent_mapping_review.MappingReview{}
	}

	// Collect unique identifiers
	hashes := collectReviewHashes(rawItems)
	imdbIds, anidbIds := collectReviewIdsByTarget(rawItems)

	// Batch fetch titles (errors logged, not fatal)
	log := GetReqCtx(r).Log

	hashTitles := make(map[string]torrent_info.TorrentInfo)
	if len(hashes) > 0 {
		if titles, err := torrent_info.GetByHashes(hashes); err != nil {
			log.Warn("failed to fetch torrent info for hashes", "error", err)
		} else {
			hashTitles = titles
		}
	}

	imdbTitleMap := make(map[string]string)
	if len(imdbIds) > 0 {
		if titles, err := imdb_title.ListByIds(imdbIds); err != nil {
			log.Warn("failed to fetch imdb titles", "error", err)
		} else {
			imdbTitleMap = buildIMDBTitleMap(titles)
		}
	}

	anidbTitleMap := make(map[string][]string)
	if len(anidbIds) > 0 {
		if titles, err := anidb.GetTitlesByIds(anidbIds); err != nil {
			log.Warn("failed to fetch anidb titles", "error", err)
		} else {
			anidbTitleMap = buildAniDBTitleMap(titles)
		}
	}

	// Merge titles into response
	items := make([]MappingReviewWithTitles, len(rawItems))
	for i, review := range rawItems {
		items[i] = MappingReviewWithTitles{
			MappingReview:   review,
			PrevIdTitles:    getTitlesForId(review.Target, review.PrevId, imdbTitleMap, anidbTitleMap),
			MappingIdTitles: getTitlesForId(review.Target, review.MappingId, imdbTitleMap, anidbTitleMap),
		}
		if info, ok := hashTitles[review.Hash]; ok {
			items[i].HashTitle = info.TorrentTitle
		}
	}

	SendData(w, r, 200, ListTorrentReviewRequestsResponse{
		Items:      items,
		NextCursor: result.NextCursor,
	})
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 3: Test API endpoint manually**

Run: `curl -s http://localhost:8080/dash/api/torrent/review-requests | jq '.data.items[0] | {hash, hash_title, prev_id, prev_id_titles, mapping_id, mapping_id_titles}'`

Expected: Response includes new title fields (may be null if no titles found)

---

### Task 3: Frontend - Extend ReviewRequest Type

**Files:**
- Modify: `apps/dash/src/api/torrent-review-requests.ts`

- [ ] **Step 1: Add title fields to ReviewRequest type**

Change from:

```typescript
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
};
```

To:

```typescript
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
  hash_title?: string;
  prev_id_titles?: string[];
  mapping_id_titles?: string[];
};
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd apps/dash && pnpm tsc --noEmit`
Expected: No errors

---

### Task 4: Frontend - Update List Table

**Files:**
- Modify: `apps/dash/src/routes/dash/torrent/review-requests.tsx`

- [ ] **Step 1: Import Tooltip components**

Add to imports:

```typescript
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
```

- [ ] **Step 2: Update hash column**

Change the hash column accessor from:

```typescript
col.accessor("hash", {
  cell: ({ getValue }) => (
    <span className="font-mono text-xs truncate max-w-[120px] inline-block">
      {getValue()}
    </span>
  ),
  header: "Hash",
}),
```

To:

```typescript
col.accessor("hash", {
  cell: ({ getValue, row }) => {
    const hash = getValue();
    const title = row.original.hash_title;
    return (
      <div className="flex flex-col gap-0.5 max-w-[200px]">
        <span className="font-mono text-xs truncate">{hash}</span>
        {title && (
          <span className="text-xs text-muted-foreground truncate">{title}</span>
        )}
      </div>
    );
  },
  header: "Hash",
}),
```

- [ ] **Step 3: Update prev_id column**

Change from:

```typescript
col.accessor("prev_id", {
  cell: ({ getValue }) => {
    const value = getValue();
    if (!value) return <span className="text-muted-foreground">—</span>;
    return <span className="font-mono text-xs">{value}</span>;
  },
  header: "Prev ID",
}),
```

To:

```typescript
col.accessor("prev_id", {
  cell: ({ getValue, row }) => {
    const value = getValue();
    if (!value) return <span className="text-muted-foreground">—</span>;
    const titles = row.original.prev_id_titles;
    const firstTitle = titles?.[0];
    const content = (
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-xs">{value}</span>
        {firstTitle && (
          <span className="text-xs text-muted-foreground truncate max-w-[150px]">
            {firstTitle}
          </span>
        )}
      </div>
    );
    if (titles && titles.length > 1) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent className="max-w-[300px]">
            <ul className="text-xs space-y-1">
              {titles.map((t, i) => (
                <li key={i}>{t}</li>
              ))}
            </ul>
          </TooltipContent>
        </Tooltip>
      );
    }
    return content;
  },
  header: "Prev ID",
}),
```

- [ ] **Step 4: Update mapping_id column**

Change from:

```typescript
col.accessor("mapping_id", {
  cell: ({ getValue }) => {
    const value = getValue();
    if (!value) return <span className="text-muted-foreground">—</span>;
    return <span className="font-mono text-xs">{value}</span>;
  },
  header: "Suggested ID",
}),
```

To:

```typescript
col.accessor("mapping_id", {
  cell: ({ getValue, row }) => {
    const value = getValue();
    if (!value) return <span className="text-muted-foreground">—</span>;
    const titles = row.original.mapping_id_titles;
    const firstTitle = titles?.[0];
    const content = (
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-xs">{value}</span>
        {firstTitle && (
          <span className="text-xs text-muted-foreground truncate max-w-[150px]">
            {firstTitle}
          </span>
        )}
      </div>
    );
    if (titles && titles.length > 1) {
      return (
        <Tooltip>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent className="max-w-[300px]">
            <ul className="text-xs space-y-1">
              {titles.map((t, i) => (
                <li key={i}>{t}</li>
              ))}
            </ul>
          </TooltipContent>
        </Tooltip>
      );
    }
    return content;
  },
  header: "Suggested ID",
}),
```

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd apps/dash && pnpm tsc --noEmit`
Expected: No errors

---

### Task 5: Frontend - Update Detail Sheet

**Files:**
- Modify: `apps/dash/src/components/torrent-review-request-detail-sheet.tsx`

- [ ] **Step 1: Update Hash DetailRow**

Change from:

```tsx
<DetailRow label="Hash">
  <span className="font-mono text-xs break-all">{request.hash}</span>
</DetailRow>
```

To:

```tsx
<DetailRow label="Hash">
  <div className="flex flex-col gap-1">
    <span className="font-mono text-xs break-all">{request.hash}</span>
    {request.hash_title && (
      <span className="text-sm text-muted-foreground">{request.hash_title}</span>
    )}
  </div>
</DetailRow>
```

- [ ] **Step 2: Update Current Mapping ID DetailRow**

Change from:

```tsx
<DetailRow label="Current Mapping ID">
  <span className="font-mono">{request.prev_id || "—"}</span>
</DetailRow>
```

To:

```tsx
<DetailRow label="Current Mapping ID">
  <div className="flex flex-col gap-1">
    <span className="font-mono">{request.prev_id || "—"}</span>
    {request.prev_id_titles && request.prev_id_titles.length > 0 && (
      <ul className="text-sm text-muted-foreground space-y-0.5">
        {request.prev_id_titles.map((title, i) => (
          <li key={i}>{title}</li>
        ))}
      </ul>
    )}
  </div>
</DetailRow>
```

- [ ] **Step 3: Update Suggested Mapping ID DetailRow**

Change from:

```tsx
<DetailRow label="Suggested Mapping ID">
  <span className="font-mono">{request.mapping_id || "—"}</span>
</DetailRow>
```

To:

```tsx
<DetailRow label="Suggested Mapping ID">
  <div className="flex flex-col gap-1">
    <span className="font-mono">{request.mapping_id || "—"}</span>
    {request.mapping_id_titles && request.mapping_id_titles.length > 0 && (
      <ul className="text-sm text-muted-foreground space-y-0.5">
        {request.mapping_id_titles.map((title, i) => (
          <li key={i}>{title}</li>
        ))}
      </ul>
    )}
  </div>
</DetailRow>
```

- [ ] **Step 4: Verify TypeScript compiles**

Run: `cd apps/dash && pnpm tsc --noEmit`
Expected: No errors

---

### Task 6: Manual Verification

- [ ] **Step 1: Start the dev server**

Run: `make dev` (or equivalent)

- [ ] **Step 2: Load review requests page**

Navigate to `/dash/torrent/review-requests` in browser

- [ ] **Step 3: Verify list table**

Check:
- Hash column shows truncated hash + torrent title below
- Prev ID column shows ID + first title below
- Suggested ID column shows ID + first title below
- For AniDB entries with multiple titles, hover to see tooltip with all titles

- [ ] **Step 4: Verify detail sheet**

Click a row to open detail sheet. Check:
- Hash section shows full hash + full torrent title
- Current Mapping ID shows ID + all titles listed
- Suggested Mapping ID shows ID + all titles listed

- [ ] **Step 5: Test edge cases**

Verify:
- Review with no prev_id shows "—" (no title section)
- Review with no mapping_id shows "—" (no title section)
- Hash not in torrent_info shows hash only (no title)
- Unknown IMDB/AniDB ID shows ID only (no title)