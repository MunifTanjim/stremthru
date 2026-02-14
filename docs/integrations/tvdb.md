# TVDB Integration

[TVDB](https://www.thetvdb.com/) integration enables TV show list support and content metadata.

## What It Enables

- TVDB lists as Stremio catalogs via the [List addon](/stremio-addons/list)
- Content metadata and ID mapping

## Setup

1. Create a TVDB account at [thetvdb.com](https://www.thetvdb.com/)
2. Get an API Key from [account settings](https://www.thetvdb.com/dashboard/account/apikey)
3. Set the environment variable:

```sh
STREMTHRU_INTEGRATION_TVDB_API_KEY=your-api-key
```

## Environment Variables

| Variable                                     | Description                            |
| -------------------------------------------- | -------------------------------------- |
| `STREMTHRU_INTEGRATION_TVDB_API_KEY`         | API Key for TVDB                       |
| `STREMTHRU_INTEGRATION_TVDB_LIST_STALE_TIME` | Stale time for list data (e.g., `12h`) |

::: info
Detailed documentation coming soon — contributions welcome.
:::
