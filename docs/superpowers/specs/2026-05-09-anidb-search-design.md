# AniDB Search Component

## Context

The torrent mapping page has IMDB and AniDB tabs. IMDB tab uses `IMDBSearch` component for autocomplete search when in "by-title" mode. AniDB tab currently uses a plain text input. This creates inconsistent UX - users expect the same search experience for both providers.

## Goal

Create `AniDBSearch` component mirroring `IMDBSearch` pattern, with backend autocomplete support.

## Design

### Backend: `/anidb/autocomplete` endpoint

**File:** `internal/dash/api/anidb.go` (new)

**Endpoint:** `GET /anidb/autocomplete?query=<string>`

**Logic:**

1. If query matches `^\d+$` (numeric), treat as AniDB ID directly
2. Otherwise, call `anidb.SearchIdsByTitle(query, nil, 0, 10)`
3. Fetch title details via `anidb.GetTitlesByIds(ids)`
4. Dedupe by TId, prefer "main" or "official" TType titles
5. Return array of `{ id, title, type, season, year }`

**Response type:**

```go
type AniDBAutocompleteItem struct {
    Id     string `json:"id"`
    Title  string `json:"title"`
    Type   string `json:"type"`
    Season string `json:"season"`
    Year   string `json:"year"`
}
```

### Frontend API

**File:** `apps/dash/src/api/anidb.ts` (new)

```typescript
type AniDBTitle = {
  id: string;
  title: string;
  type: string;
  season: string;
  year: string;
};

function useAniDBAutocomplete(query: string): UseQueryResult<AniDBTitle[]>;
```

### Frontend Component

**File:** `apps/dash/src/components/anidb-search.tsx` (new)

Structure mirrors `IMDBSearch`:

- Popover with Button trigger
- Command with CommandInput (debounced 300ms)
- CommandList with results
- Each result displays:
  - Header: Type (left), ID (right)
  - Content: Title
  - Description: Season, Year

**Props:**

```typescript
{
  onSelect: (title: AniDBTitle) => void;
  triggerLabel?: string;
}
```

### Integration

**File:** `apps/dash/src/routes/dash/torrent/mapping.tsx`

Changes:

1. Add `selectedAniDBTitle` state (type `AniDBTitle | null`)
2. Import `AniDBSearch` component
3. When `tab === "anidb" && mode === "by-title"`:
   - Render `AniDBSearch` instead of plain Input
   - OnSelect: set `selectedAniDBTitle`, set `input` to `anidb:{id}`
4. Clear `selectedAniDBTitle` in `onClearSearch`

## Verification

1. Start dev server
2. Navigate to /dash/torrent/mapping
3. Select AniDB tab, ensure mode is "by-title"
4. Type anime title in search
5. Verify autocomplete results appear with Type, ID, Title, Season, Year
6. Select a result
7. Verify search executes with `anidb:{id}` format
8. Verify results table populates
