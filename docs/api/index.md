# API

StremThru provides a REST API for interacting with stores, proxying content, and mapping content IDs.

## Authentication

API requests are authenticated using the `X-StremThru-Authorization` header with Basic auth.

```
X-StremThru-Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
```

The credentials are checked against the [`STREMTHRU_AUTH`](/configuration/environment-variables#stremthru-auth) configuration.

## Endpoints

| Section          | Description             |
| ---------------- | ----------------------- |
| [Proxy](./proxy) | Proxify URLs            |
| [Store](./store) | Unified Store interface |
| [Meta](./meta)   | Content Metadata        |
