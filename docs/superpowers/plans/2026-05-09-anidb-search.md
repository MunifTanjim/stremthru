# AniDB Search Component Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add AniDBSearch autocomplete component to torrent mapping page, mirroring IMDBSearch functionality.

**Architecture:** Backend endpoint uses existing `anidb.SearchIdsByTitle()` and `anidb.GetTitlesByIds()`. Frontend component mirrors IMDBSearch pattern with Popover+Command UI. Integration replaces plain Input with AniDBSearch when AniDB tab + by-title mode.

**Tech Stack:** Go (backend), React + TanStack Query (frontend), cmdk (Command component)

---

## File Structure

| File                                            | Action | Responsibility                      |
| ----------------------------------------------- | ------ | ----------------------------------- |
| `internal/dash/api/anidb.go`                    | Create | AniDB autocomplete endpoint handler |
| `internal/dash/router.go`                       | Modify | Register AniDB endpoints            |
| `apps/dash/src/api/anidb.ts`                    | Create | `useAniDBAutocomplete` hook         |
| `apps/dash/src/components/anidb-search.tsx`     | Create | AniDBSearch component               |
| `apps/dash/src/routes/dash/torrent/mapping.tsx` | Modify | Integrate AniDBSearch               |

---

### Task 1: Backend - AniDB Autocomplete Endpoint

**Files:**

- Create: `internal/dash/api/anidb.go`

- [ ] **Step 1: Create anidb.go with autocomplete handler**

```go
package dash_api

import (
	"net/http"
	"regexp"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

type AniDBAutocompleteItem struct {
	Id     string `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Season string `json:"season"`
	Year   string `json:"year"`
}

var anidbIdPattern = regexp.MustCompile(`^\d+$`)

func handleGetAniDBAutocomplete(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		SendData(w, r, 200, []AniDBAutocompleteItem{})
		return
	}

	var ids []string

	if anidbIdPattern.MatchString(query) {
		ids = []string{query}
	} else {
		var err error
		ids, err = anidb.SearchIdsByTitle(query, nil, 0, 10)
		if err != nil {
			SendError(w, r, err)
			return
		}
	}

	if len(ids) == 0 {
		SendData(w, r, 200, []AniDBAutocompleteItem{})
		return
	}

	titles, err := anidb.GetTitlesByIds(ids)
	if err != nil {
		SendError(w, r, err)
		return
	}

	// Dedupe by TId, prefer "main" title type, fallback to first
	itemById := make(map[string]AniDBAutocompleteItem)
	preferredType := map[string]int{"main": 3, "official": 2, "synonym": 1}

	for _, t := range titles {
		existing, exists := itemById[t.TId]
		currentPriority := preferredType[t.TType]
		existingPriority := 0
		if exists {
			existingPriority = preferredType[existing.Type]
		}

		if !exists || currentPriority > existingPriority {
			itemById[t.TId] = AniDBAutocompleteItem{
				Id:     t.TId,
				Title:  t.Value,
				Type:   t.Type,
				Season: t.Season,
				Year:   t.Year,
			}
		}
	}

	// Preserve order from search results
	items := make([]AniDBAutocompleteItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := itemById[id]; ok {
			items = append(items, item)
		}
	}

	SendData(w, r, 200, items)
}

func AddAniDBEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/anidb/autocomplete", authed(handleGetAniDBAutocomplete))
}
```

- [ ] **Step 2: Verify file compiles**

Run from project root:

```bash
go build ./internal/dash/api/
```

Expected: No errors

---

### Task 2: Backend - Register AniDB Endpoints

**Files:**

- Modify: `internal/dash/router.go:41`

- [ ] **Step 1: Add AniDB endpoints registration**

Add after line 41 (`dash_api.AddIMDBEndpoints(router)`):

```go
	dash_api.AddAniDBEndpoints(router)
```

- [ ] **Step 2: Verify backend compiles**

Run from project root:

```bash
go build ./...
```

Expected: No errors

---

### Task 3: Frontend API - useAniDBAutocomplete Hook

**Files:**

- Create: `apps/dash/src/api/anidb.ts`

- [ ] **Step 1: Create anidb.ts with type and hook**

```typescript
import { useQuery } from "@tanstack/react-query";

import { api } from "@/lib/api";

export type AniDBTitle = {
  id: string;
  title: string;
  type: string;
  season: string;
  year: string;
};

export function useAniDBAutocomplete(query: string = "") {
  return useQuery({
    enabled: Boolean(query),
    queryFn: async () => {
      const { data } = await api<AniDBTitle[]>(
        `/anidb/autocomplete?query=${encodeURIComponent(query)}`,
      );
      return data;
    },
    queryKey: ["/anidb/autocomplete", query],
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run from `apps/dash`:

```bash
npx tsc --noEmit
```

Expected: No errors

---

### Task 4: Frontend Component - AniDBSearch

**Files:**

- Create: `apps/dash/src/components/anidb-search.tsx`

- [ ] **Step 1: Create anidb-search.tsx component**

```tsx
import { CommandLoading } from "cmdk";
import { SearchIcon } from "lucide-react";
import { useState } from "react";

import { AniDBTitle, useAniDBAutocomplete } from "@/api/anidb";
import { useDebouncedValue } from "@/hooks/use-debounced-value";

import { Button } from "./ui/button";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "./ui/command";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemHeader,
  ItemTitle,
} from "./ui/item";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";

export function AniDBSearch({
  onSelect,
  triggerLabel = "Search...",
}: {
  onSelect: (title: AniDBTitle) => void;
  triggerLabel?: string;
}) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [_searchQuery, setSearchQuery] = useState("");
  const searchQuery = useDebouncedValue(_searchQuery, 300);
  const autocompleteResults = useAniDBAutocomplete(searchQuery);

  return (
    <Popover onOpenChange={setSearchOpen} open={searchOpen}>
      <PopoverTrigger asChild>
        <Button
          aria-expanded={searchOpen}
          className="w-full justify-between"
          role="combobox"
          variant="outline"
        >
          <span className="truncate">{triggerLabel}</span>
          <SearchIcon className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
        <Command shouldFilter={false}>
          <CommandInput
            onValueChange={setSearchQuery}
            placeholder="Search AniDB titles..."
            value={_searchQuery}
          />
          <CommandList>
            {autocompleteResults.isLoading && _searchQuery && (
              <CommandLoading>Searching...</CommandLoading>
            )}
            <CommandEmpty>AniDB Titles</CommandEmpty>
            {autocompleteResults.data?.map((title) => (
              <CommandItem
                key={title.id}
                onSelect={async () => {
                  onSelect(title);
                  setSearchQuery("");
                  setSearchOpen(false);
                }}
                value={title.id}
              >
                <Item className="w-full p-0" size="sm">
                  <ItemHeader className="text-muted-foreground flex justify-between text-xs">
                    <div>{title.type || "Unknown"}</div>
                    <div>{title.id}</div>
                  </ItemHeader>
                  <ItemContent>
                    <ItemTitle>{title.title}</ItemTitle>
                    <ItemDescription>
                      <span className="text-muted-foreground text-xs">
                        {title.season && `S${title.season}`}
                        {title.season && title.year && " · "}
                        {title.year}
                      </span>
                    </ItemDescription>
                  </ItemContent>
                </Item>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run from `apps/dash`:

```bash
npx tsc --noEmit
```

Expected: No errors

---

### Task 5: Integration - Add AniDBSearch to Mapping Page

**Files:**

- Modify: `apps/dash/src/routes/dash/torrent/mapping.tsx`

- [ ] **Step 1: Add imports**

Add after line 7 (`import { IMDBTitle } from "@/api/imdb";`):

```typescript
import { AniDBTitle } from "@/api/anidb";
```

Add after line 17 (`import { IMDBSearch } from "@/components/imdb-search";`):

```typescript
import { AniDBSearch } from "@/components/anidb-search";
```

- [ ] **Step 2: Add selectedAniDBTitle state**

Add after line 146 (`const [selectedTitle, setSelectedTitle] = useState<IMDBTitle | null>(null);`):

```typescript
const [selectedAniDBTitle, setSelectedAniDBTitle] = useState<AniDBTitle | null>(
  null,
);
```

- [ ] **Step 3: Update onClearSearch to clear AniDB state**

Modify `onClearSearch` function (around line 185-191) to:

```typescript
const onClearSearch = () => {
  setInput("");
  setSearch("");
  setSelectedTitle(null);
  setSelectedAniDBTitle(null);
  setSeason("");
  setEpisode("");
};
```

- [ ] **Step 4: Add AniDBSearch in search section**

Replace the search section (lines 245-324). The condition at line 246 changes from:

```tsx
        {tab === "imdb" && mode === "by-title" ? (
```

To handle both IMDB and AniDB by-title modes. Replace lines 245-324 with:

```tsx
{
  /* Search */
}
<div className="flex flex-wrap gap-2">
  {mode === "by-title" ? (
    tab === "imdb" ? (
      <>
        <div className="w-64">
          <IMDBSearch
            onSelect={(title) => {
              setSelectedTitle(title);
              setInput(title.id);
              setSeason("");
              setEpisode("");
              if (!SERIES_TYPES.includes(title.type)) {
                setSearch(title.id);
              }
            }}
            triggerLabel={
              selectedTitle
                ? `${selectedTitle.title} (${selectedTitle.id})`
                : undefined
            }
          />
        </div>
        {isSeries && (
          <>
            <Input
              className="w-24"
              onChange={(e) => setSeason(e.target.value)}
              placeholder="Season"
              type="number"
              value={season}
            />
            <Input
              className="w-24"
              onChange={(e) => setEpisode(e.target.value)}
              placeholder="Episode"
              type="number"
              value={episode}
            />
            <Button
              onClick={() => {
                let stremId = selectedTitle?.id || "";
                if (season) {
                  stremId += `:${season}`;
                  if (episode) {
                    stremId += `:${episode}`;
                  }
                }
                setSearch(stremId);
              }}
            >
              <SearchIcon className="mr-1 size-4" />
              Search
            </Button>
          </>
        )}
      </>
    ) : (
      <div className="w-64">
        <AniDBSearch
          onSelect={(title) => {
            setSelectedAniDBTitle(title);
            setInput(`anidb:${title.id}`);
            setSearch(`anidb:${title.id}`);
          }}
          triggerLabel={
            selectedAniDBTitle
              ? `${selectedAniDBTitle.title} (${selectedAniDBTitle.id})`
              : undefined
          }
        />
      </div>
    )
  ) : (
    <>
      <Input
        className="max-w-md"
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") onSearch();
        }}
        placeholder={placeholder}
        value={input}
      />
      <Button onClick={onSearch}>
        <SearchIcon className="mr-1 size-4" />
        Search
      </Button>
    </>
  )}
  {search && (
    <Button onClick={onClearSearch} variant="outline">
      Clear
    </Button>
  )}
</div>;
```

- [ ] **Step 5: Verify TypeScript compiles**

Run from `apps/dash`:

```bash
npx tsc --noEmit
```

Expected: No errors

---

### Task 6: Manual Verification

- [ ] **Step 1: Start backend**

Run from project root:

```bash
go run .
```

- [ ] **Step 2: Start frontend dev server**

Run from `apps/dash`:

```bash
npm run dev
```

- [ ] **Step 3: Test AniDBSearch functionality**

1. Open browser to http://localhost:3000/dash/torrent/mapping
2. Click "AniDB" tab
3. Ensure mode is "By AniDB"
4. Click the search button - popover should open
5. Type an anime title (e.g., "naruto")
6. Verify autocomplete results appear with Type, ID, Title, Season/Year
7. Select a result
8. Verify search triggers with `anidb:{id}` format
9. Verify results table populates (if mappings exist)
10. Click "Clear" to reset
11. Switch to "By Torrent" mode - verify plain input appears instead
