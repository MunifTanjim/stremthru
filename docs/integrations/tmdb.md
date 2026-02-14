# TMDB Integration

[The Movie Database (TMDB)](https://www.themoviedb.org/) integration enables movie and TV show list support and content metadata.

## What It Enables

- TMDB lists as Stremio catalogs via the [List addon](/stremio-addons/list)
- Content metadata and ID mapping

## Setup

1. Create a TMDB account at [themoviedb.org](https://www.themoviedb.org/)
2. Get an API Read Access Token from [API settings](https://www.themoviedb.org/settings/api)
3. Set the environment variable:

```sh
STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN=your-access-token
```

## Environment Variables

| Variable                                     | Description                            |
| -------------------------------------------- | -------------------------------------- |
| `STREMTHRU_INTEGRATION_TMDB_ACCESS_TOKEN`    | API Read Access Token                  |
| `STREMTHRU_INTEGRATION_TMDB_LIST_STALE_TIME` | Stale time for list data (e.g., `12h`) |

::: info
Detailed documentation coming soon — contributions welcome.
:::
