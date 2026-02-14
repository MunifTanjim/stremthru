# Introduction

StremThru is a companion service for [Stremio](https://www.stremio.com/). It provides an HTTP(S) proxy with authorization, debrid store integrations through a unified API, content proxying with byte serving, and a suite of Stremio addons.

## Store Integration

A _Store_ is an external debrid service that provides access to content. StremThru acts as a unified interface for interacting with these stores.

Supported stores:

- [AllDebrid](https://alldebrid.com)
- [Debrider](https://debrider.app)
- [Debrid-Link](https://debrid-link.com/id/4v8Uc)
- [EasyDebrid](https://easydebrid.com)
- [Offcloud](https://offcloud.com/?=ce30ae1f)
- [PikPak](https://mypikpak.com/drive/activity/invited?invitation-code=46013321)
- [Premiumize](https://www.premiumize.me/ref/634502061)
- [RealDebrid](http://real-debrid.com/?id=12448969)
- [TorBox](https://torbox.app/subscription?referral=fbe2c844-4b50-416a-9cd8-4e37925f5dfa)

## Stremio Addons

StremThru includes six built-in Stremio addons:

- **[Store](/stremio-addons/store)** — Browse and search your store catalog
- **[Wrap](/stremio-addons/wrap)** — Wrap other Stremio addons with StremThru
- **[Torz](/stremio-addons/torz)** — Torrent indexer integration
- **[Newz](/stremio-addons/newz)** — Stream Usenet content via Newznab indexers
- **[List](/stremio-addons/list)** — Generate catalogs from external lists (Trakt, TMDB, AniList, etc.)
- **[Sidekick](/stremio-addons/sidekick)** — Extra features for Stremio (addon management, library backup/restore)

## SDKs

Official SDKs are available for programmatic access:

- **[JavaScript](/sdk/javascript)** — `npm install stremthru`
- **[Python](/sdk/python)** — `pip install stremthru`
