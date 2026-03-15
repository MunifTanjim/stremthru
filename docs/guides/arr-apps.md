# *arr Apps Integration

StremThru exposes Torznab and Newznab compatible API endpoints, making it usable as an indexer in Prowlarr, Radarr, and Sonarr.

## Torznab

The Torznab endpoint provides torrent search results from StremThru's torrent database.

**Endpoint:** `/v0/torznab/api`

**Authentication:** Not required.

### Supported Features

| Feature | Details |
| --- | --- |
| Search Types | General search, TV search, Movie search |
| Categories | Movies (`2000`), TV (`5000`) |
| Search Parameters | `q` (query), `imdbid`, `season`, `ep`, `year` |
| Output Formats | XML (default), JSON (`o=json`) |

### Prowlarr Setup

1. Go to **Indexers > Add Indexer**
2. Select **Generic Torznab**
3. Fill in:

| Field | Value |
| --- | --- |
| Name | `StremThru Torznab` |
| URL | `http://<stremthru-host>/v0/torznab/api` |
| Categories | Select `Movies` and/or `TV` |

4. Click **Test** then **Save**

### Radarr / Sonarr Setup

1. Go to **Settings > Indexers > Add**
2. Select **Torznab**
3. Fill in:

| Field | Value |
| --- | --- |
| Name | `StremThru Torznab` |
| URL | `http://<stremthru-host>/v0/torznab/api` |
| Categories | `2000` for Radarr, `5000` for Sonarr |

4. Click **Test** then **Save**

## Newznab

The Newznab endpoint provides Usenet NZB search results from StremThru's configured Newznab indexers.

**Endpoint:** `/v0/newznab/api`

**Authentication:** Required via `apikey` query parameter.

### API Key

The API key is your StremThru credentials (`username:password`) encoded as base64:

```sh
echo -n "username:password" | base64
```

For example, if your credentials are `user:pass`, the API key would be `dXNlcjpwYXNz`.

### Supported Features

| Feature | Details |
| --- | --- |
| Search Types | General search, TV search, Movie search |
| Categories | Movies (`2000`), TV (`5000`) |
| Search Parameters | `q` (query), `imdbid`, `season`, `ep`, `year` |
| NZB Download | Via `t=get&id={nzbId}` |
| Output Formats | XML (default), JSON (`o=json`) |

### Prowlarr Setup

1. Go to **Indexers > Add Indexer**
2. Select **Generic Newznab**
3. Fill in:

| Field | Value |
| --- | --- |
| Name | `StremThru Newznab` |
| URL | `http://<stremthru-host>/v0/newznab/api` |
| API Key | Your base64-encoded `username:password` |
| Categories | Select `Movies` and/or `TV` |

4. Click **Test** then **Save**

### Radarr / Sonarr Setup

1. Go to **Settings > Indexers > Add**
2. Select **Newznab**
3. Fill in:

| Field | Value |
| --- | --- |
| Name | `StremThru Newznab` |
| URL | `http://<stremthru-host>/v0/newznab/api` |
| API Key | Your base64-encoded `username:password` |
| Categories | `2000` for Radarr, `5000` for Sonarr |

4. Click **Test** then **Save**

::: tip
If you're using Prowlarr, you can add StremThru indexers there and sync them to Radarr/Sonarr automatically instead of configuring each app separately.
:::
