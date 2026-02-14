# Database & Redis

StremThru supports SQLite and PostgreSQL for persistent storage, and Redis for caching.

## Database

### `STREMTHRU_DATABASE_URI`

URI for the database connection.

**Format:** `<scheme>://<user>:<pass>@<host>[:<port>][/<db>]`

**Supported schemes:**

| Scheme       | Database         |
| ------------ | ---------------- |
| `sqlite`     | SQLite (default) |
| `postgresql` | PostgreSQL       |

**Supported query parameters:**

| Parameter   | Description                   |
| ----------- | ----------------------------- |
| `max_conns` | Maximum number of connections |
| `min_conns` | Minimum number of connections |

### SQLite (Default)

SQLite is used by default with no configuration required. The database file is stored in the data directory.

### PostgreSQL

To use PostgreSQL, set the database URI:

```sh
STREMTHRU_DATABASE_URI=postgresql://stremthru:stremthru@localhost:5432/stremthru
```

With connection pool configuration:

```sh
STREMTHRU_DATABASE_URI=postgresql://stremthru:stremthru@localhost:5432/stremthru?max_conns=20&min_conns=5
```

A Docker Compose setup with PostgreSQL:

```yaml
services:
  stremthru:
    image: muniftanjim/stremthru
    ports:
      - 8080:8080
    environment:
      STREMTHRU_PROXY_AUTH: user:pass
      STREMTHRU_DATABASE_URI: postgresql://stremthru:stremthru@postgres:5432/stremthru
    depends_on:
      - postgres

  postgres:
    image: postgres:16.6-alpine
    environment:
      POSTGRES_DB: stremthru
      POSTGRES_USER: stremthru
      POSTGRES_PASSWORD: stremthru
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
```

## Redis

### `STREMTHRU_REDIS_URI`

URI for Redis connection.

**Format:** `redis://<user>:<pass>@<host>[:<port>][/<db>]`

If provided, Redis is used for caching instead of in-memory storage. This is recommended for production deployments.

**Example:**

```sh
STREMTHRU_REDIS_URI=redis://localhost:6379
```

A Docker Compose setup with Redis:

```yaml
services:
  stremthru:
    image: muniftanjim/stremthru
    ports:
      - 8080:8080
    environment:
      STREMTHRU_PROXY_AUTH: user:pass
      STREMTHRU_REDIS_URI: redis://redis:6379
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
```
