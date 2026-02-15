# Integrations

## AniList

#### `STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME`

Stale time for AniList list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_ANILIST_LIST_STALE_TIME=12h
```

## GitHub

#### `STREMTHRU_INTEGRATION_GITHUB_USER`

GitHub username.

**Example:**

```sh
STREMTHRU_INTEGRATION_GITHUB_USER=username
```

#### `STREMTHRU_INTEGRATION_GITHUB_TOKEN`

GitHub Personal Access Token.

**Example:**

```sh
STREMTHRU_INTEGRATION_GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```

## Letterboxd

No environment variables required. Letterboxd integration works with public profiles.

### `STREMTHRU_INTEGRATION_LETTERBOXD_LIST_STALE_TIME`

Stale time for Letterboxd list data.

- **Default:** `24h`
- **Minimum:** `6h`

**Example:**

```sh
STREMTHRU_INTEGRATION_LETTERBOXD_LIST_STALE_TIME=24h
```

## MDBList

### `STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME`

Stale time for MDBList list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME=12h
```

## TMDB

TMDB integration requires an [Access Token](https://www.themoviedb.org/settings/api).

### `STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN`

API Read Access Token for TMDB.

**Example:**

```sh
STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN=eyJhbGciOi...
```

### `STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME`

Stale time for TMDB list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME=12h
```

## Trakt

Trakt integration requires an [OAuth App](https://trakt.tv/oauth/applications).

The Redirect URI should point to the `/auth/trakt.tv/callback` endpoint of your [`STREMTHRU_BASE_URL`](#stremthru-base-url).

### `STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID`

Client ID for Trakt OAuth App.

**Example:**

```sh
STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID=your-client-id
```

### `STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET`

Client Secret for Trakt OAuth App.

**Example:**

```sh
STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET=your-client-secret
```

### `STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME`

Stale time for Trakt list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME=12h
```

## TVDB

TVDB integration requires an [API Key](https://www.thetvdb.com/dashboard/account/apikey).

#### `STREMTHRU_INTEGRATION_TVDB_API_KEY`

API Key for TVDB.

**Example:**

```sh
STREMTHRU_INTEGRATION_TVDB_API_KEY=your-api-key
```

#### `STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME`

Stale time for TVDB list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME=12h
```
