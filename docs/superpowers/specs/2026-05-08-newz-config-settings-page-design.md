# Newz Config in Settings Page

## Context

The Newz section in `/dash/settings/config` currently only displays a link to `/dash/usenet/config`. User wants to show the full usenet config inline, including indexer request headers, to reduce navigation.

## Design

### Data Source

Add `useUsenetConfig()` hook call to fetch full usenet config data:

- Endpoint: `/usenet/config`
- Returns: `UsenetConfig` type with config values + `indexer_request_header`

### Component Changes

**File:** `apps/dash/src/routes/dash/settings/config.tsx`

1. Import `useUsenetConfig` from `@/api/usenet`
2. Replace `NewzSection` component with expanded version
3. Use `CollapsibleConfigSection` pattern (like Torz section)

### Layout

```
[N] Newz ───────────────────── [6 settings] [>]

  NZB Cache Size         | Segment Cache Size
  NZB Cache TTL          | Stream Buffer Size
  NZB Max File Size      | Max Connection Per Stream

  ─── Indexer Request Headers ───

  Query Headers
    Movie
      [header-key]: [value]
    TV
      [header-key]: [value]
    Any/Fallback
      [header-key]: [value]

  Grab Headers
    [header-key]: [value]
```

### Config Fields

Display using `ConfigEntry` component:
| Field | Label |
|-------|-------|
| nzb_cache_size | NZB Cache Size |
| nzb_cache_ttl | NZB Cache TTL |
| nzb_max_file_size | NZB Max File Size |
| segment_cache_size | Segment Cache Size |
| stream_buffer_size | Stream Buffer Size |
| max_connection_per_stream | Max Connection Per Stream |

### Indexer Request Headers

Reuse `HeaderTable` pattern from `usenet/config.tsx`:

- Query headers grouped by type (Movie, TV, Any/Fallback)
- Grab headers as single table
- Show "None" if empty

### Loading State

Show "Loading..." text inside section if `useUsenetConfig` is still loading.

### Styling

- Gradient: `from-orange-500 to-amber-500` (existing)
- Icon: "N" (existing)
- Collapsible with settings count badge

## Verification

1. Navigate to `/dash/settings/config`
2. Expand Newz section
3. Verify all 6 config values display correctly
4. Verify indexer headers display (Query by type + Grab)
5. Verify loading state shows while fetching
6. Verify section hidden when `newz.disabled === true`
