# Meta API

The Meta API provides metadata for contents.

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
