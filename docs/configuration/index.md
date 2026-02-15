# Configuration

StremThru is configured using environment variables.

## Quick Reference

| Section                             | Key Variables                                                                     |
| ----------------------------------- | --------------------------------------------------------------------------------- |
| [Server](#server)                   | `STREMTHRU_BASE_URL`, `STREMTHRU_PORT`, `STREMTHRU_LOG_LEVEL`                     |
| [Authentication](#authentication)   | `STREMTHRU_AUTH`, `STREMTHRU_AUTH_ADMIN`                                          |
| [Store](#store)                     | `STREMTHRU_STORE_AUTH`, `STREMTHRU_STORE_TUNNEL`, `STREMTHRU_STORE_CONTENT_PROXY` |
| [Content Proxy](#content-proxy)     | `STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT`                                        |
| [Database & Redis](#database-redis) | `STREMTHRU_DATABASE_URI`, `STREMTHRU_REDIS_URI`                                   |
| [Integrations](#integrations)       | TMDB, Trakt, AniList, MDBList, TVDB, GitHub                                       |

## Setting Environment Variables

**Docker:**

```sh
docker run -e STREMTHRU_AUTH=user:pass muniftanjim/stremthru
```

**Docker Compose** (using `.env` file):

```sh
STREMTHRU_AUTH=user:pass
```

**From source:**

```sh
export STREMTHRU_AUTH=user:pass
make run
```

## Sections

- **[Environment Variables](./environment-variables)** — Complete reference for all environment variables
- **[Features](./features)** — Feature flags system
- **[Database & Cache](./database-and-cache)** — Database and caching configuration
