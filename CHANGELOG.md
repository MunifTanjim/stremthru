# Changelog

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
