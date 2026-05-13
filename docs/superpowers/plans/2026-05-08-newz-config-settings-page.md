# Newz Config in Settings Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Display full usenet config (values + indexer headers) in the Newz section of `/dash/settings/config`

**Architecture:** Add `useUsenetConfig` hook call to settings/config page, expand NewzSection to use CollapsibleConfigSection with config values and HeaderTable for indexer request headers.

**Tech Stack:** React, TanStack Query, TypeScript, Tailwind CSS

---

## File Structure

- Modify: `apps/dash/src/routes/dash/settings/config.tsx` - Expand NewzSection component

---

### Task 1: Import useUsenetConfig and UsenetConfig type

**Files:**

- Modify: `apps/dash/src/routes/dash/settings/config.tsx:1-14`

- [ ] **Step 1: Add import for useUsenetConfig**

Add to imports at top of file:

```tsx
import { type UsenetConfig, useUsenetConfig } from "@/api/usenet";
```

- [ ] **Step 2: Verify no TypeScript errors**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

---

### Task 2: Add HeaderTable component

**Files:**

- Modify: `apps/dash/src/routes/dash/settings/config.tsx`

- [ ] **Step 1: Add queryTypeLabels constant after imports**

Add after line 14 (after imports):

```tsx
const queryTypeLabels: Record<string, string> = {
  "*": "Any/Fallback",
  movie: "Movie",
  tv: "TV",
};
```

- [ ] **Step 2: Add HeaderTable component**

Add after `ConfigEntry` function (after line 81):

```tsx
function HeaderTable({ headers }: { headers: Record<string, string> }) {
  const entries = Object.entries(headers);
  if (entries.length === 0) {
    return <div className="text-muted-foreground text-sm">None</div>;
  }
  return (
    <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
      {entries.map(([key, value]) => (
        <div className="contents" key={key}>
          <div className="text-muted-foreground font-medium">{key}</div>
          <div className="truncate">{value}</div>
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 3: Verify no TypeScript errors**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

---

### Task 3: Replace NewzSection with expanded version

**Files:**

- Modify: `apps/dash/src/routes/dash/settings/config.tsx:249-267`

- [ ] **Step 1: Replace NewzSection function**

Replace the existing `NewzSection` function (lines 249-267) with:

```tsx
function NewzSection({ newz }: { newz: ConfigData["newz"] }) {
  const { data: usenetConfig, isLoading } = useUsenetConfig();

  if (newz.disabled) return null;

  const configFields: { key: keyof UsenetConfig; label: string }[] = [
    { key: "nzb_cache_size", label: "NZB Cache Size" },
    { key: "nzb_cache_ttl", label: "NZB Cache TTL" },
    { key: "nzb_max_file_size", label: "NZB Max File Size" },
    { key: "segment_cache_size", label: "Segment Cache Size" },
    { key: "stream_buffer_size", label: "Stream Buffer Size" },
    { key: "max_connection_per_stream", label: "Max Connection Per Stream" },
  ];

  const settingsCount = configFields.length;

  return (
    <CollapsibleConfigSection
      gradient="from-orange-500 to-amber-500"
      icon="N"
      settingsCount={settingsCount}
      title="Newz"
    >
      {isLoading ? (
        <div className="text-muted-foreground text-sm">Loading...</div>
      ) : usenetConfig ? (
        <>
          <div className="grid grid-cols-2 gap-x-4">
            {configFields.map(({ key, label }) => (
              <ConfigEntry
                key={key}
                label={label}
                value={String(usenetConfig[key])}
              />
            ))}
          </div>
          <div className="mt-4 border-t pt-4">
            <h3 className="mb-3 text-sm font-semibold">
              Indexer Request Headers
            </h3>
            <div className="flex flex-col gap-4">
              <div>
                <h4 className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
                  Query Headers
                </h4>
                <div className="flex flex-col gap-3">
                  {Object.entries(
                    usenetConfig.indexer_request_header.query,
                  ).map(([queryType, headers]) => (
                    <div key={queryType}>
                      <div className="mb-1 text-sm font-medium">
                        {queryTypeLabels[queryType] ?? queryType}
                      </div>
                      <HeaderTable headers={headers} />
                    </div>
                  ))}
                </div>
              </div>
              <div>
                <h4 className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
                  Grab Headers
                </h4>
                <HeaderTable
                  headers={usenetConfig.indexer_request_header.grab}
                />
              </div>
            </div>
          </div>
        </>
      ) : null}
    </CollapsibleConfigSection>
  );
}
```

- [ ] **Step 2: Verify no TypeScript errors**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors

---

### Task 4: Verify in browser

- [ ] **Step 1: Start dev server**

Run: `cd apps/dash && npm run dev`

- [ ] **Step 2: Navigate to settings config page**

Open: `http://localhost:5173/dash/settings/config`

- [ ] **Step 3: Verify Newz section**

Check:

- Newz section shows as collapsible with "6 settings" badge
- Click to expand shows all 6 config values in 2-column grid
- Indexer Request Headers section appears below config values
- Query Headers grouped by type (Movie, TV, Any/Fallback)
- Grab Headers shows as table
- Empty headers show "None"

- [ ] **Step 4: Verify loading state**

Hard refresh page and observe "Loading..." text briefly appears while usenet config fetches.

---

### Task 5: Cleanup

- [ ] **Step 1: Remove Link import if unused**

Check if `Link` from `@tanstack/react-router` is still used elsewhere in the file. If only used in old NewzSection, remove from imports.

- [ ] **Step 2: Final TypeScript check**

Run: `cd apps/dash && npx tsc --noEmit`
Expected: No errors
