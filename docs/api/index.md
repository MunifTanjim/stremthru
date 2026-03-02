# API

StremThru provides a REST API for various features.

## Authentication

API requests are authenticated using the `X-StremThru-Authorization` header with Basic auth.

```
X-StremThru-Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
```

The credentials are checked against the [`STREMTHRU_AUTH`](/configuration/#stremthru-auth) configuration.
