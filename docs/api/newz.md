# Newz API

The Newz API provides endpoints for managing Usenet content through StremThru's store interface.

::: info Feature Required
This API requires the [`newz`](/configuration/features) feature to be enabled.
:::

## Enums

### NewzStatus

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

**Request** (NZB file upload):

`multipart/form-data` with an NZB file in the `file` field.

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

::: info Note
The generated direct link should be valid for 12 hours.
:::

## Newznab Endpoint

**`GET /v0/newznab/api`**

::: info Feature Required
[`vault`](/configuration/features) feature needs to be enabled and configured properly.
:::

StremThru exposes a Newznab-compatible API endpoint that can be used with tools like Prowlarr, Radarr, Sonarr etc.

**Authentication:** Uses the `STREMTHRU_AUTH` credentials, passed via the `apikey` query parameter.

**Output format:** Controlled by the `o` query parameter (`xml` default, `json` supported).

## WebDAV Endpoint

**`/v0/webdav/newz/`**

::: info Feature Required
[`vault`](/configuration/features) feature needs to be enabled and configured properly.
:::

StremThru exposes a WebDAV endpoint for browsing and streaming Newz content using any WebDAV-compatible client.

**Authentication:** HTTP Basic Auth using `STREMTHRU_AUTH` credentials.

**Directory Structure:**

```
/v0/webdav/newz/
└── {newz-name}/
    └── {file-name}
```

**File Filtering:** Only files matching [`STREMTHRU_WEBDAV_FILE_EXT_FILTER`](/configuration/#stremthru-webdav-file-ext-filter) are exposed.

## SABnzbd Endpoint

**`GET /v0/sabnzbd/api`**

::: info Feature Required
[`vault`](/configuration/features) feature needs to be enabled and configured properly.
:::

StremThru exposes a SABnzbd-compatible API endpoint that can be used with tools like Sonarr, Radarr, Prowlarr etc. to send NZBs to StremThru for processing.

**Authentication:** Uses the [`STREMTHRU_AUTH_SABNZBD`](/configuration/newz#stremthru-auth-sabnzbd) credentials, passed via the `apikey` query parameter.

### Supported Modes

#### `addurl`

Queue an NZB URL for download.

**Query Parameters:**

| Parameter  | Required | Description                              |
| ---------- | -------- | ---------------------------------------- |
| `name`     | Yes      | NZB URL to download                      |
| `nzbname`  | No       | Display name for the NZB                 |
| `cat`      | No       | Category (`*` is treated as none)        |
| `priority` | No       | Priority integer (`-100` treated as `0`) |
| `password` | No       | Password for the NZB                     |

**Success Response:**

```json
{
  "status": true,
  "nzo_ids": ["SABnzbd_nzo_<id>"]
}
```

**Error Response:**

```json
{
  "status": false,
  "error": "<message>"
}
```
