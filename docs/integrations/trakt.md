# Trakt Integration

[Trakt](https://trakt.tv/) integration enables watchlist and custom list support for Stremio catalogs.

## Used For

- Trakt watchlists and custom lists as Stremio catalogs via the [List addon](/stremio-addons/list)
- Dashboard - Vault

## Prerequisites

- `STREMTHRU_BASE_URL` must be set

## Setup

1. Create a Trakt account at [trakt.tv](https://trakt.tv/)
2. Create an [OAuth application](https://trakt.tv/oauth/applications)
3. Set the Redirect URI to `${STREMTHRU_BASE_URL}/auth/trakt.tv/callback`
4. Set CORS origins to `${STREMTHRU_BASE_URL}` in the Trakt OAuth app settings
5. Set the [environment variables](/configuration/integrations#trakt)
