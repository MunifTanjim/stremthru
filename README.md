[![License](https://img.shields.io/github/license/MunifTanjim/stremthru?style=for-the-badge)](https://github.com/MunifTanjim/stremthru/blob/master/LICENSE)

# StremThru

Companion for Stremio.

## Features

- HTTP Proxy
- Proxy Authorization
- [Byte Serving](https://en.wikipedia.org/wiki/Byte_serving)

### Store Integration

- [AllDebrid](https://alldebrid.com)
- [Debrid-Link](https://debrid-link.com)
- [Premiumize](https://www.premiumize.me)
- [RealDebrid](https://real-debrid.com) _(Planned)_

## Configuration

Configuration is done using environment variables.

**`STREMTHRU_HTTP_PROXY`**

HTTP Proxy URL.

**`STREMTHRU_HTTPS_PROXY`**

HTTPS Proxy URL.

**`STREMTHRU_PROXY_AUTH`**

Comma separated list of credentials, in the following formats:

- plain text credentials, e.g. `username:password`
- or base64 encoded credentials, e.g. `dXNlcm5hbWU6cGFzc3dvcmQ=`

These will be used for proxy authorization.

**`STREMTHRU_STORE_AUTH`**

Comma separated list of store credentials, in `username:store_name:api_key` format.

For proxy-authorized requests, these credentials will be used.

## Endpoints

### Authentication

**`Proxy-Authorization` Header**

Basic auth header, e.g. `Basic dXNlcm5hbWU6cGFzc3dvcmQ=`

`Proxy-Authorization` header is checked against `STREMTHRU_PROXY_AUTH` config.

### Store

This is a common interface for interacting with external stores.

If `X-StremThru-Store-Name` header is present, the value is used as store name. Otherwise,
the first store configured for the user using `STREMTHRU_STORE_AUTH` config is used.

**Authentication**

If `STREMTHRU_STORE_AUTH` is configured, then proxy-authorized requests will be
automatically authenticated for external stores.

For non-proxy-authorized requests, the following HTTP headers are used:

- `X-StremThru-Store-Authorization`
- `Authorization`

Values for these headers will be forwarded to the external store.

#### Get User

**`GET /v0/store/user`**

Get information about authenticated user.

**Response**:

```json
{
  "data": {
    "id": "string",
    "email": "string",
    "subscription_status": "UserSubscriptionStatus"
  }
}
```

#### Add Magnet

**`POST /v0/store/magnets`**

Add manget link for download.

**Request**:

```json
{
  "magnet": "string"
}
```

**Response**:

```json
{
  "data": {
    "id": "string",
    "hash": "string",
    "magnet": "string",
    "name": "string",
    "status": "MagnetStatus",
    "files": [
      {
        "index": "int",
        "link": "string",
        "name": "string",
        "path": "string",
        "size": "int"
      }
    ]
  }
}
```

If `.status` is `downloaded`, `.files` will have the list of files.

#### List Magnets

**`GET /v0/store/magnets`**

List mangets on user's account.

**Response**:

```json
{
  "data": {
    "items": [
      {
        "id": "string",
        "name": "string",
        "status": "MagnetStatus"
      }
    ]
  }
}
```

#### Get Magnet

**`GET /v0/store/magnets/{magnetId}`**

Get manget on user's account.

**Path Parameter**:

- `magnetId`: magnet id

**Response**:

```json
{
  "data": {
    "id": "string",
    "name": "string",
    "status": "MagnetStatus",
    "files": [
      {
        "index": "int",
        "link": "string",
        "name": "string",
        "path": "string",
        "size": "int"
      }
    ]
  }
}
```

#### Remove Magnet

**`DELETE /v0/store/magnets/{magnetId}`**

Remove manget from user's account.

**Path Parameter**:

- `magnetId`: magnet id

#### Check Magnet

**`GET /v0/store/magnets/check`**

Check manget links.

**Query Parameter**:

- `magnet`: comma seperated magnet links

**Response**:

```json
{
  "data": {
    "items": [
      {
        "hash": "string",
        "magnet": "string",
        "status": "MagnetStatus",
        "files": [
          {
            "index": "int",
            "name": "string",
            "size": "int"
          }
        ]
      }
    ]
  }
}
```

If `.status` is `cached`, `.files` will have the list of files.

#### Generate Link

`POST /v0/store/link/generate`

Generate direct link for a file link.

**Request**:

```json
{
  "link": "string"
}
```

**Response**:

```json
{
  "data": {
    "link": "string"
  }
}
```

The generated direct link should be valid for 24 hours.

### Enums

#### MagnetStatus

- `cached`
- `queued`
- `downloading`
- `processing`
- `downloaded`
- `uploading`
- `failed`
- `invalid`
- `unknown`

#### UserSubscriptionStatus

- `expired`
- `premium`
- `trial`

## Usage

**Source**

```sh
git clone https://github.com/MunifTanjim/stremthru
cd stremthru

# configure
export STREMTHRU_PROXY_AUTH=username:password

# run
make run

# build and run
make build
./stremthru
```

**Docker**

```sh
docker run --name stremthru -p 8080:8080 \
  -e STREMTHRU_PROXY_AUTH=username:password \
  muniftanjim/stremthru
```

**Docker Compose**

```sh
cp compose.example.yaml compose.yaml

docker compose up stremthru
```

## Related Resources

Cloudflare WARP:

- [github.com/cmj2002/warp-docker](https://github.com/cmj2002/warp-docker)

## License

Licensed under the MIT License. Check the [LICENSE](./LICENSE) file for details.
