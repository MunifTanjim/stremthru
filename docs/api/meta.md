# Meta API

The Meta API provides content ID mapping between different services.

## Endpoints

### Get ID Map

**`GET /v0/meta/id-map/{idType}/{id}`**

Get ID mapping for a given content ID.

**Path Parameters:**

| Parameter | Description                 |
| --------- | --------------------------- |
| `idType`  | `movie` or `show`           |
| `id`      | IMDB ID (e.g., `tt0110912`) |

**Response:**

```json
{
  "type": "string",
  "imdb": "string",
  "tmdb": "string",
  "tvdb": "string",
  "trakt": "string"
}
```

**Example:**

```sh
curl -H "X-StremThru-Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=" \
  http://127.0.0.1:8080/v0/meta/id-map/movie/tt0110912
```

::: info
Detailed documentation coming soon — contributions welcome.
:::
