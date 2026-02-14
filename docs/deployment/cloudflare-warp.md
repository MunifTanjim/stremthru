# Cloudflare WARP

Using Cloudflare WARP as an HTTP proxy for tunneling store traffic.

## Overview

If your IP is blocked by a debrid service, you can use Cloudflare WARP as an HTTP proxy to tunnel traffic through Cloudflare's network.

## Docker Setup

Use [warp-docker](https://github.com/cmj2002/warp-docker) alongside StremThru:

```yaml
services:
  stremthru:
    image: muniftanjim/stremthru
    ports:
      - 8080:8080
    environment:
      STREMTHRU_PROXY_AUTH: user:pass
      STREMTHRU_HTTP_PROXY: socks5://warp:1080
      STREMTHRU_STORE_TUNNEL: "*:true"
    depends_on:
      - warp

  warp:
    image: cmj2002/warp-docker
    restart: unless-stopped
```

## Configuration

Set `STREMTHRU_HTTP_PROXY` to point to the WARP container and enable store tunneling:

```sh
STREMTHRU_HTTP_PROXY=socks5://warp:1080
STREMTHRU_STORE_TUNNEL=*:true
```

You can also selectively tunnel specific stores:

```sh
STREMTHRU_STORE_TUNNEL=realdebrid:true,alldebrid:true,*:false
```

## Related Resources

- [warp-docker](https://github.com/cmj2002/warp-docker) — Cloudflare WARP in Docker

::: info
Detailed documentation coming soon — contributions welcome.
:::
