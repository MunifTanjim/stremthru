# Installation

## Docker (Recommended)

Pull and run the Docker image:

```sh
docker run --name stremthru -p 8080:8080 \
  -e STREMTHRU_PROXY_AUTH=username:password \
  muniftanjim/stremthru
```

This starts StremThru on port `8080` with basic proxy authorization.

## Docker Compose

1. Create a `compose.yaml` file:

```yaml
services:
  stremthru:
    container_name: stremthru
    image: muniftanjim/stremthru
    ports:
      - 8080:8080
    env_file:
      - .env
    restart: unless-stopped
    volumes:
      - ./data:/app/data
  redis:
    image: redis:7-alpine
    ports:
      - 8089:6379
  postgres:
    image: postgres:16.6-alpine
    ports:
      - 8088:5432
    environment:
      POSTGRES_DB: stremthru
      POSTGRES_USER: stremthru
      POSTGRES_PASSWORD: stremthru
    restart: always
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
```

2. Create a `.env` file with your configuration:

```sh
STREMTHRU_PROXY_AUTH=username:password
```

3. Start the services:

```sh
docker compose up -d stremthru
```

::: tip
Redis and PostgreSQL are optional. StremThru uses in-memory caching and SQLite by default. See [Database & Redis](/configuration/database) for details.
:::

## From Source

### Prerequisites

- [Go](https://go.dev/) 1.22 or later
- [Make](https://www.gnu.org/software/make/)

### Build and Run

```sh
git clone https://github.com/MunifTanjim/stremthru
cd stremthru

# configure
export STREMTHRU_PROXY_AUTH=username:password

# run directly
make run

# or build and run the binary
make build
./stremthru
```
