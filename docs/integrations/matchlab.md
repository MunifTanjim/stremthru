# MatchLab

[MatchLab](https://github.com/Dizzybog/MatchLab) is an **optional external torrent parser and IMDb/AniDB matcher** for StremThru. It provides verified release and per-file identities, including anime episode ranges. No MatchLab service is needed for the default StremThru installation.

## Before you begin

- Deploy a compatible MatchLab instance and make its HTTP API accessible from the StremThru container or host.
- Confirm `GET /health/ready` returns HTTP 200 on the MatchLab base URL.
- Apply the normal StremThru database migrations before starting workers. The integration adds state and retry columns on supported PostgreSQL and SQLite databases.
- If your installation has custom IMDb/AniDB triggers **outside** the StremThru schema, disable conflicting automatic remapping yourself before enabling the external matching engine.

MatchLab is a separate application. It is **not** provided by StremThru's Docker image and is **not required** for native parsing and matching.

## Configure

Set these environment variables on the **StremThru** service:

```env
STREMTHRU_INTEGRATION_MATCHLAB_URL=http://matchlab:8765
STREMTHRU_INTEGRATION_MATCHLAB_TIMEOUT=10s
STREMTHRU_INTEGRATION_MATCHLAB_CONCURRENCY=8
```

`STREMTHRU_INTEGRATION_MATCHLAB_URL` must be a reachable HTTP(S) base URL with no `/parse` or `/match` suffix. The alternative `MATCHLAB_URL` is also recognized. The default timeout is `10s`; concurrent calls default to `8` (allowed: 1–64).

For Docker Compose, connect both applications to a network where `matchlab` resolves. **Do not replace PostgreSQL's entrypoint or install curl in the PostgreSQL container** for this integration.

The external matcher takes ownership of parsing, IMDb/AniDB matching, and verified per-video-file identities only when its URL is configured. It uses the existing torrent tables, not a replacement database.

## Choose the matching mode

| Mode | MatchLab URL | `STREMTHRU_FEATURE` | Behavior |
| --- | --- | --- | --- |
| Native (default) | Unset | Default | Existing Go-PTT and native IMDb/AniDB workers |
| External matching | Set | Default | MatchLab owns release parsing and matching; original Torz request path |
| External + optimized Torz | Set | `+performance` | MatchLab owns matching; fast indexed-file reads and asynchronous peer/indexer refresh |
| External + anime datasets | Set | `+anime` | MatchLab matching plus original anime metadata synchronization |
| External + both | Set | `+anime,+performance` | Both optional features |

The `+anime` flag is **not** required for MatchLab to populate verified AniDB identities. It only enables the existing local anime dataset workers.

`+performance` is disabled by default. Its optimized file lookup requires MatchLab's precomputed identities; without a MatchLab URL, StremThru automatically uses the original Torz implementation even when the flag remains set.

### Debrid cache checks

With `+performance`, torrent discovery is cache-first, but debrid availability is **checked synchronously** for the unique visible hashes after filtering. The stream response waits for the configured debrid provider(s) to finish checking cache status; an unavailable provider is not treated as a confirmed cache miss. P2P requests do not invoke debrid checks.

Peer pulling and user indexer refresh remain in bounded background queues. A release found only by a new background search may appear on a subsequent request.

## How identities are stored

The parser receives the complete raw torrent title. The mapping worker also sends each video file's combined torrent title and path, enabling independent identities for multi-season packs or mixed anime. Only verified single-episode coordinates are saved on individual files. Mapping changes to `torrent_info`, `imdb_torrent`, `anidb_torrent`, and `torrent_stream` commit transactionally.

Anime crosswalk segments can use different season/episode conventions from IMDb. The integration does not manufacture conversions when MatchLab cannot verify them. See the [mapping design and limitations](/matchlab).

## Switching MatchLab off

Remove `STREMTHRU_INTEGRATION_MATCHLAB_URL` and `MATCHLAB_URL`, then restart StremThru:

1. The original Go-PTT parser and native matching workers become active again.
2. Previously MatchLab-parsed torrents are reparsed incrementally, not through a destructive bulk reset.
3. For each affected torrent, stale external IMDb/AniDB and file identities are cleared as it enters the native mapping path.
4. Native workers progressively rebuild the mappings; some previously matched results may temporarily be absent.

Existing native mappings are left alone. Do not truncate torrent tables or reset millions of rows to switch engines.

## Monitoring and troubleshooting

Check the worker dashboard and logs for the `parse-torrent` and `map-imdb-torrent` jobs. With MatchLab configured, the mapping worker is shown as **Map IMDb/AniDB via MatchLab**.

You can inspect the mapping backlog using:

```sql
SELECT count(*)
FROM torrent_info
WHERE matchlab_mapped_at IS NULL
  AND t_title = parser_input;
```

This is a read-only query. On large databases, it may be expensive; use only when needed. Avoid bulk-clearing `matchlab_mapped_at` to force reprocessing.

If MatchLab is unavailable, mapping work is deferred for retry; StremThru does **not** silently switch to native matching while the URL is configured. If an episode pack has not yet received verified file identities, optimized Torz mode will not guess the episode from filenames during the HTTP request.

## Further details

- [MatchLab mapping design, database behavior, and tests](/matchlab)
- [Feature flags](/configuration/features)
- [Torz configuration](/configuration/torz)
- [MatchLab source code](https://github.com/Dizzybog/MatchLab)
