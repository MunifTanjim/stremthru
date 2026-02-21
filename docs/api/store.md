# Store API

The Store API provides a unified interface for interacting with external debrid stores.

## Store Selection

For store endpoints, use the `X-StremThru-Store-Name` header to specify which store to use. If not provided, the first store configured for the user via `STREMTHRU_STORE_AUTH` is used.

For non-proxy-authorized requests, the store is authenticated using:

- `X-StremThru-Store-Authorization` header
- `Authorization` header

Values for these headers will be forwarded to the external store.

## Enums

### UserSubscriptionStatus

| Value     |
| --------- |
| `expired` |
| `premium` |
| `trial`   |

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

## Endpoints

### Get User

**`GET /v0/store/user`**

Get information about the authenticated user.

**Response:**

```json
{
  "data": {
    "id": "string",
    "email": "string",
    "subscription_status": "UserSubscriptionStatus",
    "has_usenet": "boolean"
  }
}
```

## Newz Endpoints

The Store API supports Newz (Usenet). See the [Newz API](./newz) page for full documentation of all `/v0/store/newz/*` endpoints.

## Torz Endpoints

The Store API supports Torz (Torrent). See the [Torz API](./torz) page for full documentation of all `/v0/store/torz/*` endpoints.
