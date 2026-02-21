# Torz (Torrent)

Configuration for StremThru's Torz (Torrent) functionality.

## Torz

### `STREMTHRU_TORZ_TORRENT_FILE_CACHE_SIZE`

Size of the torrent file cache.

- **Default:** `256MB`

**Example:**

```sh
STREMTHRU_TORZ_TORRENT_FILE_CACHE_SIZE=256MB
```

::: info
Disk backed cache. Make sure you have enough disk space.
:::

### `STREMTHRU_TORZ_TORRENT_FILE_CACHE_TTL`

TTL for cached torrent files.

- **Default:** `6h`

**Example:**

```sh
STREMTHRU_TORZ_TORRENT_FILE_CACHE_TTL=6h
```

### `STREMTHRU_TORZ_TORRENT_FILE_MAX_SIZE`

Maximum torrent file size allowed.

- **Default:** `1MB`

**Example:**

```sh
STREMTHRU_TORZ_TORRENT_FILE_MAX_SIZE=1MB
```
