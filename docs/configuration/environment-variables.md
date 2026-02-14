# Environment Variables

Complete reference for all StremThru environment variables.

## Server

### `STREMTHRU_BASE_URL`

Base URL for StremThru. Used for generating callback URLs and links.

### `STREMTHRU_PORT`

Port to listen on.

- **Default:** `8080`

### `STREMTHRU_LOG_LEVEL`

Log level for the application.

| Value   | Description                       |
| ------- | --------------------------------- |
| `TRACE` | Most verbose                      |
| `DEBUG` | Debug information                 |
| `INFO`  | General information **(default)** |
| `WARN`  | Warnings                          |
| `ERROR` | Errors only                       |
| `FATAL` | Fatal errors only                 |

### `STREMTHRU_LOG_FORMAT`

Log output format.

| Value  | Description               |
| ------ | ------------------------- |
| `json` | JSON format **(default)** |
| `text` | Plain text format         |

### `STREMTHRU_DATA_DIR`

Directory for StremThru data files (database, etc.).

### `STREMTHRU_HTTP_PROXY`

HTTP proxy URL. Used for tunneling traffic when configured.

## Authentication {#authentication}

### `STREMTHRU_PROXY_AUTH`

Comma-separated list of credentials for proxy authorization. Supports two formats:

- Plain text: `username:password`
- Base64 encoded: `dXNlcm5hbWU6cGFzc3dvcmQ=`

**Example:**

```sh
STREMTHRU_PROXY_AUTH=user1:pass1,user2:pass2
```

### `STREMTHRU_AUTH_ADMIN`

Comma-separated list of admin usernames.

```sh
STREMTHRU_AUTH_ADMIN=admin_user
```

## Store {#store}

### `STREMTHRU_STORE_AUTH`

Comma-separated list of store credentials in `username:store_name:store_token` format.

For proxy-authorized requests, these credentials are used to authenticate with external stores.

If `username` is `*`, it is used as a fallback for users without explicit store credentials.

| Store       | `store_name` | `store_token`        |
| ----------- | ------------ | -------------------- |
| AllDebrid   | `alldebrid`  | `<api-key>`          |
| Debrider    | `debrider`   | `<api-key>`          |
| Debrid-Link | `debridlink` | `<api-key>`          |
| EasyDebrid  | `easydebrid` | `<api-key>`          |
| Offcloud    | `offcloud`   | `<email>:<password>` |
| PikPak      | `pikpak`     | `<email>:<password>` |
| Premiumize  | `premiumize` | `<api-key>`          |
| RealDebrid  | `realdebrid` | `<api-token>`        |
| TorBox      | `torbox`     | `<api-key>`          |

**Example:**

```sh
STREMTHRU_STORE_AUTH=user1:realdebrid:rd-token,user2:torbox:tb-key
```

### `STREMTHRU_STORE_TUNNEL`

Comma-separated list of tunnel configuration for stores in `store_name:tunnel_config` format.

::: warning
Only used when using StremThru to interact with the Store. Not affected by `STREMTHRU_TUNNEL`.
StremThru will _try_ to automatically adjust `STREMTHRU_TUNNEL` to reflect `STREMTHRU_STORE_TUNNEL`.
:::

| `tunnel_config` | Description         |
| --------------- | ------------------- |
| `true`          | Enable tunneling    |
| `false`         | Disable tunneling   |
| `api`           | Enable for API only |

If `store_name` is `*`, it is used as a fallback.

When enabled, `STREMTHRU_HTTP_PROXY` is used to tunnel traffic for the store.

**Example:**

```sh
STREMTHRU_STORE_TUNNEL=realdebrid:true,*:false
```

### `STREMTHRU_STORE_CONTENT_PROXY`

Comma-separated list of store content proxy configuration in `store_name:content_proxy_config` format.

| `content_proxy_config` | Description              |
| ---------------------- | ------------------------ |
| `true`                 | Enable content proxying  |
| `false`                | Disable content proxying |

If `store_name` is `*`, it is used as a fallback.

**Example:**

```sh
STREMTHRU_STORE_CONTENT_PROXY=*:true
```

### `STREMTHRU_STORE_CONTENT_CACHED_STALE_TIME`

Comma-separated list of stale time for cached/uncached content in `store_name:cached_stale_time:uncached_stale_time` format.

If `store_name` is `*`, it is used as a fallback.

**Example:**

```sh
STREMTHRU_STORE_CONTENT_CACHED_STALE_TIME=*:24h:8h
```

## Tunnel

### `STREMTHRU_TUNNEL`

::: warning
Cannot override `STREMTHRU_STORE_TUNNEL`.
:::

Comma-separated list of tunnel configuration in `hostname:tunnel_config` format.

| `tunnel_config` | Description                        |
| --------------- | ---------------------------------- |
| `true`          | Enable with `STREMTHRU_HTTP_PROXY` |
| `false`         | Disable                            |

If `hostname` is `*` and `tunnel_config` is `false`, only explicitly enabled hostnames will be tunneled.

## Content Proxy {#content-proxy}

### `STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT`

Comma-separated list of content proxy connection limits per user in `username:connection_limit` format.

If `username` is `*`, it is used as a fallback.

If `connection_limit` is `0`, no limit is applied.

## Stremio Addons

### `STREMTHRU_STREMIO_STORE_CATALOG_ITEM_LIMIT`

Maximum number of items to fetch for store catalog.

### `STREMTHRU_STREMIO_STORE_CATALOG_CACHE_TIME`

Cache time for store catalog.

### `STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_UPSTREAM_COUNT`

Maximum number of upstreams allowed on public instance.

### `STREMTHRU_STREMIO_WRAP_PUBLIC_MAX_STORE_COUNT`

Maximum number of stores allowed on public instance.

### `STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT`

Max timeout for newz indexer requests.

- **Default:** `15s`
- **Minimum:** `2s`
- **Maximum:** `60s`

### `STREMTHRU_STREMIO_TORZ_LAZY_PULL`

If `true`, Torz will pull from the public database in the background, so on first query it returns fewer results but responds faster.

### `STREMTHRU_STREMIO_TORZ_PUBLIC_MAX_STORE_COUNT`

Maximum number of stores allowed on public instance for Torz.

### `STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT`

Maximum number of lists allowed on public instance.

## Integrations {#integrations}

### AniList

#### `STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME`

Stale time for AniList list data. Example: `12h`.

### GitHub

#### `STREMTHRU_INTEGRATION_GITHUB_USER`

GitHub username.

#### `STREMTHRU_INTEGRATION_GITHUB_TOKEN`

GitHub Personal Access Token.

### Letterboxd

No environment variables required. Letterboxd integration works with public profiles.

### MDBList

#### `STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME`

Stale time for MDBList list data. Example: `12h`.

### TMDB

TMDB integration requires an [Access Token](https://www.themoviedb.org/settings/api).

#### `STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN`

API Read Access Token for TMDB.

#### `STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME`

Stale time for TMDB list data. Example: `12h`.

### Trakt

Trakt integration requires an [OAuth App](https://trakt.tv/oauth/applications).

The Redirect URI should point to the `/auth/trakt.tv/callback` endpoint of your [`STREMTHRU_BASE_URL`](#stremthru-base-url).

#### `STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID`

Client ID for Trakt OAuth App.

#### `STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET`

Client Secret for Trakt OAuth App.

#### `STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME`

Stale time for Trakt list data. Example: `12h`.

### TVDB

TVDB integration requires an [API Key](https://www.thetvdb.com/dashboard/account/apikey).

#### `STREMTHRU_INTEGRATION_TVDB_API_KEY`

API Key for TVDB.

#### `STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME`

Stale time for TVDB list data. Example: `12h`.

## Usenet {#usenet}

See [Usenet Configuration](./usenet) for all `STREMTHRU_NEWZ_*` environment variables.

## Security

### `STREMTHRU_VAULT_SECRET`

Secret for encrypting sensitive data.

## Database & Redis {#database-redis}

See [Database & Redis](./database) for details.

### `STREMTHRU_DATABASE_URI`

URI for the database. See [Database & Redis](./database).

### `STREMTHRU_REDIS_URI`

URI for Redis. See [Database & Redis](./database).

## Feature Flags

### `STREMTHRU_FEATURE`

See [Features](./features) for details.
