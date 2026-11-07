# MatchLab: torrent parser and identity matcher

StremThru can optionally use [MatchLab](https://github.com/Dizzybog/MatchLab) as an external torrent parser and authoritative IMDb/AniDB matcher. With no URL configured, StremThru retains its original Go-PTT, IMDb and AniDB workers. For installation and configuration, see the [MatchLab integration guide](/integrations/matchlab).

## Enable

MatchLab works with StremThru's default disabled `anime` feature. It supplies its own verified IMDb/AniDB identities. Enabling `+anime` activates local anime dataset sync and related workers, **not** MatchLab itself. The optional `+performance` flag enables asynchronous peer/indexer refresh and fast indexed-file lookup when MatchLab is configured. Debrid availability checks remain synchronous before responding with streams.

Set the following in the StremThru container environment:

```env
STREMTHRU_INTEGRATION_MATCHLAB_URL=http://matchlab:8765
STREMTHRU_INTEGRATION_MATCHLAB_TIMEOUT=10s
STREMTHRU_INTEGRATION_MATCHLAB_CONCURRENCY=8
```

`MATCHLAB_URL` is supported as an alias for the URL. Both containers must be on a Docker network that resolves `matchlab`. The endpoint must be the **base URL**; do not include `/match`, `/parse` or credentials. MatchLab must return HTTP 200 from `/health/ready` before processing. Run migrations before starting workers.

In Docker Compose, configure these variables on the StremThru service and connect it to the MatchLab API network. No extra PostgreSQL container packages, entrypoint changes or custom triggers are required.

## Data flow and source of truth

1. Torrent discovery/import upserts `torrent_info` with raw `t_title` and optionally `torrent_stream` file paths `p`.
2. The parse worker calls MatchLab `POST /parse` with the **full raw title**. It stores the ordinary parser fields in `torrent_info`, including normalized title, year, seasons and episodes. The parser version marks externally parsed data. This parsing is incremental; enabling MatchLab does **not** bulk-clear the parser input of every historical torrent.
3. The mapping worker calls MatchLab `POST /match` with raw `t_title`, then separately with `t_title + "/" + torrent_stream.p` **for every video file**. For anime, a separate path-only match can disambiguate mixed anime from the package name. A successful MatchLab result is never replaced by a local IMDb fuzzy search or AniDB fuzzy title matcher.
4. File-specific `sid` and `asid` are written only when the respective start and end identities are the **same single coordinate**. Multi-episode files are never incorrectly labeled as a single episode. A missing field does not erase an existing independently supplied field.
5. `imdb_torrent.tid` contains the verified bare IMDb title IDs (e.g. `tt0460681`), one per independently identified work. Mixed anime packages can therefore have multiple IMDb IDs for one hash.
6. `torrent_info.seasons` is a comma-separated ascending list (`1,2,3`). If a torrent spans multiple seasons, its `episodes` field is **always empty** because that field does not encode which season owns each number. If it is a single season, `episodes` is populated across the verified range (e.g. `1,2,3,4,5,6,7`), capped to StremThru's 2,048-value field.
7. `anidb_torrent` receives one mapping per AniDB ID, mapping convention (`ani`, verified `tv`, or verified absolute `abs`) and season. The mapped `ep_start`, `ep_end`, and `eps` are derived from verified coordinates. When endpoints represent a crosswalk-spanning anime range, StremThru reads `POST /debug/match` to retrieve **verified contiguous crosswalk segments** rather than guessing where a season or anime ID changes. `eps` is used for exact episode lookups; missing middle episodes are not inferred. Unknown AniDB season is represented by `-1`, following StremThru's lookup convention.
8. A single SQL transaction updates `torrent_info`, replaces authoritative rows for the hash in `imdb_torrent` and `anidb_torrent`, writes verified file IDs, and marks `matchlab_mapped_at`. The torrent-stream cache is invalidated after commit. Transport/metadata errors roll back all changes and leave the torrent retryable.

**Episode numbering:** MatchLab's verified anime season coordinates can follow the TVDB-style crosswalk, which is not guaranteed to equal canonical IMDb numbering. The application preserves the returned identity convention and never manufactures an unverified conversion.

### Picking the series in a pack

Matching every video path is stricter than sampling only one: the highest-episode file in a single season and highest-season file in a multi-season pack are included automatically, even when filenames are not ordered. Torrent-level identification is accepted only when it agrees with the independently identified file title. With multiple different IMDb IDs, the torrent has no single global seasons/episodes list; each video retains its own SID/ASID. This prevents a mixed anime bundle from inheriting the wrong identity from its folder.

### Compatibility

- **URL absent:** the original Go-PTT parser and local IMDb mapper run, and the native AniDB mapper runs when `+anime` is enabled. Rows previously parsed by MatchLab are reparsed incrementally. Before their native parser version is stored, the parser worker removes only those hashes' old MatchLab IMDb/AniDB mappings and file SID/ASID identities, making them eligible for native remapping. All other torrents and existing native mappings remain intact. This handover can temporarily reduce results while the local workers rebuild mappings.
- **URL present:** local IMDb and AniDB matching workers are not run. External requests are bounded (default eight simultaneous HTTP calls; worker processes at most four torrent hashes in parallel). A MatchLab outage does **not** silently fall back to the local matcher.
- **Existing data:** all existing hashes initially have `matchlab_mapped_at IS NULL`, so they are progressively reprocessed. Fresh file records clear this marker for their hash, so new `torrent_stream` data is re-matched. You can monitor the remaining backlog with:

```sql
SELECT count(*) FROM torrent_info
WHERE matchlab_mapped_at IS NULL
  AND t_title = parser_input;
```

- **Custom PostgreSQL triggers:** some installations have custom IMDb/anime triggers created outside StremThru's migrations. Audit and disable those separately *before* turning on the external mapper to avoid a second identity engine modifying the same records. This fork does not automatically drop user-defined triggers.
- **Existing metadata sync:** the IMDb/AniDB source-data sync jobs are unchanged. MatchLab can map without waiting for their daily synchronization. AniDB title metadata is used only to obtain AniDB season labels when available.
- **Anime flag off:** this is StremThru's default. MatchLab still writes verified AniDB mappings and ASID file identities. If AniDB title/season datasets are unavailable, the mapper stores an unknown season (`-1`), and AniDB lookups accept unknown-season mappings. IMDb anime-only Torznab indexers first use existing `anime_id_map` or verified per-file SID/ASID identities, then ask MatchLab to classify the requested IMDb title if needed. A MatchLab classification failure does not block regular indexers.
- **Anime flag on:** local AnimeAPI, AniDB titles and crosswalk synchronization remain active as before. MatchLab owns external identity matching in either mode; native anime matching is not started when MatchLab is enabled.
- **Scaling:** use a network-local MatchLab URL, sensible client concurrency for your server capacity, and monitoring of parser/mapping backlog and error logs. The initial historical mapping can produce substantial API traffic.
- **Incremental safety:** `matchlab_file_revision` increments when torrent files change. An in-flight mapping is committed only if its file revision is still current, avoiding lost file updates. Failed parsing and matching are deferred through `matchlab_parse_retry_at` and `matchlab_retry_at`; newly updated files are immediately eligible for reprocessing.
- **File identities:** request-derived Stremio `sid/asid` tagging is disabled in MatchLab mode. A completed MatchLab pass removes obsolete file IDs where no verified identity exists, instead of leaking stale mappings from the old engine.

## Validation

The unit tests cover HTTP response validation, malformed configurations, single-/multi-season metadata, mixed anime IDs, verified anime crosswalk segments, and combined file paths. From the project root run:

```sh
STREMTHRU_ENV=test go test --tags 'fts5' ./internal/matchlab ./internal/worker ./internal/torrent_info ./internal/torrent_stream ./internal/torznab ./internal/stremio/torz
GOFLAGS='-tags=fts5' go vet ./internal/matchlab ./internal/worker ./internal/torrent_info ./internal/torrent_stream
test -z "$(gofmt -l internal/matchlab internal/worker/map_matchlab_torrent.go internal/worker/map_matchlab_torrent_test.go internal/worker/torrent_parser.go internal/torrent_info/db.go internal/torrent_stream/db.go)"
```

Then verify against a **test** PostgreSQL database with synthetic hash rows representing a movie, normal episode, single-season pack, multi-season pack, anime episode ranges (including 1-7), anime split across TV seasons and mixed anime. Verify the query paths for IMDb and AniDB episodes, not just that writes succeeded.

## Known boundary

The public MatchLab API returns only verified start/end coordinates, not complete release file membership. For actual file lists, StremThru enumerates only the video files that are present in `torrent_stream`. If only a torrent name is known (no file list), MatchLab season-level metadata is used, with the documented limitations of that source. No unmatched or unavailable external identity is invented.
