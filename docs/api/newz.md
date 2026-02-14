# Newz API

The Newz API provides endpoints for managing Usenet (NZB) content through StremThru's store interface.

## NewzStatus

| Value         | Description                 |
| ------------- | --------------------------- |
| `cached`      | Content is cached and ready |
| `queued`      | Queued for download         |
| `downloading` | Currently downloading       |
| `processing`  | Processing after download   |
| `downloaded`  | Download complete           |
| `failed`      | Download failed             |
| `invalid`     | Invalid NZB                 |
| `unknown`     | Unknown status              |

## Endpoints

### Add Newz

**`POST /v0/store/newz`**

Add an NZB for download.

**Request** (NZB link):

```json
{
  "link": "string"
}
```

**Request** (NZB file upload): `multipart/form-data` with an NZB file in the `file` field.

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "status": "NewzStatus"
  }
}
```

### List Newz

**`GET /v0/store/newz`**

List newz on the user's account.

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
        "status": "NewzStatus",
        "added_at": "datetime"
      }
    ],
    "total_items": "int"
  }
}
```

### Get Newz

**`GET /v0/store/newz/{newzId}`**

Get a specific newz on the user's account.

**Path Parameters:**

- `newzId` — Newz ID

**Response:**

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "name": "string",
    "size": "int",
    "status": "NewzStatus",
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

### Remove Newz

**`DELETE /v0/store/newz/{newzId}`**

Remove a newz from the user's account.

**Path Parameters:**

- `newzId` — Newz ID

### Check Newz

**`GET /v0/store/newz/check`**

Check NZB hashes.

**Query Parameters:**

- `hash` — Comma-separated hashes (min `1`, max `500`)

**Response:**

```json
{
  "data": {
    "items": [
      {
        "hash": "string",
        "status": "NewzStatus",
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

### Generate Newz Link

**`POST /v0/store/newz/link/generate`**

Generate a direct link for a newz file link.

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

::: tip
The generated direct link should be valid for 12 hours.
:::

## Newznab Endpoint

**`GET /v0/newznab/api`**

StremThru exposes a Newznab-compatible API endpoint that can be used with tools like Prowlarr, Radarr, and Sonarr.

**Authentication:** Uses the same proxy auth credentials, passed via the `apikey` query parameter.

**Supported operations:**

| `t` parameter | Description      |
| ------------- | ---------------- |
| `caps`        | Get capabilities |
| `search`      | General search   |
| `tvsearch`    | TV search        |
| `movie`       | Movie search     |
| `get`         | Download NZB     |

**Output format:** Controlled by the `o` query parameter (`xml` default, `json` supported).
