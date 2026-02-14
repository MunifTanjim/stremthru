# Quick Start

This guide walks you through getting StremThru running and connecting your first store.

## 1. Start StremThru

The fastest way to start is with Docker:

```sh
docker run --name stremthru -p 8080:8080 \
  -e STREMTHRU_PROXY_AUTH=user:pass \
  muniftanjim/stremthru
```

## 2. Verify It's Running

```sh
curl http://127.0.0.1:8080/v0/health
```

You should get a successful response.

## 3. Connect a Store

To connect a debrid store, set the `STREMTHRU_STORE_AUTH` environment variable. The format is `username:store_name:store_token`.

For example, to connect a RealDebrid account:

```sh
docker run --name stremthru -p 8080:8080 \
  -e STREMTHRU_PROXY_AUTH=user:pass \
  -e STREMTHRU_STORE_AUTH=user:realdebrid:your-api-token \
  muniftanjim/stremthru
```

See the [Store Auth](/configuration/environment-variables#stremthru-store-auth) documentation for the full list of stores and token formats.

## 4. Test the Store Connection

Verify the store is connected by checking the user endpoint:

```sh
curl -H "X-StremThru-Authorization: Basic $(echo -n user:pass | base64)" \
  http://127.0.0.1:8080/v0/store/user
```

## 5. Install a Stremio Addon

Open your browser and navigate to `http://127.0.0.1:8080/stremio/store` to configure and install the Store addon in Stremio.

## Next Steps

- [Configuration overview](/configuration/) — Learn about all configuration options
- [Environment variables](/configuration/environment-variables) — Full reference for all env vars
- [Stremio Addons](/stremio-addons/) — Explore the five built-in addons
- [Deployment](/deployment/) — Set up for production use
