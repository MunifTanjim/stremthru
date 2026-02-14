# Trakt Integration

[Trakt](https://trakt.tv/) integration enables watchlist and custom list support for Stremio catalogs.

## What It Enables

- Trakt watchlists and custom lists as Stremio catalogs via the [List addon](/stremio-addons/list)

## Setup

1. Create a Trakt account at [trakt.tv](https://trakt.tv/)
2. Create an [OAuth application](https://trakt.tv/oauth/applications)
3. Set the Redirect URI to `{STREMTHRU_BASE_URL}/auth/trakt.tv/callback`
4. Set the environment variables:

```sh
STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID=your-client-id
STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET=your-client-secret
```

## Environment Variables

| Variable                                      | Description                            |
| --------------------------------------------- | -------------------------------------- |
| `STREMTHRU_INTEGRATION_TRAKT_CLIENT_ID`       | Client ID for Trakt OAuth App          |
| `STREMTHRU_INTEGRATION_TRAKT_CLIENT_SECRET`   | Client Secret for Trakt OAuth App      |
| `STREMTHRU_INTEGRATION_TRAKT_LIST_STALE_TIME` | Stale time for list data (e.g., `12h`) |

::: info
Detailed documentation coming soon — contributions welcome.
:::
