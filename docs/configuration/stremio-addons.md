# Stremio Addons

## StremThru List

### `STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT`

Maximum number of lists allowed on public instance.

- **Default:** `10`

**Example:**

```sh
STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT=10
```

## StremThru Newz

### `STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT`

Max timeout for newz indexer requests.

- **Default:** `15s`
- **Minimum:** `2s`
- **Maximum:** `60s`

**Example:**

```sh
STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT=15s
```

## StremThru Torz

### `STREMTHRU_STREMIO_TORZ_INDEXER_MAX_TIMEOUT`

Max timeout for Torz indexer requests.

- **Default:** `10s`
- **Minimum:** `2s`
- **Maximum:** `60s`

**Example:**

```sh
STREMTHRU_STREMIO_TORZ_INDEXER_MAX_TIMEOUT=10s
```

### `STREMTHRU_STREMIO_TORZ_LAZY_PULL`

If `true`, Torz will pull from the public database in the background, so on first query it returns fewer results but responds faster.

**Example:**

```sh
STREMTHRU_STREMIO_TORZ_LAZY_PULL=false
```

### `STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_INDEXER_COUNT`

Maximum number of indexers allowed on public instance for Torz.

- **Default:** `2`

**Example:**

```sh
STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_INDEXER_COUNT=2
```

### `STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_STORE_COUNT`

Maximum number of stores allowed on public instance for Torz.

- **Default:** `3`

**Example:**

```sh
STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_STORE_COUNT=3
```

## StremThru Store

### `STREMTHRU_STREMIO_STORE_CATALOG_ITEM_LIMIT`

Maximum number of items to fetch for store catalog.

- **Default:** `2000`

**Example:**

```sh
STREMTHRU_STREMIO_STORE_CATALOG_ITEM_LIMIT=2000
```

### `STREMTHRU_STREMIO_STORE_CATALOG_CACHE_TIME`

Cache time for store catalog.

- **Default:** `10m`
- **Minimum:** `1m`

**Example:**

```sh
STREMTHRU_STREMIO_STORE_CATALOG_CACHE_TIME=10m
```

## StremThru Wrap

### `STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_UPSTREAM_COUNT`

Maximum number of upstreams allowed on public instance.

- **Default:** `5`

**Example:**

```sh
STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_UPSTREAM_COUNT=5
```

### `STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_STORE_COUNT`

Maximum number of stores allowed on public instance.

- **Default:** `3`

**Example:**

```sh
STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_STORE_COUNT=3
```
