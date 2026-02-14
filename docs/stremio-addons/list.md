# List Addon

The List addon generates Stremio catalogs from external list providers like Trakt, TMDB, AniList, Letterboxd, MDBList, and TVDB.

**Path:** `/stremio/list`

## Overview

- Generate Stremio catalogs from external lists
- Multiple list provider support
- Automatic list syncing

## Supported Providers

| Provider                               | Details                  |
| -------------------------------------- | ------------------------ |
| [AniList](/integrations/anilist)       | Anime lists              |
| [Letterboxd](/integrations/letterboxd) | Movie lists              |
| [MDBList](/integrations/mdblist)       | Custom lists             |
| [TMDB](/integrations/tmdb)             | Movie and TV lists       |
| [Trakt](/integrations/trakt)           | Watchlists, custom lists |
| [TVDB](/integrations/tvdb)             | TV show lists            |

## Configuration

| Environment Variable                           | Description                  |
| ---------------------------------------------- | ---------------------------- |
| `STREMTHRU_STREMIO_LIST_PUBLIC_MAX_LIST_COUNT` | Max lists on public instance |

## Topics

This page will cover:

- Adding list providers
- Catalog generation and syncing
- Public instance limits
- Provider-specific setup

::: info
Detailed documentation coming soon — contributions welcome.
:::
