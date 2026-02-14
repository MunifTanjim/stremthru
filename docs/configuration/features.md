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
STREMTHRU_FEATURE=+some_feature
```

Disable a specific opt-out feature:

```sh
STREMTHRU_FEATURE=-some_feature
```

Combine multiple flags:

```sh
STREMTHRU_FEATURE=+feature_a,-feature_b
```

::: tip
Use the `+` and `-` prefix syntax to selectively toggle features without affecting others. Without prefixes, only the explicitly listed features will be enabled.
:::

## Available Features

| Feature            | Description                 | Notes            |
| ------------------ | --------------------------- | ---------------- |
| `anime`            | Anime support               |                  |
| `dmm_hashlist`     | DMM hashlist support        |                  |
| `imdb_title`       | IMDB title support          |                  |
| `stremio_list`     | Stremio List addon          |                  |
| `stremio_newz`     | Stremio Newz addon (Usenet) | Requires `vault` |
| `stremio_p2p`      | Stremio P2P support         |                  |
| `stremio_sidekick` | Stremio Sidekick addon      |                  |
| `stremio_store`    | Stremio Store addon         |                  |
| `stremio_torz`     | Stremio Torz addon          |                  |
| `stremio_wrap`     | Stremio Wrap addon          |                  |
| `vault`            | Vault for encrypted secrets |                  |
