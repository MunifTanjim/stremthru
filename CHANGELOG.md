# Changelog

## [0.45.0](https://github.com/MunifTanjim/stremthru/compare/0.44.0...0.45.0) (2025-02-06)


### Features

* **kv:** add type field ([fc4ae55](https://github.com/MunifTanjim/stremthru/commit/fc4ae55c34bbec48c441c837026551464e96f7fc))
* **stremio/wrap:** add support for transformer ([908c1a7](https://github.com/MunifTanjim/stremthru/commit/908c1a79faa209b8a5d4c4e063ab98567497c0c1))

## [0.44.0](https://github.com/MunifTanjim/stremthru/compare/0.43.1...0.44.0) (2025-02-06)


### Features

* **stremio/sidekick:** allow id/name change when reloading addon ([a6643d9](https://github.com/MunifTanjim/stremthru/commit/a6643d97aec97a5c057f87f792939ee5408df771))
* **stremio/wrap:** disable multi addons for public instance ([f153432](https://github.com/MunifTanjim/stremthru/commit/f1534324cccb3f03b5582588bfbf2cc0ec046a9f))
* **stremio/wrap:** support multiple upstream addons ([7777d27](https://github.com/MunifTanjim/stremthru/commit/7777d2740b3bc9ace93db4d11e0d1af97f1ab623))
* **stremio:** update types ([7210a3c](https://github.com/MunifTanjim/stremthru/commit/7210a3c0f39e7bd94ebf59c3c05b4e6184abb7d1))


### Bug Fixes

* **store/torbox:** fix type for torrent availability ([2c8f70c](https://github.com/MunifTanjim/stremthru/commit/2c8f70cd639f5ecfcce2bb83e64b6b30ccc2d6c1))
* **stremio/wrap:** return blank manifest without config ([222c07c](https://github.com/MunifTanjim/stremthru/commit/222c07cf104dd275a763ed0c1dfab7ff0a4aaff2))

## [0.43.1](https://github.com/MunifTanjim/stremthru/compare/0.43.0...0.43.1) (2025-02-01)


### Bug Fixes

* **stremio/sidekick:** do not escape html in json for backup ([3d09426](https://github.com/MunifTanjim/stremthru/commit/3d09426b77b6bebf5364cd37c9f9d6d01c08f18b))

## [0.43.0](https://github.com/MunifTanjim/stremthru/compare/0.42.1...0.43.0) (2025-01-29)


### Features

* **config:** try to sync TUNNEL with STORE_TUNNEL ([28f5794](https://github.com/MunifTanjim/stremthru/commit/28f579409f95edc0dea838af7bd2390a8bc24d94))
* **stremio/sidekick:** add library backup/restore ([c819b87](https://github.com/MunifTanjim/stremthru/commit/c819b87f711a694717518a206fa5856c0d34f4f1))

## [0.42.1](https://github.com/MunifTanjim/stremthru/compare/0.42.0...0.42.1) (2025-01-27)


### Bug Fixes

* **stremio/store:** unescape url path component properly ([e53cf95](https://github.com/MunifTanjim/stremthru/commit/e53cf95ed50a114242127bbc8e5b374b247bea5f))

## [0.42.0](https://github.com/MunifTanjim/stremthru/compare/0.41.0...0.42.0) (2025-01-27)


### Features

* **stremio/store:** support installation per store ([8d24803](https://github.com/MunifTanjim/stremthru/commit/8d24803f4152d10c96e2da210c96bdc4f6c1341a))


### Bug Fixes

* **stremio/store:** support stores without file index ([dc92a43](https://github.com/MunifTanjim/stremthru/commit/dc92a4387bec76033203203f806b67cd4746dc5c))

## [0.41.0](https://github.com/MunifTanjim/stremthru/compare/0.40.0...0.41.0) (2025-01-26)


### Features

* **stremio:** improve proxied addon request headers adjustment ([859dbb8](https://github.com/MunifTanjim/stremthru/commit/859dbb80efde698ea3260167e18b705950c84b77))

## [0.40.0](https://github.com/MunifTanjim/stremthru/compare/0.39.0...0.40.0) (2025-01-23)


### Features

* integrate tunnel config to content proxy ([b1d0d93](https://github.com/MunifTanjim/stremthru/commit/b1d0d93325a8b73c827d526142e364e260513b93))


### Bug Fixes

* **buddy:** detect and ignore duplicate magnet cache file ([9c9f54b](https://github.com/MunifTanjim/stremthru/commit/9c9f54b7ac13874d24d521a3915eec6b85ccf5d9))
* do not reuse http transport for multiple clients ([9e50f51](https://github.com/MunifTanjim/stremthru/commit/9e50f511bf1842fd0d2d6486a8e6fdd9948d849e))
* **store/torbox:** fix type for torrent progress ([6137698](https://github.com/MunifTanjim/stremthru/commit/6137698dfe48cff3d2d3c383aada7ce8d887adeb))

## [0.39.0](https://github.com/MunifTanjim/stremthru/compare/0.38.0...0.39.0) (2025-01-22)


### Features

* improve tunnel config ([d435748](https://github.com/MunifTanjim/stremthru/commit/d4357483881c3014ea4e94aecb50686699dae4a2))

## [0.38.0](https://github.com/MunifTanjim/stremthru/compare/0.37.1...0.38.0) (2025-01-19)


### Features

* **cache:** make lru cache thread-safe ([59392c3](https://github.com/MunifTanjim/stremthru/commit/59392c3772da7494ab64b4db42f540a484fbb537))
* **peer:** send stremthru version in header ([782a8f5](https://github.com/MunifTanjim/stremthru/commit/782a8f53fef73267dd77d947f624e5005987a5f3))
* **store/offcloud:** disable expensive file size query ([24b7b06](https://github.com/MunifTanjim/stremthru/commit/24b7b06d9208bd9e8194f35751f15102e153c311))
* **store/offcloud:** try not to add existing magnet ([ba0cc0c](https://github.com/MunifTanjim/stremthru/commit/ba0cc0c6127d2eca42d3b69e830efe209b4f3393))
* **store:** set missing magnet.added_at to unix 0 timestamp ([c8b30fa](https://github.com/MunifTanjim/stremthru/commit/c8b30fa4579dce0f2d2642fee5d4d2b6f4c24b1b))
* **stremio:** adjust proxied addon request headers ([9de9759](https://github.com/MunifTanjim/stremthru/commit/9de97599c58397b29ed33e9752f948fd5f70e43e))


### Bug Fixes

* **store/offcloud:** detect error for add magnet ([e5ec209](https://github.com/MunifTanjim/stremthru/commit/e5ec20959dd33a7de3c99dfac1e39927f9846edf))
* **store/offcloud:** set correct magnet file path ([cd8818f](https://github.com/MunifTanjim/stremthru/commit/cd8818fd79fc15a2e80d1c343f004089cbaa7964))


### Performance Improvements

* **stremio/wrap:** group duplicate fetch stream calls ([881f3a7](https://github.com/MunifTanjim/stremthru/commit/881f3a7b80f2e2e209b2d02291799bd53d3eb8b3))

## [0.37.1](https://github.com/MunifTanjim/stremthru/compare/0.37.0...0.37.1) (2025-01-17)


### Bug Fixes

* **peer:** fix check magnet retry after throttling ([9450dcf](https://github.com/MunifTanjim/stremthru/commit/9450dcf12ebe48deb89bdba0fc2250e45634cb29))

## [0.37.0](https://github.com/MunifTanjim/stremthru/compare/0.36.1...0.37.0) (2025-01-17)


### Features

* **stremio/store:** add static video feedback ([b251cca](https://github.com/MunifTanjim/stremthru/commit/b251cca3643b9bf43ddde1659095feae94294e55))


### Bug Fixes

* **store/offcloud:** deal with inconsistent json type ([85ea26c](https://github.com/MunifTanjim/stremthru/commit/85ea26c1971062b1d98b4ab2d8d25a9dc8c3dcf9))
* **store/torbox:** pass file_id correctly to generate link ([f00473a](https://github.com/MunifTanjim/stremthru/commit/f00473a1171d19c0380f0ba1c51d614fd76ee4c8))
* **stremio/store:** allow clear cache from mobile app ([703a93d](https://github.com/MunifTanjim/stremthru/commit/703a93d312e067f89a9bad03bb67393e5fb054b9))
* **stremio/store:** allow head request for stream ([c1a7b4d](https://github.com/MunifTanjim/stremthru/commit/c1a7b4d0648a1c2271251b203b3353a837e5fcf2))
* **stremio/wrap:** add log for request context error ([589ceb3](https://github.com/MunifTanjim/stremthru/commit/589ceb3a3dd4e33c0df94b1459f4857a2650013f))
* **stremio/wrap:** allow head request for stream ([ba17bd2](https://github.com/MunifTanjim/stremthru/commit/ba17bd255fbb74176259f4330cd80436aa6a691f))
* **stremio/wrap:** dedupe concurrent link generation ([431507d](https://github.com/MunifTanjim/stremthru/commit/431507dc5d9e79ad86b7c6adf2032d25b39d672d))

## [0.36.1](https://github.com/MunifTanjim/stremthru/compare/0.36.0...0.36.1) (2025-01-16)


### Bug Fixes

* **store:** enable cors for link access endpoint ([e695e55](https://github.com/MunifTanjim/stremthru/commit/e695e551f12677946527d847d94444009acb9733))
* **stremio:** remove duplicate header ([88252e7](https://github.com/MunifTanjim/stremthru/commit/88252e7c1e853028662767f3be8008068e8ba726))

## [0.36.0](https://github.com/MunifTanjim/stremthru/compare/0.35.1...0.36.0) (2025-01-16)


### Features

* **buddy:** make touching local magnet cache non-blocking ([a3f6055](https://github.com/MunifTanjim/stremthru/commit/a3f60554bb3857c4e779239f479ac1a033b2b450))
* **buddy:** make upstream track magnet non-blocking ([7e11b66](https://github.com/MunifTanjim/stremthru/commit/7e11b66af08f5fe5f955b2773d1c908dd9f38068))
* **store:** increase generated link lifetime to 12 hours ([2ddab33](https://github.com/MunifTanjim/stremthru/commit/2ddab33970ca264abbd3d3108a3e192c4be7a6ab))
* **stremio/wrap:** make track magnet cache non-blocking ([c5d0212](https://github.com/MunifTanjim/stremthru/commit/c5d0212450e750fe8d98ea45ca2244461f502190))
* **stremio/wrap:** pass stream id for tracking magnet cache ([e9f7642](https://github.com/MunifTanjim/stremthru/commit/e9f76428bfdd75ba7dcada6979c70dec03653222))

## [0.35.1](https://github.com/MunifTanjim/stremthru/compare/0.35.0...0.35.1) (2025-01-15)


### Bug Fixes

* enable http keep-alive for non-public deployment ([b314a2b](https://github.com/MunifTanjim/stremthru/commit/b314a2b1f01b42c001ba97c1378f7f60bd5937b6))

## [0.35.0](https://github.com/MunifTanjim/stremthru/compare/0.34.0...0.35.0) (2025-01-15)


### Features

* disable http keep-alive ([379e054](https://github.com/MunifTanjim/stremthru/commit/379e054237f5ba271c9374c6bdd3a9634cb2d788))
* **peer:** temporarily disable on slow response ([37c103a](https://github.com/MunifTanjim/stremthru/commit/37c103aab7c56be7f5720196c216faefb61ed430))

## [0.34.0](https://github.com/MunifTanjim/stremthru/compare/0.33.0...0.34.0) (2025-01-15)


### Features

* **buddy:** do not try to track if unauthorized ([af23e0e](https://github.com/MunifTanjim/stremthru/commit/af23e0e4165fe4567a6841ac688426c629cd5f35))
* **config:** panic if unresolved tunnel ip at startup ([16f12bc](https://github.com/MunifTanjim/stremthru/commit/16f12bc41b3e1c9cbd9ce6ff66f77b79d17ea53f))
* **stremio/disabled:** autoload addons on configure ([2860e37](https://github.com/MunifTanjim/stremthru/commit/2860e37a6980117f7b942a9459039319553b2296))
* update log for proxy connection close ([d14a28c](https://github.com/MunifTanjim/stremthru/commit/d14a28cc24dc4ea1d9dafc9a46622e0991708c5d))


### Bug Fixes

* **stremio/sidekick:** stop triggering multiple downloads ([f44805f](https://github.com/MunifTanjim/stremthru/commit/f44805fa486e0196692f6652a43c6d822a5c34a6))
* **stremio:** allow cors consistently ([c7d3010](https://github.com/MunifTanjim/stremthru/commit/c7d301047e33e3f8e3f3655f97f225188cbbcac1))

## [0.33.0](https://github.com/MunifTanjim/stremthru/compare/0.32.1...0.33.0) (2025-01-10)


### Features

* **store:** add toggle for content proxy ([87d6203](https://github.com/MunifTanjim/stremthru/commit/87d620392250c81de6939afe50050d74a8857890))
* **stremio/wrap:** support content proxy for usenet links ([adb6368](https://github.com/MunifTanjim/stremthru/commit/adb6368b326a666bcc99fb547952c90eb548888b))

## [0.32.1](https://github.com/MunifTanjim/stremthru/compare/0.32.0...0.32.1) (2025-01-10)


### Bug Fixes

* **stremio/sidekick:** fix addons load button click ([5b00287](https://github.com/MunifTanjim/stremthru/commit/5b0028774f93110b078be40247aad2f3842123e1))

## [0.32.0](https://github.com/MunifTanjim/stremthru/compare/0.31.1...0.32.0) (2025-01-09)


### Features

* **config:** show ip at startup ([f4ed73c](https://github.com/MunifTanjim/stremthru/commit/f4ed73cd3ab93b9adba48f7a68815746ff6b4cdc))
* **store/pikpak:** add pagination for list magnets ([9d1b5bc](https://github.com/MunifTanjim/stremthru/commit/9d1b5bc822c0a1d27c5e265fec3fa3289d5fe740))
* **store:** forward machine ip if stream tunnel is disabled ([bc15430](https://github.com/MunifTanjim/stremthru/commit/bc1543006fbc3722f79c9d39a4f5af30855a0efd))
* **stremio/sidekick:** add addons backup/restore ([2c250cc](https://github.com/MunifTanjim/stremthru/commit/2c250cc9b95e88892ae983db61fc8e86d35e5a06))


### Bug Fixes

* **store/pikpak:** do not add duplicate magnet ([93241f3](https://github.com/MunifTanjim/stremthru/commit/93241f387004349fbb777f212fe9e2bacda5b21f))
* **store/pikpak:** fix link generation for single file ([70c4619](https://github.com/MunifTanjim/stremthru/commit/70c461939c4c53e2369499aea1efeba313475841))
* **store/pikpak:** handle expired refresh token ([806680f](https://github.com/MunifTanjim/stremthru/commit/806680fc7159577eb2f366353c250bdb79df2959))
* **stremio/sidekick:** swap whole addons section to avoid duplication ([0e5cf0a](https://github.com/MunifTanjim/stremthru/commit/0e5cf0ad152bdc276687b271a4e0d4900a4e3eec))

## [0.31.1](https://github.com/MunifTanjim/stremthru/compare/0.31.0...0.31.1) (2025-01-08)


### Bug Fixes

* **stremio/wrap:** always return transformed streams ([40ce0e6](https://github.com/MunifTanjim/stremthru/commit/40ce0e685096268c88b160c71a25dc0fc227a3b1))

## [0.31.0](https://github.com/MunifTanjim/stremthru/compare/0.30.0...0.31.0) (2025-01-08)


### Features

* **kv:** add kv storage ([90be63b](https://github.com/MunifTanjim/stremthru/commit/90be63be06c8943c08acf3e60a37a1305c297f13))
* **store/pikpak:** add initial implementation ([d757e62](https://github.com/MunifTanjim/stremthru/commit/d757e62f33e783d26725fcfb943312cbbcbeec9a))
* **stremio:** add pikpak as store ([f2636e8](https://github.com/MunifTanjim/stremthru/commit/f2636e8bf6df9424475916eedf6a31b42ca6a475))

## [0.30.0](https://github.com/MunifTanjim/stremthru/compare/0.29.0...0.30.0) (2025-01-07)


### Features

* **stremio/wrap:** prioritize file matching by filename ([d489204](https://github.com/MunifTanjim/stremthru/commit/d489204d94d8e05a810c6b99b4cd91ada61b31d3))
* **stremio:** add navbar ([d3a486f](https://github.com/MunifTanjim/stremthru/commit/d3a486f2b47781097198031c0e51995e83214abc))


### Bug Fixes

* **stremio:** hide token description for empty option ([68823d7](https://github.com/MunifTanjim/stremthru/commit/68823d73a544f6dd03ee8d69cc408a0e9ff74f5f))

## [0.29.0](https://github.com/MunifTanjim/stremthru/compare/0.28.1...0.29.0) (2025-01-05)


### Features

* **stremio/disabled:** support configure button in stremio ([947ea39](https://github.com/MunifTanjim/stremthru/commit/947ea39ce54e573eab4e84f30fea92b16a9b2207))
* **stremio/sidekick:** add missing error logs ([706e3c4](https://github.com/MunifTanjim/stremthru/commit/706e3c46c8dc325fc60d8faf7ecad38d4943fd8c))
* **stremio/wrap:** improve config validation ([2d46275](https://github.com/MunifTanjim/stremthru/commit/2d462752ea1346d32594d8def72adf60f6e9a66c))
* **stremio/wrap:** pick largest file if file name/index missing ([486e7d7](https://github.com/MunifTanjim/stremthru/commit/486e7d71967a9c2aa85bbb1f64d56bdaf45a5769))
* **stremio/wrap:** reduce time to wait for download ([692122d](https://github.com/MunifTanjim/stremthru/commit/692122d7b9245d9b2718d1cf5e5257c3e9b534d5))
* **stremio:** add link for easydebrid api key ([2ffd31d](https://github.com/MunifTanjim/stremthru/commit/2ffd31d9de70bc50cafde8313d6e12494c8fe43c))
* **stremio:** do not try to parse non-json response ([49db8c5](https://github.com/MunifTanjim/stremthru/commit/49db8c5328444c712a965fe0b632d5307b162819))
* **stremio:** reduce addon client http timeout ([5a3e5cc](https://github.com/MunifTanjim/stremthru/commit/5a3e5cc737f8457e8924df440d31fbe79a8d8829))


### Bug Fixes

* **stremio/sidekick:** close reload modal on outside click ([60b29af](https://github.com/MunifTanjim/stremthru/commit/60b29af2f33af4be6175e15a4300baad7f81e92c))
* **stremio/sidekick:** resolve path escaping issue ([6c57f34](https://github.com/MunifTanjim/stremthru/commit/6c57f34509f6bca89e0cd5b032df66c3dd0f3586))

## [0.28.1](https://github.com/MunifTanjim/stremthru/compare/0.28.0...0.28.1) (2025-01-03)


### Bug Fixes

* **cache:** remove local cache for redis ([3bc1a20](https://github.com/MunifTanjim/stremthru/commit/3bc1a200fcda36a3991c0c5aa1c5f144b325debe))

## [0.28.0](https://github.com/MunifTanjim/stremthru/compare/0.27.1...0.28.0) (2025-01-03)


### Features

* **stremio/store:** hide stremthru store if not usable ([c3a77ae](https://github.com/MunifTanjim/stremthru/commit/c3a77ae6ae9e8666b6b21ec534fbda5d20bce243))
* **stremio/wrap:** add cache only config and sort ([472d937](https://github.com/MunifTanjim/stremthru/commit/472d937cd5e1a730dfc09aa12d76c278b3b4228a))
* **stremio/wrap:** add store hint in addon name ([6b9040b](https://github.com/MunifTanjim/stremthru/commit/6b9040b121a4e1ab5a8ddfbc14fb6a61a47a9327))
* **stremio/wrap:** hide stremthru store if not usable ([b291197](https://github.com/MunifTanjim/stremthru/commit/b29119718c05c7381375cc7294e5ca946ca2341f))


### Bug Fixes

* **magnet_cache:** skip db query for empty input ([4a9cfdf](https://github.com/MunifTanjim/stremthru/commit/4a9cfdf27e3764f25510685e3148e7c5cb1b239e))
* **stremio/store:** handle empty metas ([2c91c21](https://github.com/MunifTanjim/stremthru/commit/2c91c2164fc1e21cdd30a54968bdf334a0da0a45))

## [0.27.1](https://github.com/MunifTanjim/stremthru/compare/0.27.0...0.27.1) (2025-01-03)


### Bug Fixes

* **stremio/wrap:** remove double link generation ([80606c5](https://github.com/MunifTanjim/stremthru/commit/80606c50dd737d1621f93c14a8c3674a775cc798))

## [0.27.0](https://github.com/MunifTanjim/stremthru/compare/0.26.0...0.27.0) (2025-01-03)


### Features

* **stremio/wrap:** add easydebrid as store ([2f710b9](https://github.com/MunifTanjim/stremthru/commit/2f710b9d29c81bb6c3a0779feb10011940d69640))
* **stremio/wrap:** auto-correct manifest url scheme ([5a23b0a](https://github.com/MunifTanjim/stremthru/commit/5a23b0aa5c2a1b94f7955a522144099e3c3dffc2))
* update log for track magnet ([f880962](https://github.com/MunifTanjim/stremthru/commit/f88096275a8ddf49293a66711b7b59c4dd571be5))

## [0.26.0](https://github.com/MunifTanjim/stremthru/compare/0.25.0...0.26.0) (2025-01-02)


### Features

* **store/easydebrid:** add initial implementation ([bd6f579](https://github.com/MunifTanjim/stremthru/commit/bd6f5790f503d142192a28f5f43eccac0320065d))
* **stremio/sidekick:** allow either id or name to change in reload ([18870ce](https://github.com/MunifTanjim/stremthru/commit/18870cebceca4443364fd411d179076f16ded711))


### Bug Fixes

* update type to fix unexported field issue ([de5f5ba](https://github.com/MunifTanjim/stremthru/commit/de5f5ba07be4aee6f24aa7917c6e7bca298d63b8))

## [0.25.0](https://github.com/MunifTanjim/stremthru/compare/0.24.0...0.25.0) (2025-01-02)


### Features

* allow api only store tunnel ([1867ed5](https://github.com/MunifTanjim/stremthru/commit/1867ed5b83c8be070830b208c309b77855ed60cf))
* **buddy:** send imdb id for movies ([48280b5](https://github.com/MunifTanjim/stremthru/commit/48280b5af5e025c3dd08715d73fdb13467160577))
* decrease peer http timeout ([24cfdf4](https://github.com/MunifTanjim/stremthru/commit/24cfdf40dc2ad6bea990c3c00e40c6d44c7afe96))
* **magnet_cache:** preserve more specific sid for files ([0cc4e2d](https://github.com/MunifTanjim/stremthru/commit/0cc4e2d488dfd4453d0d8748a8db08ff2e42f156))
* print config at startup ([205f9e8](https://github.com/MunifTanjim/stremthru/commit/205f9e8c7b32935aa4ddec7b9930e8c204ecb624))

## [0.24.0](https://github.com/MunifTanjim/stremthru/compare/0.23.0...0.24.0) (2025-01-02)


### Features

* **stremio:** log packed error ([d16b617](https://github.com/MunifTanjim/stremthru/commit/d16b6172d883d828120a24eec2c820241b59acde))


### Bug Fixes

* **store/realdebrid:** fix type for .progress ([f5e5a0f](https://github.com/MunifTanjim/stremthru/commit/f5e5a0f2f2f6c6b4f316866b6646f46c4dc3ce95))

## [0.23.0](https://github.com/MunifTanjim/stremthru/compare/0.22.0...0.23.0) (2025-01-01)


### Features

* add log for check manget ([cc8dafa](https://github.com/MunifTanjim/stremthru/commit/cc8dafa20b23ed660f3e1d05b5f529ff30b14bfd))
* **buddy:** log packed error ([98a0ed1](https://github.com/MunifTanjim/stremthru/commit/98a0ed1e69906a16dcb8dcefcb0867945e3edbae))
* log unexpected errors ([1d6e780](https://github.com/MunifTanjim/stremthru/commit/1d6e7809250bbf6e60554e2c2cfb9b4e757aefaa))
* **magnet_cache:** adjust stale time ([e69f156](https://github.com/MunifTanjim/stremthru/commit/e69f156491edd2d30e8b7ab3d406e2d036a44ba2))
* set http client timeout ([0803e1e](https://github.com/MunifTanjim/stremthru/commit/0803e1e7e4a3349602b0b909d8577bfc2e8fb21b))
* **store:** add configurable user agent ([1e2d46c](https://github.com/MunifTanjim/stremthru/commit/1e2d46cd9dcea23503139cfea051e592145c3be2))
* **stremio/sidekick:** show server error in ui ([580d42d](https://github.com/MunifTanjim/stremthru/commit/580d42d455c9f65f63512ae1628884e5a6d6c209))
* **stremio/sidekick:** support login with auth token ([a9f2967](https://github.com/MunifTanjim/stremthru/commit/a9f29671c66bcc1a0c2fdabb6fd25fb5500ed423))
* **stremio/wrap:** track added magnet ([e4a6c2b](https://github.com/MunifTanjim/stremthru/commit/e4a6c2b8aa21418ba18a40a30a4e2af3ce076a6d))

## [0.22.0](https://github.com/MunifTanjim/stremthru/compare/0.21.0...0.22.0) (2024-12-31)


### Features

* **store:** add config for toggling tunnel ([b4ecc59](https://github.com/MunifTanjim/stremthru/commit/b4ecc59cd95fbd990fed5fa14b66630bdd1fb5ad))
* **store:** improve magnet cache check ([f144494](https://github.com/MunifTanjim/stremthru/commit/f144494b8d3e3f45d90d94eca81a18b5df88ce85))
* **stremio/wrap:** improve magnet cache check ([b19fbf2](https://github.com/MunifTanjim/stremthru/commit/b19fbf20d7495c30c7de4232c4811ac63bd8f463))

## [0.21.0](https://github.com/MunifTanjim/stremthru/compare/0.20.1...0.21.0) (2024-12-29)


### Features

* **stremio/sidekick:** log ignored errors ([29bf96a](https://github.com/MunifTanjim/stremthru/commit/29bf96adfbe82dc4ae87af2af586d3b9d93585bd))
* **stremio/store:** log ignored errors ([9d3f641](https://github.com/MunifTanjim/stremthru/commit/9d3f6419a40125851180364382837b00f1a93a89))
* **stremio/wrap:** add logs for static video redirects ([2231568](https://github.com/MunifTanjim/stremthru/commit/2231568c1c24fdeb5d8974c6bad884187d9ebaa3))
* **stremio/wrap:** log ignored errors ([1acbeac](https://github.com/MunifTanjim/stremthru/commit/1acbeac3eafa547dc61f195b369a4eff171ec595))

## [0.20.1](https://github.com/MunifTanjim/stremthru/compare/0.20.0...0.20.1) (2024-12-28)


### Bug Fixes

* **stremio:** marshal json correctly for Resource ([8e0a196](https://github.com/MunifTanjim/stremthru/commit/8e0a1964ebf4b680715e1ee977494a0a8c431cda))

## [0.20.0](https://github.com/MunifTanjim/stremthru/compare/0.19.1...0.20.0) (2024-12-28)


### Features

* **buddy:** forward client ip ([dee549b](https://github.com/MunifTanjim/stremthru/commit/dee549b8970286ae24ccee56714227e7bc8ac60d))
* **peer:** forward client ip ([13a555d](https://github.com/MunifTanjim/stremthru/commit/13a555d5874e589856ce6e248ee665817b72b288))
* **stremio/wrap:** forward client ip ([e8c4a5d](https://github.com/MunifTanjim/stremthru/commit/e8c4a5dbc7956015cab80fff9f8c40f2589a42bb))

## [0.19.1](https://github.com/MunifTanjim/stremthru/compare/0.19.0...0.19.1) (2024-12-28)


### Bug Fixes

* **stremio/wrap:** handle missing .behaviorHints ([808136f](https://github.com/MunifTanjim/stremthru/commit/808136fd0549925d3d7ff23774d8a8f8a1afc378))

## [0.19.0](https://github.com/MunifTanjim/stremthru/compare/0.18.0...0.19.0) (2024-12-27)


### Features

* **stremio/sidekick:** add configure button for addons ([74c375a](https://github.com/MunifTanjim/stremthru/commit/74c375adb10aed1f0678ea435a6c7bcb522a4339))
* **stremio/sidekick:** support reloading addon ([0f5c4a3](https://github.com/MunifTanjim/stremthru/commit/0f5c4a343f65482c956c3cdcc771693ea08c2577))

## [0.18.0](https://github.com/MunifTanjim/stremthru/compare/0.17.0...0.18.0) (2024-12-27)


### Features

* **stremio/wrap:** add configure button for upstream addon ([618e1df](https://github.com/MunifTanjim/stremthru/commit/618e1df549906a8b40471e57a2f44f24c4ad5c3a))
* **stremio/wrap:** forward client ip to upstream addon ([2ce883b](https://github.com/MunifTanjim/stremthru/commit/2ce883bba0a13178405b707bfe5b3cac13a6f0e4))
* **stremio:** add manifest validity check ([b619f10](https://github.com/MunifTanjim/stremthru/commit/b619f10007b46f8ba018e9a6c4f8d4365f35f699))


### Bug Fixes

* **stremio/sidekick:** fix double header on successful login ([62801c9](https://github.com/MunifTanjim/stremthru/commit/62801c92ed161c60d5342de317ee09f63f56caff))

## [0.17.0](https://github.com/MunifTanjim/stremthru/compare/0.16.0...0.17.0) (2024-12-26)


### Features

* **cache:** add method AddWithLifetime ([4bdf6b4](https://github.com/MunifTanjim/stremthru/commit/4bdf6b43cc4d44fe0580956d732eb2dc0e406e9a))
* **store:** add static videos ([ba0e514](https://github.com/MunifTanjim/stremthru/commit/ba0e514f09aba83a02fd6bce7b628e1996a47214))
* **store:** include filename in generated link ([85347ac](https://github.com/MunifTanjim/stremthru/commit/85347acb1f31fe742f124110303c2391a89244d8))
* **stremio/sidekick:** improve login ux ([c4e3e34](https://github.com/MunifTanjim/stremthru/commit/c4e3e34da9d45ef62f892656c176f1f3947730fd))
* **stremio/store:** update token field description ([2591e2f](https://github.com/MunifTanjim/stremthru/commit/2591e2fbb2ee31197448cab2d43a43fc6b38ed47))
* **stremio/wrap:** support magnet hash wrapping ([bd49120](https://github.com/MunifTanjim/stremthru/commit/bd491209afecef5e50556961d42ac3e9d4fdb722))


### Bug Fixes

* core.ParseBasicAuth .Token value ([1cd1240](https://github.com/MunifTanjim/stremthru/commit/1cd124069900a80e82e6d1969c8dfd8c3064179a))
* **stremio/store:** check allowed method correctly ([3a97a23](https://github.com/MunifTanjim/stremthru/commit/3a97a231aa93cc16af6005031acd9c7344efa3f1))

## [0.16.0](https://github.com/MunifTanjim/stremthru/compare/0.15.1...0.16.0) (2024-12-23)


### Features

* **request:** support passing query params ([f634ac7](https://github.com/MunifTanjim/stremthru/commit/f634ac7861fdef1a71eb8185760c413737482d24))
* **store/offcloud:** add initial implementation ([a7156f5](https://github.com/MunifTanjim/stremthru/commit/a7156f5d05195727814ae6a745a8929885504380))


### Bug Fixes

* **stremio/store:** allow trial subscription ([329781f](https://github.com/MunifTanjim/stremthru/commit/329781fb493b106f413f301c1ec2f4759312c33c))

## [0.15.1](https://github.com/MunifTanjim/stremthru/compare/0.15.0...0.15.1) (2024-12-21)


### Bug Fixes

* **stremio/wrap:** mark as wrapped only for proxy url ([d36a696](https://github.com/MunifTanjim/stremthru/commit/d36a69658cfd8f2637189960440bd0f0f015731a))
* **stremio:** add missing types for manifest ([0c25101](https://github.com/MunifTanjim/stremthru/commit/0c251011a628ba6385855f4ff5d992ab7559df80))

## [0.15.0](https://github.com/MunifTanjim/stremthru/compare/0.14.0...0.15.0) (2024-12-21)


### Features

* **stremio/sidekick:** improve button ux ([fc2928e](https://github.com/MunifTanjim/stremthru/commit/fc2928efce1283182ba23216b9799b450c66e434))
* **stremio/sidekick:** update login ui ([1176f5a](https://github.com/MunifTanjim/stremthru/commit/1176f5ade98fe6642ac3a8407485248fb973cec9))
* **stremio:** add description for addons ([e7f5758](https://github.com/MunifTanjim/stremthru/commit/e7f5758972b0f50f415dde928d9b81104025a02b))

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
