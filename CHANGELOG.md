# Changelog

## [0.14.0](https://github.com/MunifTanjim/stremthru/compare/0.13.1...0.14.0) (2024-12-20)


### Features

* **stremio/sidekick:** initial addon implementation ([1c88b4f](https://github.com/MunifTanjim/stremthru/commit/1c88b4fbf3b6cfadbb8d12a0963eda807c404d21))
* **stremio/sidekick:** support disabling addon ([e0ea92d](https://github.com/MunifTanjim/stremthru/commit/e0ea92d63cb74c4d475504d2d8ea3f662bd5b205))


### Bug Fixes

* **stremio:** use string body for addon client error ([87c764b](https://github.com/MunifTanjim/stremthru/commit/87c764bf847e6b60591058681c9e00c78a0d5f96))

## [0.13.1](https://github.com/MunifTanjim/stremthru/compare/0.13.0...0.13.1) (2024-12-19)


### Bug Fixes

* **stremio:** deduplicate id in configure page ([b56f934](https://github.com/MunifTanjim/stremthru/commit/b56f9344a00478a1ec61820dd702b68aadee34aa))
* **stremio:** stop loading indicator on failed request ([e370417](https://github.com/MunifTanjim/stremthru/commit/e3704178dfa013f91bb29ce02e62e793deafbcb3))

## [0.13.0](https://github.com/MunifTanjim/stremthru/compare/0.12.0...0.13.0) (2024-12-19)


### Features

* **stremio:** add addon - wrap ([bfc9e1c](https://github.com/MunifTanjim/stremthru/commit/bfc9e1c9ce1344568d6aae10edd097088780caeb))

## [0.12.0](https://github.com/MunifTanjim/stremthru/compare/0.11.1...0.12.0) (2024-12-19)


### Features

* add config for landing page ([e383a0d](https://github.com/MunifTanjim/stremthru/commit/e383a0dc13233b5604571e980b443f1401931ea5))
* **stremio:** improve landing page for store ([7a0638c](https://github.com/MunifTanjim/stremthru/commit/7a0638c48faed2f42757727764cbe578bd9a81ed))
* **stremio:** mention store name in manifest ([559467c](https://github.com/MunifTanjim/stremthru/commit/559467c76a249fb94abd9f94a3bbf8102fa2cddb))


### Bug Fixes

* **endpoint:** match landing page route exactly ([2d76b0e](https://github.com/MunifTanjim/stremthru/commit/2d76b0ed7433cce78608cb62cd93dd4f969225d7))

## [0.11.1](https://github.com/MunifTanjim/stremthru/compare/0.11.0...0.11.1) (2024-12-18)


### Bug Fixes

* **stremio:** malformed manifest for store ([886d2a3](https://github.com/MunifTanjim/stremthru/commit/886d2a342a40c0dd809cebe41d7688db73928f44))
* **stremio:** properly handle user data error for store ([ff6842d](https://github.com/MunifTanjim/stremthru/commit/ff6842d74a51db2aa55ecffac2d79c51f95610cc))

## [0.11.0](https://github.com/MunifTanjim/stremthru/compare/0.10.0...0.11.0) (2024-12-17)


### Features

* add landing page ([f499cbf](https://github.com/MunifTanjim/stremthru/commit/f499cbf722da4c84057e87e326272b316245191d))
* add version in health debug ([e9c1980](https://github.com/MunifTanjim/stremthru/commit/e9c1980ac17786be8015b67c164ec26345a17dbd))
* **stremio:** add addon - store ([39dca81](https://github.com/MunifTanjim/stremthru/commit/39dca81be9eed485877fa8b3c85c9aca1930181c))
* **stremio:** add config to toggle addons ([a6c06f8](https://github.com/MunifTanjim/stremthru/commit/a6c06f811a0b6e45a3a5b7faae47503b3907e926))

## [0.10.0](https://github.com/MunifTanjim/stremthru/compare/0.9.0...0.10.0) (2024-12-15)


### Features

* **store:** add added_at field for magnet list/get ([d772158](https://github.com/MunifTanjim/stremthru/commit/d772158053e4a519d43dc607125b4244afcc1ae0))

## [0.9.0](https://github.com/MunifTanjim/stremthru/compare/0.8.0...0.9.0) (2024-12-10)


### Features

* **db:** add 'heavy' tag for auto schema migration ([0d6b28f](https://github.com/MunifTanjim/stremthru/commit/0d6b28fd4cf98cb27e0fc1940a560e06c1f31b59))


### Bug Fixes

* **peer_token:** fix schema file for postgresql ([aad8e7b](https://github.com/MunifTanjim/stremthru/commit/aad8e7b37d504feaa749353d1937420b8607393b))

## [0.8.0](https://github.com/MunifTanjim/stremthru/compare/0.7.0...0.8.0) (2024-12-09)


### Features

* **db:** switch from libsql to sqlite3 ([dc9e0c8](https://github.com/MunifTanjim/stremthru/commit/dc9e0c86212ea1c050348415c9f04c1feab10c1d))
* **magnet_cache:** get rid of unnecessary transaction ([fb7b244](https://github.com/MunifTanjim/stremthru/commit/fb7b2441c1150d0499114183a891ffeabe1472c8))

## [0.7.0](https://github.com/MunifTanjim/stremthru/compare/0.6.0...0.7.0) (2024-12-07)


### Features

* **db:** handle connection and transaction better ([3c60920](https://github.com/MunifTanjim/stremthru/commit/3c609203f1953ac545549d562729c9c74c3688d6))
* **db:** log magnet_cache insert failure better ([0624b1a](https://github.com/MunifTanjim/stremthru/commit/0624b1a8460f83e48cd5c94bba79d112438159ee))
* **magnet_cache:** extract and fix db stuffs ([8fbeafb](https://github.com/MunifTanjim/stremthru/commit/8fbeafbb7c9a3099f203e1837d84adfbad9b0e2c))
* **store/realdebrid:** update error codes ([8a49941](https://github.com/MunifTanjim/stremthru/commit/8a499413edb220cfd32a124a2138797f1e7f28ad))

## [0.6.0](https://github.com/MunifTanjim/stremthru/compare/0.5.0...0.6.0) (2024-12-06)


### Features

* add support for uptream node ([f704542](https://github.com/MunifTanjim/stremthru/commit/f70454298382413dfe7b04d92799eaf376173cd9))
* **db:** add support for postgresql ([df3473c](https://github.com/MunifTanjim/stremthru/commit/df3473c461f9aae8a95d56a715befbfbd6461a6f))
* **db:** initial setup ([7371667](https://github.com/MunifTanjim/stremthru/commit/73716677b9a9301763a61e5f584da13f489e65f9))
* **db:** use wal mode for sqlite ([39f2b18](https://github.com/MunifTanjim/stremthru/commit/39f2b18e8c628c01a00d9c933d1b9ed16f8cdc5f))
* extract request stuffs ([031dd77](https://github.com/MunifTanjim/stremthru/commit/031dd77a09db0370e4344d682f9cd53e4c86a4d3))
* **peer:** introduce concept of peer ([18ced66](https://github.com/MunifTanjim/stremthru/commit/18ced66a55e6a72630767e8231eeeb783011212d))
* store magnet cache info in db ([7b32556](https://github.com/MunifTanjim/stremthru/commit/7b325560ac69f1a0e126df020feebccca8ca74c4))
* **store:** integrate upstream for check and track magnet cache ([f6bf4d7](https://github.com/MunifTanjim/stremthru/commit/f6bf4d7900c58d61b238fb075c58725f5bd158bc))
* **store:** update error code for invalid store name ([281041e](https://github.com/MunifTanjim/stremthru/commit/281041e978e2a6a66de37cc12b182df0b5ef9b4d))
* update header for buddy token ([45afd6d](https://github.com/MunifTanjim/stremthru/commit/45afd6dd4cf5df37fa2f4c248acf1ff0ffd598f6))


### Bug Fixes

* **config:** handle env var with empty value ([7a1abcf](https://github.com/MunifTanjim/stremthru/commit/7a1abcfee81d02fcf1c3128f4f2bb7733053f90e))

## [0.5.0](https://github.com/MunifTanjim/stremthru/compare/0.4.0...0.5.0) (2024-12-04)


### Features

* **store:** use local storage when buddy not available ([43fe0a3](https://github.com/MunifTanjim/stremthru/commit/43fe0a308af600fd44df7a680ecafd5163c466f0))


### Bug Fixes

* **store:** pass client ip only for non-proxy-authorized requests ([7f89bc3](https://github.com/MunifTanjim/stremthru/commit/7f89bc3dd889100529e71b29b675f395c4fa7668))

## [0.4.0](https://github.com/MunifTanjim/stremthru/compare/0.3.0...0.4.0) (2024-12-03)


### Features

* **buddy:** add auth token config ([f830911](https://github.com/MunifTanjim/stremthru/commit/f8309119fb8a469662027961c413cb10a00bab1e))
* **buddy:** add local cache ([73b869e](https://github.com/MunifTanjim/stremthru/commit/73b869ee810fd3547088794a4b500f55948ad755))
* **core:** rename magnet invalid error ([0b6be1f](https://github.com/MunifTanjim/stremthru/commit/0b6be1f7b0264c878da5ce0a7464eda999f460f1))
* **store/realdebrid:** support passing client ip ([1265f1b](https://github.com/MunifTanjim/stremthru/commit/1265f1bc8d1c897bfd27239d591f3793435eb751))
* **store:** add support for buddy ([5243279](https://github.com/MunifTanjim/stremthru/commit/5243279eac80290843c2243223b5d3c9213afcb3))
* **store:** integrate buddy with all stores ([cd4998d](https://github.com/MunifTanjim/stremthru/commit/cd4998d1543d72f17cdc14fca83082ea8216db0d))
* support redis cache ([3bfbe70](https://github.com/MunifTanjim/stremthru/commit/3bfbe70a7dfe16f12cb6689d5772e63eece4da8f))


### Bug Fixes

* handle upstream service unavailable ([80d69ab](https://github.com/MunifTanjim/stremthru/commit/80d69abc7266234c205eff726db300a0070e467d))
* **store:** nil-error for buddy ([1d597ab](https://github.com/MunifTanjim/stremthru/commit/1d597ab7d4e2f440fe966e09b824c00c89bfd613))

## [0.3.0](https://github.com/MunifTanjim/stremthru/compare/0.2.0...0.3.0) (2024-11-24)


### Features

* improve errors ([dcbe689](https://github.com/MunifTanjim/stremthru/commit/dcbe689d057a0b1714d4fb68b245f3dd8d3a9fa7))
* **store/premiumize:** improve types ([6d92bd9](https://github.com/MunifTanjim/stremthru/commit/6d92bd9c3d77381530f2ff02c6d63846a2586dfe))

## [0.2.0](https://github.com/MunifTanjim/stremthru/compare/0.1.0...0.2.0) (2024-11-23)


### Features

* **store:** support pagination for list magnets ([0869539](https://github.com/MunifTanjim/stremthru/commit/0869539a3ac4ac2e447af87658efc0612f05ec30))


### Bug Fixes

* **store/torbox:** handle 404 for list torrents ([9730b8a](https://github.com/MunifTanjim/stremthru/commit/9730b8a52bcba39a6c42180c2f2ce8f900b83441))
* **store/torbox:** handle extra item for list torrents ([a43167d](https://github.com/MunifTanjim/stremthru/commit/a43167d10b12046e452b6a879d92318839e16b4b))

## 0.1.0 (2024-11-21)


### Features

* add .env.example ([9f564f7](https://github.com/MunifTanjim/stremthru/commit/9f564f760f2878cccb7e281f15dfa55adea1a667))
* add Dockerfile ([ab4d4db](https://github.com/MunifTanjim/stremthru/commit/ab4d4db4a0fbe1cbd302430e965370243752808e))
* add health/__debug__ endpoint ([94c4268](https://github.com/MunifTanjim/stremthru/commit/94c4268b9d986b624a04647ff785a962e4d2da05))
* **core:** improve cache initialization ([8c31e5b](https://github.com/MunifTanjim/stremthru/commit/8c31e5bc5123fa4ecd9e74c68c49423abbde50e6))
* initial implementation ([054a20f](https://github.com/MunifTanjim/stremthru/commit/054a20f1ab84725f1221c9047c767db4d4db938a))
* pass X-StremThru-Store-Name request header to response ([010f626](https://github.com/MunifTanjim/stremthru/commit/010f62680aff744f41dddcb45c10325e5b7c41ac))
* **store/premiumize:** improve magnet status detection ([81f1f2a](https://github.com/MunifTanjim/stremthru/commit/81f1f2a07e220f605f1471d355b5d98a7ec41f14))
* **store/realdebrid:** improve add magnet ([0e9a3ca](https://github.com/MunifTanjim/stremthru/commit/0e9a3cab32d439e789fb99b54218530423570947))
* **store:** add .hash for GetMagnet and ListMagnets ([aa93af5](https://github.com/MunifTanjim/stremthru/commit/aa93af5f8fed38d2dd6ff8118e3d49893127cff6))
* **store:** add cache for torbox store ([dc2f26a](https://github.com/MunifTanjim/stremthru/commit/dc2f26a8e8f3abec69504e8ea8a19b688688a0eb))
* **store:** add enum for UserSubscriptionStatus ([5e9a0c9](https://github.com/MunifTanjim/stremthru/commit/5e9a0c956b83695b05b0fb16f2b7bb58c483602b))
* **store:** expose lowercase .hash for magnet ([709ec45](https://github.com/MunifTanjim/stremthru/commit/709ec45a85892c2c8b98e2a9d8fce261f49ee1f6))
* **store:** initial alldebrid integration ([8e80efe](https://github.com/MunifTanjim/stremthru/commit/8e80efe5eb14b16277a4dce35b8de42ea3d965b6))
* **store:** initial debridlink integration ([c31f836](https://github.com/MunifTanjim/stremthru/commit/c31f836f1bd7e0bba0ae6b62d6d64a7e90fb3a9d))
* **store:** initial premiumize integration ([d73fa42](https://github.com/MunifTanjim/stremthru/commit/d73fa426f233d8d76875545c7162cb56bbd04f6c))
* **store:** initial realdebrid integration ([440cab2](https://github.com/MunifTanjim/stremthru/commit/440cab237e5264df16b8655f2131f294a48cf5c0))
* **store:** initial torbox integration ([23d5cfd](https://github.com/MunifTanjim/stremthru/commit/23d5cfdddb6dc06b58c56e2fff3ca4731914b60d))
* **store:** support json payload in request body ([aa73c7e](https://github.com/MunifTanjim/stremthru/commit/aa73c7e58700cbeb26415c8b5de76ffd432ecd03))
* support fallback store auth config ([8a6cbd8](https://github.com/MunifTanjim/stremthru/commit/8a6cbd8a89844d6b8f77b92f38e8668c1b644cce))
* support proxy auth ([9659c05](https://github.com/MunifTanjim/stremthru/commit/9659c05c1629d2325664ff92500b1197c65ca426))


### Bug Fixes

* **core:** handle empty body for 204 status ([21417f1](https://github.com/MunifTanjim/stremthru/commit/21417f11107ea7f3a412e2829d2aa0eef49eada6))
* **core:** remove empty dn query in ParseMagnet ([2aa59ff](https://github.com/MunifTanjim/stremthru/commit/2aa59ffbc2e381d06487f35f78768ebb237e9080))
* **endpoint:** add missing early return ([274efee](https://github.com/MunifTanjim/stremthru/commit/274efeea4fc338e016103a824c7c566e2d2d5bab))
* **endpoint:** do not send null for empty array ([93edc4d](https://github.com/MunifTanjim/stremthru/commit/93edc4d99ea1d004f4d4aeb958385d53930f360c))
* **endpoint:** do not send null for empty array ([a2aba63](https://github.com/MunifTanjim/stremthru/commit/a2aba633506284cb7e534fb4c057ea6536dcebc0))
* **endpoint:** expose delete magnet ([8171a29](https://github.com/MunifTanjim/stremthru/commit/8171a29effe709a3d366a71c5896d98ecdafeb9d))
* **store/alldebrid:** ensure non-null .files for GetMagnet ([784ee1f](https://github.com/MunifTanjim/stremthru/commit/784ee1fa5eb637b782c296a7c6e01e69224e815f))
* **store/debridlink:** handle not found for GetMagnet ([5cb1fb7](https://github.com/MunifTanjim/stremthru/commit/5cb1fb7f20876e897a0794b904327dfe131fd831))
* **store/debridlink:** pass query params for ListSeedboxTorrents ([8a10e26](https://github.com/MunifTanjim/stremthru/commit/8a10e26519108b899ad65072af2b01687a0a21d9))
* **store/premiumize:** handle not found for GetMagnet ([77dc312](https://github.com/MunifTanjim/stremthru/commit/77dc31288f3bf084af2197dde5ed61506eca6e2b))
* **store/premiumize:** prefix file path with / ([a3eb584](https://github.com/MunifTanjim/stremthru/commit/a3eb5844b78612d0f6beac25c8bf508924627545))
* **store/realdebrid:** deal with inconsistent type in response ([5f22bfb](https://github.com/MunifTanjim/stremthru/commit/5f22bfb9d351619de0388a7c69bf58d4f3869b1a))
* **store/torbox:** error handling for get magnet ([e28e401](https://github.com/MunifTanjim/stremthru/commit/e28e401263ea0113dc5787ffad228eb640c0d82a))
* **store:** store name in error ([fee51a2](https://github.com/MunifTanjim/stremthru/commit/fee51a26dcab67cd3cfd0ca5791906c2de3c3167))


### Performance Improvements

* **store:** cache access link token verification ([0db97d2](https://github.com/MunifTanjim/stremthru/commit/0db97d2f8c235ce1f57ffa68e4db509bf645e0ef))


### Continuous Integration

* add release job ([d6bdd2e](https://github.com/MunifTanjim/stremthru/commit/d6bdd2ea57153ae03483cb8bc6639ea04bd913cc))
