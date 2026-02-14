# Newz Addon

The Newz addon searches Newznab-compatible indexers and streams Usenet content directly in Stremio.

**Path:** `/stremio/newz`

## Overview

- Search Newznab-compatible indexers for content
- Stream Usenet content via NNTP
- NZB parsing and download management
- Connection pooling for concurrent streams

## Prerequisites

The Newz addon requires:

- `vault` and `stremio_newz` [feature flags](../configuration/features) enabled
- At least one NNTP server configured via the dashboard
- At least one Newznab indexer configured via the dashboard

## Configuration

| Environment Variable                         | Description                      |
| -------------------------------------------- | -------------------------------- |
| `STREMTHRU_STREMIO_NEWZ_INDEXER_MAX_TIMEOUT` | Max timeout for indexer requests |

See [Usenet Configuration](../configuration/usenet) for all Newz-related environment variables.

## Topics

This page will cover:

- Newznab indexer configuration
- NNTP server setup
- Streaming behavior and buffering
- NZB caching

::: info
Detailed documentation coming soon — contributions welcome.
:::
