# Environment Variables

Complete reference for all StremThru environment variables.

## Server

### `STREMTHRU_BASE_URL`

Base URL for StremThru. Used for generating callback URLs and links.

- **Default:** `http://localhost:8080`

**Example:**

```sh
STREMTHRU_BASE_URL=http://localhost:8080
```

### `STREMTHRU_LISTEN_ADDR`

Address to listen on.

**Example:**

```sh
STREMTHRU_LISTEN_ADDR=127.0.0.1:8080
```

### `STREMTHRU_PORT`

Port to listen on.

- **Default:** `8080`

**Example:**

```sh
STREMTHRU_PORT=8080
```

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

**Example:**

```sh
STREMTHRU_LOG_LEVEL=INFO
```

### `STREMTHRU_LOG_FORMAT`

Log output format.

| Value  | Description               |
| ------ | ------------------------- |
| `json` | JSON format **(default)** |
| `text` | Plain text format         |

**Example:**

```sh
STREMTHRU_LOG_FORMAT=json
```

### `STREMTHRU_DATA_DIR`

Directory for StremThru data files (cache, database, temporary files etc.).

- **Default:** `./data`

**Example:**

```sh
STREMTHRU_DATA_DIR=./data
```

## Authentication {#authentication}

### `STREMTHRU_AUTH`

Comma-separated list of credentials for proxy authorization. Supports two formats:

- Plain text: `username:password`
- Base64 encoded: `dXNlcm5hbWU6cGFzc3dvcmQ=`

**Example:**

```sh
STREMTHRU_AUTH=user1:pass1,user2:pass2
```

### `STREMTHRU_AUTH_ADMIN`

Comma-separated list of admin usernames or credentials.

**Example:**

```sh
STREMTHRU_AUTH_ADMIN=user1,user3:pass3
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
STREMTHRU_STORE_AUTH=user1:realdebrid:rd-api-token,user2:torbox:tb-api-key
```

### `STREMTHRU_STORE_CONTENT_CACHED_STALE_TIME`

Comma-separated list of stale time for cached/uncached content in `store_name:cached_stale_time:uncached_stale_time` format.

If `store_name` is `*`, it is used as a fallback.

- **Default:** `*:24h:8h`

**Example:**

```sh
STREMTHRU_STORE_CONTENT_CACHED_STALE_TIME=*:24h:8h
```

### `STREMTHRU_STORE_CONTENT_PROXY`

Comma-separated list of store content proxy configuration in `store_name:content_proxy_config` format.

| `content_proxy_config` | Description              |
| ---------------------- | ------------------------ |
| `true`                 | Enable content proxying  |
| `false`                | Disable content proxying |

If `store_name` is `*`, it is used as a fallback.

- **Default:** `*:true`

**Example:**

```sh
STREMTHRU_STORE_CONTENT_PROXY=*:true
```

### `STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT`

Comma-separated list of content proxy connection limits per user in `username:connection_limit` format.

If `username` is `*`, it is used as a fallback.

If `connection_limit` is `0`, no limit is applied.

- **Default:** `*:0`

**Example:**

```sh
STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT=*:0
```

## Tunnel

### `STREMTHRU_HTTP_PROXY`

HTTP proxy URL. Used for tunneling traffic when configured.

**Example:**

```sh
STREMTHRU_HTTP_PROXY=http://proxy:8080
```

### `STREMTHRU_TUNNEL`

Comma-separated list of tunnel configuration in `hostname:tunnel_config` format.

| `tunnel_config` | Description                        |
| --------------- | ---------------------------------- |
| `true`          | Enable with `STREMTHRU_HTTP_PROXY` |
| `false`         | Disable                            |
| `<url>`         | Enable with specified `url`        |

If `hostname` is `*` and `tunnel_config` is `false`, only explicitly enabled hostnames will be tunneled.

**Example:**

```sh
STREMTHRU_TUNNEL=*:false,example.com:true
```

::: warning
Cannot override `STREMTHRU_STORE_TUNNEL`.
:::

### `STREMTHRU_STORE_TUNNEL`

Comma-separated list of tunnel configuration for stores in `store_name:tunnel_config` format.

| `tunnel_config` | Description         |
| --------------- | ------------------- |
| `true`          | Enable tunneling    |
| `false`         | Disable tunneling   |
| `api`           | Enable for API only |

If `store_name` is `*`, it is used as a fallback.

When enabled, `STREMTHRU_HTTP_PROXY` is used to tunnel traffic for the store.

- **Default:** `*:true`

**Example:**

```sh
STREMTHRU_STORE_TUNNEL=realdebrid:true,*:false
```

::: warning
Only used when using StremThru to interact with the Store. Not affected by `STREMTHRU_TUNNEL`.
StremThru will _try_ to automatically adjust `STREMTHRU_TUNNEL` to reflect `STREMTHRU_STORE_TUNNEL`.
:::

## Vault

### `STREMTHRU_VAULT_SECRET`

Secret for encrypting sensitive data.

**Example:**

```sh
STREMTHRU_VAULT_SECRET=my-super-secret-vault-key
```

## Database & Cache

See [Database & Cache](./database-and-cache) for details.

## Stremio Addons

See [Stremio Addons](./stremio-addons) for all `STREMTHRU_STREMIO_*` environment variables.

## Integrations

See [Integrations](./integrations) for all `STREMTHRU_INTEGRATION_*` environment variables.

## Features

See [Features](./features) for details.

## Newz

See [Newz Configuration](./newz) for all `STREMTHRU_NEWZ_*` environment variables.

## Torz

See [Torz Configuration](./torz) for all `STREMTHRU_TORZ_*` environment variables.
