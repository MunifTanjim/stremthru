# Newz (Usenet)

Configuration for StremThru's Newz (Usenet) functionality.

## Feature Flag

The Newz feature requires the `vault` feature.

See [Features](./features) for more details on feature flags.

## Newz

### `STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM`

Maximum number of concurrent connections per stream.

- **Default:** `8`

**Example:**

```sh
STREMTHRU_NEWZ_MAX_CONNECTION_PER_STREAM=8
```

### `STREMTHRU_NEWZ_NZB_FILE_CACHE_SIZE`

Size of the NZB file cache.

- **Default:** `512MB`

**Example:**

```sh
STREMTHRU_NEWZ_NZB_FILE_CACHE_SIZE=512MB
```

::: info
Disk backed cache. Make sure you have enough disk space.
:::

### `STREMTHRU_NEWZ_NZB_FILE_CACHE_TTL`

TTL for cached NZB files.

- **Default:** `24h`
- **Minimum:** `6h`

**Example:**

```sh
STREMTHRU_NEWZ_NZB_FILE_CACHE_TTL=24h
```

### `STREMTHRU_NEWZ_NZB_LINK_MODE`

Comma-separated list of NZB link mode config, in `hostname:mode` format.

| `mode`     | Description                 |
| ---------- | --------------------------- |
| `proxy`    | Act as a proxy for NZB file |
| `redirect` | Redirect to NZB URL         |

If `hostname` is `*`, it is used as fallback.

- **Default:** `*:proxy`

**Example:**

```sh
STREMTHRU_NEWZ_NZB_LINK_MODE=*:proxy
```

### `STREMTHRU_NEWZ_NZB_FILE_MAX_SIZE`

Maximum NZB file size allowed.

- **Default:** `50MB`

**Example:**

```sh
STREMTHRU_NEWZ_NZB_FILE_MAX_SIZE=50MB
```

### `STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE`

Size of the Usenet segment cache.

- **Default:** `10GB`

**Example:**

```sh
STREMTHRU_NEWZ_SEGMENT_CACHE_SIZE=10GB
```

::: info
Disk backed cache. Make sure you have enough disk space.
:::

### `STREMTHRU_NEWZ_STREAM_BUFFER_SIZE`

Buffer size for streaming Usenet content.

- **Default:** `200MB`

**Example:**

```sh
STREMTHRU_NEWZ_STREAM_BUFFER_SIZE=200MB
```

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

```sh
STREMTHRU_NEWZ_QUERY_HEADER="
[*]
:prowlarr:

[movie]
:radarr:

[tv]
:sonarr:
Header-Name: Header-Value
"
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

```sh
STREMTHRU_NEWZ_GRAB_HEADER=":sabnzbd:"
```
