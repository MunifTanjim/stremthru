# Features

StremThru uses a feature flag system to enable or disable specific functionality.

## Configuration

Set the `STREMTHRU_FEATURE` environment variable with a comma-separated list of feature flags.

### Syntax

- `+feature` — Enable an opt-in feature
- `-feature` — Disable an opt-out feature
- `feature` — Enable only the specified features (disables all others not listed)

### Examples

Enable a specific opt-in feature:

```sh
STREMTHRU_FEATURE=+feature_a
```

Disable a specific opt-out feature:

```sh
STREMTHRU_FEATURE=-feature_b
```

Combine multiple flags:

```sh
STREMTHRU_FEATURE=+feature_a,-feature_b
```

::: tip
Use the `+` and `-` prefix syntax to selectively toggle features without affecting others. Without prefixes, only the explicitly listed features will be enabled.
:::

## Available Features

| Feature            | Description                 | Default  | Notes                     |
| ------------------ | --------------------------- | -------- | ------------------------- |
| `anime`            | Anime support               | Disabled |                           |
| `dmm_hashlist`     | DMM hashlist support        | Enabled  | Requires `torz`           |
| `imdb_title`       | IMDB title support          | Enabled  | Requires `newz` or `torz` |
| `meta`             | Meta API endpoints          | Enabled  |                           |
| `newz`             | Usenet support              | Enabled  |                           |
| `probe_media_info` | Probe Media Info            | Enabled  | Requires `torz`           |
| `stremio_list`     | Stremio List addon          | Enabled  |                           |
| `stremio_newz`     | Stremio Newz addon          | Enabled  | Requires `newz`           |
| `stremio_p2p`      | Stremio P2P support         | Disabled |                           |
| `stremio_sidekick` | Stremio Sidekick addon      | Enabled  |                           |
| `stremio_store`    | Stremio Store addon         | Enabled  | Requires `newz` or `torz` |
| `stremio_torz`     | Stremio Torz addon          | Enabled  | Requires `torz`           |
| `stremio_wrap`     | Stremio Wrap addon          | Enabled  |                           |
| `sync`             | Sync functionality          | Enabled  | Requires `vault`          |
| `torz`             | Torrent support             | Enabled  |                           |
| `vault`            | Vault for encrypted secrets | Enabled  |                           |

## Feature Interactions

### `newz` + `vault`

When both `newz` and `vault` are enabled, additional Usenet capabilities become available:

- **Indexer Management** — Add and manage Newznab indexers for newznab aggregation
- **Server Management** — Configure Usenet servers for downloading/streaming
- **SABnzbd Endpoint** — SABnzbd compatible API for tools like Sonarr/Radarr
- **WebDAV Endpoint** — Browse and stream Newz content
