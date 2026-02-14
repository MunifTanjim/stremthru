# Proxy API

The Proxy API creates proxified URLs that route content through StremThru.

## Authentication

Authorization is checked against the [`STREMTHRU_PROXY_AUTH`](/configuration/environment-variables#stremthru-proxy-auth) configuration.

- If the `token` query parameter is present, the proxified link will **not** be encrypted.
- If the `X-StremThru-Authorization` header is present, the proxified link will be encrypted.

## Endpoints

### `GET /v0/proxy`

Proxify one or more URLs via query parameters.

**Query Parameters:**

| Parameter        | Description                                                    |
| ---------------- | -------------------------------------------------------------- |
| `url`            | URL to proxify _(can be specified multiple times)_             |
| `exp`            | Expiration time duration _(optional)_                          |
| `req_headers[i]` | Headers for the URL at position `i` _(optional)_               |
| `req_headers`    | Fallback headers if `req_headers[i]` is missing _(optional)_   |
| `filename[i]`    | Filename for the URL at position `i` _(optional)_              |
| `token`          | Token for authorization _(optional)_                           |
| `redirect`       | Redirect to proxified URL, valid for single `url` _(optional)_ |

### `POST /v0/proxy`

Proxify one or more URLs via form body.

**Request:** `x-www-form-urlencoded` body with the same fields as the GET endpoint (except `redirect`).

**Response:**

```json
{
  "items": ["string"],
  "total_items": "int"
}
```

::: info
Detailed documentation coming soon — contributions welcome.
:::
