# Configuration

StremThru is configured entirely through environment variables. This makes it straightforward to configure in Docker, Docker Compose, or any other deployment method.

## Quick Reference

| Section                             | Key Variables                                                                     |
| ----------------------------------- | --------------------------------------------------------------------------------- |
| [Server](#server)                   | `STREMTHRU_BASE_URL`, `STREMTHRU_PORT`, `STREMTHRU_LOG_LEVEL`                     |
| [Authentication](#authentication)   | `STREMTHRU_PROXY_AUTH`, `STREMTHRU_AUTH_ADMIN`                                    |
| [Store](#store)                     | `STREMTHRU_STORE_AUTH`, `STREMTHRU_STORE_TUNNEL`, `STREMTHRU_STORE_CONTENT_PROXY` |
| [Content Proxy](#content-proxy)     | `STREMTHRU_CONTENT_PROXY_CONNECTION_LIMIT`                                        |
| [Database & Redis](#database-redis) | `STREMTHRU_DATABASE_URI`, `STREMTHRU_REDIS_URI`                                   |
| [Integrations](#integrations)       | TMDB, Trakt, AniList, MDBList, TVDB, GitHub                                       |

## Setting Environment Variables

**Docker:**

```sh
docker run -e STREMTHRU_PROXY_AUTH=user:pass muniftanjim/stremthru
```

**Docker Compose** (using `.env` file):

```sh
STREMTHRU_PROXY_AUTH=user:pass
STREMTHRU_STORE_AUTH=user:realdebrid:your-token
```

**From source:**

```sh
export STREMTHRU_PROXY_AUTH=user:pass
make run
```

## Sections

- **[Environment Variables](./environment-variables)** — Complete reference for all environment variables
- **[Features](./features)** — Feature flags system
- **[Database & Redis](./database)** — Database and caching configuration
