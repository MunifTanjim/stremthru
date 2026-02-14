# API

StremThru provides a REST API for interacting with stores, proxying content, and mapping content IDs.

## Authentication

API requests are authenticated using the `X-StremThru-Authorization` header with Basic auth.

```
X-StremThru-Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
```

The credentials are checked against the [`STREMTHRU_PROXY_AUTH`](/configuration/environment-variables#stremthru-proxy-auth) configuration.

## Store Selection

For store endpoints, use the `X-StremThru-Store-Name` header to specify which store to use. If not provided, the first store configured for the user via `STREMTHRU_STORE_AUTH` is used.

For non-proxy-authorized requests, the store is authenticated using:

- `X-StremThru-Store-Authorization` header
- `Authorization` header (forwarded to the external store)

## Endpoints

| Section          | Description                       |
| ---------------- | --------------------------------- |
| [Proxy](./proxy) | Proxify URLs                      |
| [Store](./store) | Manage magnets and generate links |
| [Meta](./meta)   | Content ID mapping                |

## Enums

### MagnetStatus

| Value         | Description                    |
| ------------- | ------------------------------ |
| `cached`      | Content is cached on the store |
| `queued`      | Magnet is queued for download  |
| `downloading` | Content is being downloaded    |
| `processing`  | Content is being processed     |
| `downloaded`  | Content is fully downloaded    |
| `uploading`   | Content is being uploaded      |
| `failed`      | Download failed                |
| `invalid`     | Invalid magnet                 |
| `unknown`     | Unknown status                 |

### NewzStatus

| Value         | Description                    |
| ------------- | ------------------------------ |
| `cached`      | Content is cached on the store |
| `queued`      | Item is queued for download    |
| `downloading` | Content is being downloaded    |
| `processing`  | Content is being processed     |
| `downloaded`  | Content is fully downloaded    |
| `failed`      | Download failed                |
| `invalid`     | Invalid item                   |
| `unknown`     | Unknown status                 |

### UserSubscriptionStatus

| Value     |
| --------- |
| `expired` |
| `premium` |
| `trial`   |
