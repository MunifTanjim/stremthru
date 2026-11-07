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

## MatchLab

Optional external torrent parser and IMDb/AniDB matcher. It is not required for the default local parsing and matching behavior.

### `STREMTHRU_INTEGRATION_MATCHLAB_URL`

Reachable MatchLab HTTP(S) base URL. Leave unset to use the built-in Go-PTT/IMDb/AniDB matching. `MATCHLAB_URL` is also supported as an alias.

**Example:**

```sh
STREMTHRU_INTEGRATION_MATCHLAB_URL=http://matchlab:8765
```

### `STREMTHRU_INTEGRATION_MATCHLAB_TIMEOUT`

- **Default:** `10s`

### `STREMTHRU_INTEGRATION_MATCHLAB_CONCURRENCY`

- **Default:** `8` concurrent API calls
- **Allowed range:** 1–64

See the [MatchLab integration guide](/integrations/matchlab) for setup, feature flags, migrations, and how to return to native matching.

## MDBList

### `STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME`

Stale time for MDBList list data.

- **Default:** `12h`
- **Minimum:** `15m`

**Example:**

```sh
STREMTHRU_INTEGRATION_MDBLIST_LIST_STALE_TIME=12h
```

## Serializd

No environment variables required. Serializd integration requires [TMDB](#tmdb) to be enabled.

### `STREMTHRU_INTEGRATION_SERIALIZD_LIST_STALE_TIME`

Stale time for Serializd list data.

- **Default:** `24h`
- **Minimum:** `1h`

**Example:**

```sh
STREMTHRU_INTEGRATION_SERIALIZD_LIST_STALE_TIME=24h
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
