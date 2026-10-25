# Meta API

The Meta API provides metadata for contents.

::: info Feature Required
This API requires the [`meta`](/configuration/features) feature to be enabled.
:::

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

### Get IMDb Title

**`GET /v0/meta/imdb/titles/{id}`**

Get details for a specific IMDb title. For TV shows the same response
includes the full season and episode hierarchy inline.

::: info Feature Required
This endpoint requires the [`meta`](/configuration/features) feature and an
executable `imdb` plugin binary in [`STREMTHRU_PLUGIN_DIR`](/configuration#stremthru_plugin_dir).
Data is served from a dedicated `imdb.db` SQLite file (sibling of the main
database file); the plugin is the sole writer.
:::

**Path Parameters:**

| Parameter | Description                 |
| --------- | --------------------------- |
| `id`      | IMDB ID (e.g., `tt0903747`) |

**Caching:** `Cache-Control: public, max-age=N, stale-while-revalidate=604800, stale-if-error=86400`,
where `N` is the number of seconds until the title is next due for a refresh,
with `ETag` / `Last-Modified` (`304` on `If-None-Match` match). Ratings, and
the episode list of a running series, are refreshed every 3 days for running
series, their episodes, and titles released in the last 90 days or not yet
released. Everything else is refreshed every 30 days.

A request never waits for a refresh when the title is stored. Stale data is
served at once with `Cache-Control: public, max-age=60` and a `Warning: 110`
header, and the refresh runs in the background. Only a title that is not
stored yet waits for the `imdb` plugin; if that fails the response is `502`,
and the plugin is not asked again for that title for 60s. A title IMDb does not
know returns `404`, cached for 60s; StremThru does not ask IMDb about it again
for 7 days. A stored title that IMDb has removed is deleted.

An episode is stored with its series as soon as the series is fetched. Until
the episode itself is first requested, it holds only what the series carries
(name, plot, image, release date, runtime, rating), and its response has
`"partial": true` and empty `genres`, `directors`, `cast` and `akas`. That
first request is served this partial data as stale (`max-age=60`) while the
full fetch runs in the background; later requests get the full episode. The
`uat` field is the time of the last full fetch; rating and episode-list
refreshes do not change it, but they do change `ETag` and `Last-Modified`.

**Response:**

```json
{
  "data": {
    "title": {
      "id": "tt0903747",
      "type": "tvSeries",
      "is_series": true,
      "name": "Breaking Bad",
      "year_start": 2008,
      "image": {
        "id": "rm3116305665",
        "url": "https://...",
        "width": 2000,
        "height": 3000
      },
      "genres": ["Crime"],
      "akas": [{ "text": "Breaking Bad", "country": "US" }]
    },
    "seasons": [
      {
        "number": 1,
        "episodes": [
          {
            "id": "tt0959621",
            "name": "Pilot",
            "image": {
              "id": "rm1098004225",
              "url": "https://...",
              "width": 3224,
              "height": 1814
            },
            "episode": 1
          }
        ]
      }
    ]
  }
}
```

Episodes are ordered by `episode` (`0` is a real number, used for specials),
then by release date. An episode IMDb gives no number has no `episode` field
and is listed after the numbered ones.

### Get IMDb Title Images

**`GET /v0/meta/imdb/titles/{id}/images`**

Get a page of a title's image gallery, 500 images per page. The first request
for a title waits until the first page is loaded from the `imdb` plugin, and
the remaining images load into the database in the background; later requests
are served from the database. A complete gallery is refreshed in the background
after 30 days, and the cached copy keeps being served meanwhile. A refresh adds
new images, updates existing ones, and removes images IMDb no longer lists; the
title's and its episodes' current images are always kept, and images after a
removed one move to earlier positions. Images are listed in the order they were
first stored, so the title's own image and its episodes' images, stored when
the title is fetched, come first. Galleries are loaded up to 25,000 images (50
plugin pages); a larger gallery stays truncated at that point.

**Path Parameters:**

| Parameter | Description                 |
| --------- | --------------------------- |
| `id`      | IMDB ID (e.g., `tt0903747`) |

**Query Parameters:**

| Parameter | Description                                       |
| --------- | ------------------------------------------------- |
| `page`    | Page number, from `1`. Default `1`. 500 per page. |

**Response:**

```json
{
  "data": {
    "images": [
      {
        "id": "rm3116305665",
        "kind": "poster",
        "url": "https://...",
        "width": 2000,
        "height": 3000,
        "caption": "..."
      },
      {
        "id": "rm1098004225",
        "ep_id": "tt0959621",
        "kind": "still_frame",
        "url": "https://...",
        "width": 3224,
        "height": 1814
      }
    ],
    "total": 120,
    "pages": 1,
    "complete": true
  }
}
```

`total` is the number of images and `pages` the number of pages; a page past
the end returns an empty `images` list. While the background load is still
running, `complete` is `false`, `total` counts the images stored so far, and
pages are cached for 60s. Retry until `complete` is `true`.

**Caching:** a complete gallery is served with
`Cache-Control: public, max-age=N, stale-if-error=86400`, where `N` is the
number of seconds until the gallery is due for a refresh, so all of its pages
expire at the same time. There is no `stale-while-revalidate`, so caches do not
keep serving old pages past that point. A gallery due for a refresh, or still
loading, is served with `Cache-Control: public, max-age=60`. Responses carry an
`ETag` (`304` on `If-None-Match` match) and, once the gallery is complete,
`Last-Modified`.

`ep_id` is set on an episode's image and names the episode. A title IMDb does
not know returns `404`, and is not asked about again for 7 days. If the first
page fails, the response is `502` only
when nothing is stored for the title yet; otherwise the stored images are
served with `complete: false`. A title fetched through the title endpoint
already has its own and its episodes' images stored. A failed load is not
retried for 60s.
