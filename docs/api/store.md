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
