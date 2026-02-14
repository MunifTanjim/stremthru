# Usenet

Configuration for StremThru's Usenet (Newz) functionality.

## Feature Flag

The Usenet feature requires the `stremio_newz` feature flag, which in turn requires `vault`.

```sh
STREMTHRU_FEATURE=+vault,+stremio_newz
```

See [Features](./features) for more details on feature flags.

## Stremio Newz

### `STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT`

Max timeout for newz indexer requests.

- **Default:** `15s`
- **Minimum:** `2s`
- **Maximum:** `60s`

## Newz

### `STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM`

Maximum number of concurrent connections per stream.

- **Default:** `8`

### `STREMTHRU_NEWZ_NZB_CACHE_SIZE`

Size of the NZB file cache.

- **Default:** `512MB`

### `STREMTHRU_NEWZ_NZB_CACHE_TTL`

TTL for cached NZB files.

- **Default:** `24h`
- **Minimum:** `6h`

### `STREMTHRU_NEWZ_NZB_LINK_MODE`

Comma-separated list of NZB link mode config, in `hostname:mode` format.

| `mode`     | Description                 |
| ---------- | --------------------------- |
| `proxy`    | Act as a proxy for NZB file |
| `redirect` | Redirect to NZB URL         |

If `hostname` is `*`, it is used as fallback.

- **Default:** `*:proxy`

### `STREMTHRU_NEWZ_NZB_MAX_FILE_SIZE`

Maximum NZB file size allowed.

- **Default:** `50MB`

### `STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE`

Size of the Usenet segment cache.

- **Default:** `10GB`

### `STREMTHRU_NEWZ_STREAM_BUFFER_SIZE`

Buffer size for streaming Usenet content.

- **Default:** `200MB`

### `STREMTHRU_NEWZ_QUERY_HEADER`

Custom headers for indexer query requests.

**Format:**

```
[query_type]
:preset_name:
Header-Name: Header-Value
```

`query_type` can be `*` (fallback), `movie`, `tv`.

**Available presets:**

- `chrome`
- `prowlarr`
- `radarr`
- `sonarr`

**Example:**

```
[*]
:prowlarr:

[movie]
:radarr:

[tv]
:sonarr:
Header-Name: Header-Value
```

### `STREMTHRU_NEWZ_GRAB_HEADER`

Custom headers for NZB file download requests.

**Format:**

```
:preset_name:
Header-Name: Header-Value
```

**Available presets:**

- `chrome`
- `nzbget`
- `sabnzbd`

**Example:**

```
:sabnzbd:
```
