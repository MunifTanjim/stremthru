# Torz API

The Torz API provides endpoints for managing Torrent content through StremThru's store interface.

## Enums

### TorzStatus

| Value         | Description                    |
| ------------- | ------------------------------ |
| `cached`      | Content is cached on the store |
| `queued`      | Queued for download            |
| `downloading` | Currently downloading          |
| `processing`  | Processing after download      |
| `downloaded`  | Download complete              |
| `uploading`   | Currently uploading            |
| `failed`      | Download failed                |
| `invalid`     | Invalid Torrent                |
| `unknown`     | Unknown status                 |

## Endpoints

### Add Torz

**`POST /v0/store/torz`**

Add a torrent for download.

**Request** (magnet/torrent link):

```json
{
  "link": "string"
}
```

**Request** (torrent file upload):

`multipart/form-data` with a torrent file in the `file` field.

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "magnet": "string",
    "name": "string",
    "size": "int",
    "status": "TorzStatus",
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

### List Torz

**`GET /v0/store/torz`**

List torz on the user's account.

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
        "status": "TorzStatus",
        "private": "boolean",
        "added_at": "datetime"
      }
    ],
    "total_items": "int"
  }
}
```

### Get Torz

**`GET /v0/store/torz/{torzId}`**

Get a specific torz on the user's account.

**Path Parameters:**

- `torzId` — Torz ID

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "name": "string",
    "size": "int",
    "status": "TorzStatus",
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

### Remove Torz

**`DELETE /v0/store/torz/{torzId}`**

Remove a torz from the user's account.

**Path Parameters:**

- `torzId` — Torz ID

### Check Torz

**`GET /v0/store/torz/check`**

Check torrent hashes.

**Query Parameters:**

- `hash` — Comma-separated hashes (min `1`, max `500`)
- `sid` — Stremio stream ID

**Response:**

```json
{
  "data": {
    "items": [
      {
        "hash": "string",
        "magnet": "string",
        "status": "TorzStatus",
        "files": [
          {
            "index": "int",
            "link": "string",
            "name": "string",
            "path": "string",
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

### Generate Torz Link

**`POST /v0/store/torz/link/generate`**

Generate a direct link for a torz file link.

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

## Torznab Endpoint

**`GET /v0/torzab/api`**

StremThru exposes a Torznab-compatible API endpoint that can be used with tools like Prowlarr, Radarr, Sonarr etc.

**Authentication:** Uses the `STREMTHRU_AUTH` credentials, passed via the `apikey` query parameter.

**Output format:** Controlled by the `o` query parameter (`xml` default, `json` supported).
