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

### Add Magnet

**`POST /v0/store/magnets`**

Add a magnet link for download.

**Request** (magnet link):

```json
{
  "magnet": "string"
}
```

**Request** (torrent file link):

```json
{
  "torrent": "string"
}
```

**Request** (torrent file upload):

`multipart/form-data` with a torrent file in the `torrent` field.

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "magnet": "string",
    "name": "string",
    "size": "int",
    "status": "MagnetStatus",
    "private": "boolean",
    "files": [
      {
        "index": "int",
        "link": "string",
        "name": "string",
        "path": "string",
        "size": "int",
        "video_hash": "string"
      }
    ],
    "added_at": "datetime"
  }
}
```

If `.status` is `downloaded`, `.files` will contain the list of files.

### List Magnets

**`GET /v0/store/magnets`**

List magnets on the user's account.

**Query Parameters:**

| Parameter | Default | Range       |
| --------- | ------- | ----------- |
| `limit`   | `100`   | `1` – `500` |
| `offset`  | `0`     | `0`+        |

**Response:**

```json
{
  "data": {
    "items": [
      {
        "id": "string",
        "hash": "string",
        "name": "string",
        "size": "int",
        "status": "MagnetStatus",
        "private": "boolean",
        "added_at": "datetime"
      }
    ],
    "total_items": "int"
  }
}
```

### Get Magnet

**`GET /v0/store/magnets/{magnetId}`**

Get a specific magnet on the user's account.

**Path Parameters:**

- `magnetId` — Magnet ID

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "name": "string",
    "size": "int",
    "status": "MagnetStatus",
    "private": "boolean",
    "files": [
      {
        "index": "int",
        "link": "string",
        "name": "string",
        "path": "string",
        "size": "int",
        "video_hash": "string"
      }
    ],
    "added_at": "datetime"
  }
}
```

### Remove Magnet

**`DELETE /v0/store/magnets/{magnetId}`**

Remove a magnet from the user's account.

**Path Parameters:**

- `magnetId` — Magnet ID

### Check Magnet

**`GET /v0/store/magnets/check`**

Check magnet link availability.

**Query Parameters:**

- `magnet` — Comma-separated magnet links (min `1`, max `500`)
- `sid` — Stremio stream ID

**Response:**

```json
{
  "data": {
    "items": [
      {
        "hash": "string",
        "magnet": "string",
        "status": "MagnetStatus",
        "files": [
          {
            "index": "int",
            "path": "string",
            "name": "string",
            "size": "int",
            "video_hash": "string"
          }
        ]
      }
    ]
  }
}
```

If `.status` is `cached`, `.files` will contain the list of files.

::: info Notes

- For `offcloud`, the `.files` list is always empty.
- If `.files[].index` is `-1`, the file index is unknown — rely on `.name` instead.
- If `.files[].size` is `-1`, the file size is unknown.
  :::

### Generate Link

**`POST /v0/store/link/generate`**

Generate a direct download link for a file.

**Request:**

```json
{
  "link": "string"
}
```

**Response:**

```json
{
  "data": {
    "link": "string"
  }
}
```

::: info Note
The generated direct link should be valid for 12 hours.
:::

## Newz Endpoints

The Store API supports Newz (Usenet). See the [Newz API](./newz) page for full documentation of all `/v0/store/newz/*` endpoints.

## Torz Endpoints

The Store API supports Torz (Torrent). See the [Torz API](./torz) page for full documentation of all `/v0/store/torz/*` endpoints.
