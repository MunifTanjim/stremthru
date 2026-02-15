# Installation

## Docker

Pull and run the Docker image:

```sh
docker run --rm -p 8080:8080 \
  -e STREMTHRU_AUTH=username:password \
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
```

2. Create a `.env` file with your configuration:

```sh
STREMTHRU_AUTH=username:password
```

3. Start the services:

```sh
docker compose up -d stremthru
```

## From Source

### Prerequisites

- [Go](https://go.dev/) 1.25 or later
- [Make](https://www.gnu.org/software/make/)

### Build and Run

```sh
git clone https://github.com/MunifTanjim/stremthru
cd stremthru
```

Create a `.env` file with your configuration:

```sh
STREMTHRU_AUTH=username:password
```

```sh
source .env
make run
```
