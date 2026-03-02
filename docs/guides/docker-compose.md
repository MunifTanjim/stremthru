# Docker Compose

## Basic Setup

**Create directories for volume mounts**

```sh
mkdir -p ./docker/volumes/stremthru
mkdir -p ./docker/volumes/traefik/letsencrypt
```

**Clone Repository**

```sh
git clone https://github.com/MunifTanjim/stremthru.git
cd stremthru
cp .env.example .env.prod
```

**Config: `stremthru/.env.prod`**

```sh
STREMTHRU_AUTH=username:password
STREMTHRU_STORE_AUTH=username:torbox:torbox-api-key
```

**Compose file: `compose.yaml`**

```yaml
services:
  traefik:
    image: traefik:v3.2
    container_name: traefik
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entryPoints.web.address=:80"
      - "--entryPoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=<YOUR_EMAIL_ADDRESS_HERE>"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    ports:
      - 80:80
      - 443:443
    volumes:
      - ./docker/volumes/traefik/letsencrypt:/letsencrypt
      - /var/run/docker.sock:/var/run/docker.sock:ro

  stremthru:
    image: muniftanjim/stremthru:latest
    build:
      context: ./stremthru
      dockerfile: ./Dockerfile
    labels:
      - traefik.enable=true
      - traefik.http.routers.stremthru.rule=Host(`stremthru.<YOUR_DOMAIN_HERE>`)
      - traefik.http.routers.stremthru.entrypoints=websecure
      - traefik.http.routers.stremthru.tls.certresolver=letsencrypt
    ports:
      - 127.0.0.1:8080:8080
    env_file:
      - ./stremthru/.env.prod
    volumes:
      - ./docker/volumes/stremthru:/app/data
    restart: unless-stopped
```

**Run**

```sh
# build
docker compose build stremthru
# or pull
docker compose pull stremthru

# start
docker compose up -d traefik stremthru
```

## Gluetun

**Compose file: `compose.yaml`**

```yml
services:
  gluetun:
    image: qmcgaw/gluetun
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - ./docker/volumes/gluetun:/gluetun
    ports:
      - 127.0.0.1:1088:8888
      - 127.0.0.1:1080:1080
    environment:
      - VPN_SERVICE_PROVIDER=
      - OPENVPN_USER=
      - OPENVPN_PASSWORD=

  gost:
    image: gogost/gost
    restart: unless-stopped
    network_mode: service:gluetun
    command: "-L socks5h://:1080"
```

**Config: `stremthru/.env.prod`**

```sh
STREMTHRU_HTTP_PROXY=socks5://gluetun:1080
STREMTHRU_TUNNEL=*:false
STREMTHRU_STORE_TUNNEL=*:false
```

**Run**

```sh
docker compose up -d gost gluetun

docker compose up -d --force-recreate stremthru
```
