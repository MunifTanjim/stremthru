# Torz API

The Newz API provides endpoints for managing Torrent content through StremThru's store interface.

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
| `invalid`     | Invalid torz                   |
| `unknown`     | Unknown status                 |

## Endpoints

::: tip Note
Coming Soon!
:::

## Torznab Endpoint

**`GET /v0/torzab/api`**

StremThru exposes a Torznab-compatible API endpoint that can be used with tools like Prowlarr, Radarr, Sonarr etc.

**Authentication:** Uses the `STREMTHRU_AUTH` credentials, passed via the `apikey` query parameter.

**Output format:** Controlled by the `o` query parameter (`xml` default, `json` supported).
