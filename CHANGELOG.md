# Changelog

## [0.79.2](https://github.com/MunifTanjim/stremthru/compare/0.79.1...0.79.2) (2025-06-16)


### Bug Fixes

* **stremio/list:** unmarshal trakt.tv popular list response properly ([86c6e47](https://github.com/MunifTanjim/stremthru/commit/86c6e47ed2b213bed234c7e1920636c1a55258dc))

## [0.79.1](https://github.com/MunifTanjim/stremthru/compare/0.79.0...0.79.1) (2025-06-14)


### Bug Fixes

* **torznab:** add missing imdb id for search by query ([92a2b43](https://github.com/MunifTanjim/stremthru/commit/92a2b4365a9d8ea78b9002edbef30fe6a9dcdadd))

## [0.79.0](https://github.com/MunifTanjim/stremthru/compare/0.78.4...0.79.0) (2025-06-14)


### Features

* **buddy:** support lazy pull for check magnet ([cdca861](https://github.com/MunifTanjim/stremthru/commit/cdca86122dd08ac54e1585cd677ce44a9df6401a))
* **store:** skip valid subs check for request from trusted peer ([c145c1c](https://github.com/MunifTanjim/stremthru/commit/c145c1c58ecaaf733fc2f324e0776e1846ceb973))
* **worker:** keep item in queue if processor returns error ([5044130](https://github.com/MunifTanjim/stremthru/commit/50441304392a7dcd00f4e4fdbaea93c240a572dc))


### Bug Fixes

* **imdb_title:** fix typo in sqlite search ids func ([c5be5d1](https://github.com/MunifTanjim/stremthru/commit/c5be5d1ee2b1520d37f0806c3e924b2f78975222))

## [0.78.4](https://github.com/MunifTanjim/stremthru/compare/0.78.3...0.78.4) (2025-06-13)


### Bug Fixes

* **store/premiumize:** fix add magnet for cached contents ([526dc59](https://github.com/MunifTanjim/stremthru/commit/526dc598a59fe4cf7eb9448b0e1498b270b0f7c5))

## [0.78.3](https://github.com/MunifTanjim/stremthru/compare/0.78.2...0.78.3) (2025-06-11)


### Bug Fixes

* **stremio/list:** fix typo in trakt.tv popular list ([9c5e870](https://github.com/MunifTanjim/stremthru/commit/9c5e870037250372354d86c39825599af764063b))

## [0.78.2](https://github.com/MunifTanjim/stremthru/compare/0.78.1...0.78.2) (2025-06-07)


### Bug Fixes

* **torznab:** parse query param names as case-insensitive ([dc91b56](https://github.com/MunifTanjim/stremthru/commit/dc91b561e5b17be0d75ea26fa0c23ed246a8bc31))
* **torznab:** remove tt prefix from imdb attr in response ([93bbe74](https://github.com/MunifTanjim/stremthru/commit/93bbe74d114836e47c0a7d1d9b278c8a140e77ee))
* **torznab:** respect limit and offset query ([0f2f99b](https://github.com/MunifTanjim/stremthru/commit/0f2f99bea398b1f589fd790dd8ad34255e660992))
* **torznab:** use correct type for magnet uri enclosure ([44bfa49](https://github.com/MunifTanjim/stremthru/commit/44bfa494bf4d8c8992f3f62d0a8110ca82e57ef8))
* **torznab:** use proper capabilities response ([1781d15](https://github.com/MunifTanjim/stremthru/commit/1781d15c1a6354f61afaec8c1901c56cb5babcff))

## [0.78.1](https://github.com/MunifTanjim/stremthru/compare/0.78.0...0.78.1) (2025-06-07)


### Bug Fixes

* **torznab:** support imdbid without tt prefix ([57eb3b5](https://github.com/MunifTanjim/stremthru/commit/57eb3b5ea4e45f3e945488a7b56052864c173170))

## [0.78.0](https://github.com/MunifTanjim/stremthru/compare/0.77.4...0.78.0) (2025-06-07)


### Features

* **animeapi:** sync mapping dataset ([24fa5fb](https://github.com/MunifTanjim/stremthru/commit/24fa5fbe3aa2892e89ba028eecd5a75ab8ef900e))
* **kitsu:** add integration ([cb170e2](https://github.com/MunifTanjim/stremthru/commit/cb170e28b6d56c5889d206665d5a0237dea69ced))
* **stremio/list:** add mdblist watchlist example ([d35c13b](https://github.com/MunifTanjim/stremthru/commit/d35c13ba884b4cdcd0110d597b137ae8ad5a6d8d))
* **stremio/list:** support mdblist watchlist ([cca70c2](https://github.com/MunifTanjim/stremthru/commit/cca70c23b45bedfdb9f79042d4a7e813735d1d8a))
* **stremio/torz:** support p2p stream ([ad721da](https://github.com/MunifTanjim/stremthru/commit/ad721daf9b466ec2250b5b7a2c06bf4a8a76b287))


### Bug Fixes

* **config:** do not panic on non-default non-responsive http proxy ([4f92e2b](https://github.com/MunifTanjim/stremthru/commit/4f92e2bc3ad6b4b926c7049d872df061d0adc7ce))

## [0.77.4](https://github.com/MunifTanjim/stremthru/compare/0.77.3...0.77.4) (2025-06-04)


### Bug Fixes

* **anime:** fix GetIdMapsForAniList query for postgresql ([f1576dd](https://github.com/MunifTanjim/stremthru/commit/f1576ddb6cb93eb25765daa3749296652c6f707b))
* **oauth:** handle missing oauth token ([1e21726](https://github.com/MunifTanjim/stremthru/commit/1e21726fba76d1d91a08922fde23edea83955fbf))
* **stremio/list:** preserve trakt.tv urls when auth expired/revoked ([c3fdeac](https://github.com/MunifTanjim/stremthru/commit/c3fdeacc4434428cac6d7a4fa32b7def7362b0c1))
* **trakt:** fix db queries for postgresql ([70bc3ce](https://github.com/MunifTanjim/stremthru/commit/70bc3ce85623467b4d83187236968ab9d9accb25))
* **trakt:** use fallback when period is missing ([0c1809a](https://github.com/MunifTanjim/stremthru/commit/0c1809a19ce2babab60ac1c23a814aaa1abc26fb))

## [0.77.3](https://github.com/MunifTanjim/stremthru/compare/0.77.2...0.77.3) (2025-06-03)


### Bug Fixes

* **dmm_hashlist:** ignore invalid magnet hash ([0bf142e](https://github.com/MunifTanjim/stremthru/commit/0bf142e26d0f37545677b94bc9819ba315a7cdf3))
* **oauth:** fix save token query for postgresql ([a364ee4](https://github.com/MunifTanjim/stremthru/commit/a364ee4965a4370bbde01ef48ab91c2b5803ae6f))
* **store/premiumize:** fix 414 uri too long error for check magnet ([4672111](https://github.com/MunifTanjim/stremthru/commit/46721117858b47d3e88b04ffffe4599859fd6024))
* **torrent_info:** ignore invalid magnet hash ([ff6b5b0](https://github.com/MunifTanjim/stremthru/commit/ff6b5b00f62f0ef06b134677cf1f29c21aad1aa3))

## [0.77.2](https://github.com/MunifTanjim/stremthru/compare/0.77.1...0.77.2) (2025-06-02)


### Bug Fixes

* **anilist:** update months for season detection ([344ba9b](https://github.com/MunifTanjim/stremthru/commit/344ba9b32d62b0e1c3ae867abaee1230104ba44d))
* **stremio/list:** update type for mixed catalog ([e4930ae](https://github.com/MunifTanjim/stremthru/commit/e4930aeca91cc1052cdb6131880af077ba2ddb0f))

## [0.77.1](https://github.com/MunifTanjim/stremthru/compare/0.77.0...0.77.1) (2025-06-01)


### Bug Fixes

* **stremio/list:** display correct error when trakt.tv is disabled ([93f1263](https://github.com/MunifTanjim/stremthru/commit/93f1263a814d36e2aca0a588b95c78cda433a8b1))
* **stremio/list:** render list shuffle checkbox correctly ([12230c1](https://github.com/MunifTanjim/stremthru/commit/12230c158ab13b3a33329cbaba643a3a8ccec9eb))

## [0.77.0](https://github.com/MunifTanjim/stremthru/compare/0.76.0...0.77.0) (2025-05-31)


### Features

* **oauth:** add oauth http client ([e164ac5](https://github.com/MunifTanjim/stremthru/commit/e164ac5abce4d71f815c3956868e8b57a11a2224))
* **stremio/list:** add open button for list url ([3717bc1](https://github.com/MunifTanjim/stremthru/commit/3717bc163cc5cd1d51aacd7ea3ed0dd4b9172032))
* **stremio/list:** show supported services ([3131269](https://github.com/MunifTanjim/stremthru/commit/3131269b239d841afedb7c51efba9b9681823bda))
* **stremio/list:** support anilist named search lists ([733d601](https://github.com/MunifTanjim/stremthru/commit/733d601f94c3643e798027e87445849c4e850680))
* **stremio/list:** support genre filter for anilist ([4f5acac](https://github.com/MunifTanjim/stremthru/commit/4f5acac3560c80be45fe2ac1dd568082cc96b4b5))
* **stremio/list:** support shuffle per list ([a75ccff](https://github.com/MunifTanjim/stremthru/commit/a75ccfff461e7d47ef1c2d6ace840157a29afa2f))
* **stremio/list:** support trakt.tv ([2679d51](https://github.com/MunifTanjim/stremthru/commit/2679d51fb3d261204f0cfb9b7608f679da9db2a1))
* **stremio/list:** support trakt.tv recommendations ([39c098b](https://github.com/MunifTanjim/stremthru/commit/39c098b51430161f60683d39ec387ae7a9916eb6))
* **stremio/list:** support trakt.tv standard user lists ([b81fee6](https://github.com/MunifTanjim/stremthru/commit/b81fee6947280f2c155ec1d1b7186147a76a7775))


### Bug Fixes

* **anilist:** add mutex for list fetching ([d8e2fb1](https://github.com/MunifTanjim/stremthru/commit/d8e2fb18201918f7f09e6731fcf606d8dc66988e))
* **anilist:** use fallback for missing title ([08d1d76](https://github.com/MunifTanjim/stremthru/commit/08d1d76d392c7b03935ab08f2a8565b9ce5f4449))
* **anizip:** handle 404 response ([ebf7fa3](https://github.com/MunifTanjim/stremthru/commit/ebf7fa3ef1a9c98c136463b75e020a27d8949cfe))
* **stremio/list:** preserve config on error ([91341fc](https://github.com/MunifTanjim/stremthru/commit/91341fc77d1b57eff295f8d42404dee6a44b9ae2))
* **stremio/list:** use correct image for anilist poster ([bd413a6](https://github.com/MunifTanjim/stremthru/commit/bd413a6c19fe7bd7c01642abf126cd8991a6a7d6))
* **stremio/torz:** respect cached only config ([7987d93](https://github.com/MunifTanjim/stremthru/commit/7987d93afe913102d7170c74f585648303c04de1))
* **util:** handle empty RepeatJoin ([9c9e54f](https://github.com/MunifTanjim/stremthru/commit/9c9e54f485b8137ca35b5b0c3d49928afc617777))

## [0.76.0](https://github.com/MunifTanjim/stremthru/compare/0.75.0...0.76.0) (2025-05-24)


### Features

* **store/easydebrid:** handle account not premium error ([72d38c2](https://github.com/MunifTanjim/stremthru/commit/72d38c2f8162a0a720f8d45ec7f0ee51db361c26))
* **stremio/list:** fetch and use imdb title meta from mdblist ([8dd5d80](https://github.com/MunifTanjim/stremthru/commit/8dd5d805ca9d1bbd6f63954d308131038199ef6e))
* **stremio/list:** support anilist ([d0769d0](https://github.com/MunifTanjim/stremthru/commit/d0769d0c308160ee102e2221427223a6efbf422e))
* **stremio/torz:** priotize match using sid before filename for series ([73b43e8](https://github.com/MunifTanjim/stremthru/commit/73b43e8ef4371afe6f45bf5118fa37aabd6a2e80))
* **stremio/wrap:** rearrange configure form fields ([4c53e90](https://github.com/MunifTanjim/stremthru/commit/4c53e90057462779f26e6af0dca23e1f66d81993))
* **stremio/wrap:** support rpdb for catalogs ([4edcd78](https://github.com/MunifTanjim/stremthru/commit/4edcd78dd33abc7ece1711e4c941ce0b4d7b3b9a))


### Bug Fixes

* **stremio/list:** ignore mdblist items with missing id ([6c1153e](https://github.com/MunifTanjim/stremthru/commit/6c1153ea1b774389a7dd7f0e8d878e8f9032ebc4))
* **stremio/list:** resolve various data integrity issues ([a89a86d](https://github.com/MunifTanjim/stremthru/commit/a89a86d7b218268b12b0fe30c08cbcccb364b24d))
* **stremio/torz:** do not show non-video file in stream description ([abc2c15](https://github.com/MunifTanjim/stremthru/commit/abc2c1545fef84eb5e10f5d530179c1ba675f245))

## [0.75.0](https://github.com/MunifTanjim/stremthru/compare/0.74.1...0.75.0) (2025-05-20)


### Features

* **stremio/list:** dedupe lists when importing my lists ([f3e1e37](https://github.com/MunifTanjim/stremthru/commit/f3e1e37738f101e244fbf3ad2bbeba69e93c8f68))
* **stremio/list:** disable autocomplete for rpdb api key ([08cd7d7](https://github.com/MunifTanjim/stremthru/commit/08cd7d7d9457da7dd545c9bcda2e1b2e540d1034))
* **stremio/list:** improve empty lists validation ([00bebc4](https://github.com/MunifTanjim/stremthru/commit/00bebc430c13f276eabbdf796cafa198dc19a522))
* **stremio/list:** improve template for mdblist api key ([7efcc81](https://github.com/MunifTanjim/stremthru/commit/7efcc81b867da13d588d160ca4785f36d809c009))
* **stremio/list:** prepare to support multiple services ([e8bb0f1](https://github.com/MunifTanjim/stremthru/commit/e8bb0f1e0de7e68f58e2c69adb0152e208566deb))
* **stremio/list:** support custom list name ([e82d447](https://github.com/MunifTanjim/stremthru/commit/e82d447790d85d907062e3c692e0046eb894aab8))
* **stremio/list:** support saved userdata ([e0ec191](https://github.com/MunifTanjim/stremthru/commit/e0ec1912e7a82fe8d95fe02d633e471b1a297a0f))
* **stremio/sidekick:** support hiding catalog from board ([0d9cce8](https://github.com/MunifTanjim/stremthru/commit/0d9cce850f95f73b2cbcd9f738d3e4b9f430f4bb))


### Bug Fixes

* **stremio/list:** insert items for large lists in chunks ([17b67c9](https://github.com/MunifTanjim/stremthru/commit/17b67c9dc8ee04fef5784feedf05eed82f684716))

## [0.74.1](https://github.com/MunifTanjim/stremthru/compare/0.74.0...0.74.1) (2025-05-19)


### Bug Fixes

* **stremio/list:** add database migration for postgresql ([0e2dd36](https://github.com/MunifTanjim/stremthru/commit/0e2dd36d833d3a9d370848f6f40425ae82b25780))

## [0.74.0](https://github.com/MunifTanjim/stremthru/compare/0.73.3...0.74.0) (2025-05-18)


### Features

* **db:** add dialect specific Tx.Exec ([e5ceb31](https://github.com/MunifTanjim/stremthru/commit/e5ceb3172ca19366099ea23c7210b7bb03dec9a1))
* **stremio/list:** add auth ui for private instance ([d72fb22](https://github.com/MunifTanjim/stremthru/commit/d72fb2233664262567e315050f7d755faa7e591a))
* **stremio/list:** add rpdb poster support ([68ac0ed](https://github.com/MunifTanjim/stremthru/commit/68ac0ed47de4541d4095d92ff3d48fdf6db1e8b2))
* **stremio/list:** initial implementation ([4c93e24](https://github.com/MunifTanjim/stremthru/commit/4c93e24a4aa2989c0a5683a7b16d2c44961e05ab))
* **stremio/list:** support genre filter ([ccd8a87](https://github.com/MunifTanjim/stremthru/commit/ccd8a87bf1aed313e6bab66da36e33cf9db6721f))
* **stremio/list:** support mdblist import my lists ([7f294cf](https://github.com/MunifTanjim/stremthru/commit/7f294cf2e2f372e27fc011730db0812383288474))
* **stremio/list:** support shuffle ([ddb13fc](https://github.com/MunifTanjim/stremthru/commit/ddb13fc62e3b8c7cb354714f6045fc05bdd83843))
* **stremio/list:** validate mdblist api key on install ([578e3aa](https://github.com/MunifTanjim/stremthru/commit/578e3aad06f30e4023fbf08b6da89cebc00026b7))


### Bug Fixes

* **stremio/wrap:** auto-open auth modal on error ([6e0daad](https://github.com/MunifTanjim/stremthru/commit/6e0daadb810fac5a8fad01964aae87c363b1f716))

## [0.73.3](https://github.com/MunifTanjim/stremthru/compare/0.73.2...0.73.3) (2025-05-18)


### Bug Fixes

* **db:** break out of sqlite infinite retry loop ([b2ff6b6](https://github.com/MunifTanjim/stremthru/commit/b2ff6b67f41b8ab6b552e95368c84cb9b811b1c4))

## [0.73.2](https://github.com/MunifTanjim/stremthru/compare/0.73.1...0.73.2) (2025-05-17)


### Bug Fixes

* **stremio/torz:** ignore non-video files for stream ([96f25f4](https://github.com/MunifTanjim/stremthru/commit/96f25f428cb1e32a6672e83ef01306d31debdd2d))
* **stremio/wrap:** ignore non-video files for stream ([a73cafc](https://github.com/MunifTanjim/stremthru/commit/a73cafc73ecfae25193e9dc4bfeaaca124a9a0b6))

## [0.73.1](https://github.com/MunifTanjim/stremthru/compare/0.73.0...0.73.1) (2025-05-17)


### Bug Fixes

* **stremio/store:** ignore non-video files for stream ([3cc69ff](https://github.com/MunifTanjim/stremthru/commit/3cc69ffa45b9317d7db6c805d76986d97383bc21))

## [0.73.0](https://github.com/MunifTanjim/stremthru/compare/0.72.2...0.73.0) (2025-05-17)


### Features

* **store/realdebrid:** add list downloads endpoint to api client ([d6d5d3c](https://github.com/MunifTanjim/stremthru/commit/d6d5d3c2237a2a6aa9f2d43ce8941f5509c54024))
* **stremio/store:** add support for realdebrid webdl ([088af96](https://github.com/MunifTanjim/stremthru/commit/088af9624ddae3f7deea2177a5614e8275e2f977))
* **stremio/torz:** pull torrents from peer ([e41056c](https://github.com/MunifTanjim/stremthru/commit/e41056c925ffab669131cfb7fba3fa1ccf62e7ff))
* **stremio:** improve episode file matching using sid for playback ([46fd314](https://github.com/MunifTanjim/stremthru/commit/46fd31427e378fb2cb592bd4c5b67c9c29345f74))
* **stremio:** show stream proxy indicator consistently ([2aee50e](https://github.com/MunifTanjim/stremthru/commit/2aee50ea4037cb00181e8cb996ddf4031b669dd6))
* **torrent_info:** improve ListByStremId for season packs ([da36531](https://github.com/MunifTanjim/stremthru/commit/da365313254c82e6082c659ed7d6e62811b2f682))
* **torrent_info:** improve ListHashesByStremId for season packs ([7f1ae56](https://github.com/MunifTanjim/stremthru/commit/7f1ae567af35479da589280bb7cf9a3abea88eaf))


### Bug Fixes

* **stremio/sidekick:** fix modal close button ([f997f48](https://github.com/MunifTanjim/stremthru/commit/f997f48b9e88059281c1176e6f538e950d7fba00))
* **stremio/store:** clear rd downloads cache on action ([5fed2c2](https://github.com/MunifTanjim/stremthru/commit/5fed2c225a5fd705396a83ffb24ba1b58caf8b23))
* **stremio/store:** fix title detection for series episode ([bf00a78](https://github.com/MunifTanjim/stremthru/commit/bf00a78dcf4089cc0056a1ee9130670f0c87896b))
* **stremio:** resolve missing early return for errors ([7539fcd](https://github.com/MunifTanjim/stremthru/commit/7539fcdcc927b390f586af70ad9416f7f71b0b1c))
* **worker:** resolve torrent pusher memory leak ([6d6aaa9](https://github.com/MunifTanjim/stremthru/commit/6d6aaa91a24e9bfe741b8581bbcbe470a771df62))


### Performance Improvements

* **torrent_stream:** tweak pull torrent frequency ([6f538b5](https://github.com/MunifTanjim/stremthru/commit/6f538b5188863e569d41a8961ae11035e45efa3a))

## [0.72.2](https://github.com/MunifTanjim/stremthru/compare/0.72.1...0.72.2) (2025-05-16)


### Bug Fixes

* **stremio:** handle empty template ([6fb1632](https://github.com/MunifTanjim/stremthru/commit/6fb1632066d72f0bcf660b1544a74c358c40cd17))
* **util:** fix panic recovery ([805a952](https://github.com/MunifTanjim/stremthru/commit/805a9529774caefc655e832aeb953efff90f3674))
* **util:** handle nil error in panic recovery ([a789366](https://github.com/MunifTanjim/stremthru/commit/a789366bf7a46b66b8344f4ddcbe1398b7219d0a))

## [0.72.1](https://github.com/MunifTanjim/stremthru/compare/0.72.0...0.72.1) (2025-05-16)


### Bug Fixes

* **stremio/transformer:** add panic recovery for template exec ([31845df](https://github.com/MunifTanjim/stremthru/commit/31845df415e2783cd879ebc1db6163a84d6bc8ff))

## [0.72.0](https://github.com/MunifTanjim/stremthru/compare/0.71.0...0.72.0) (2025-05-15)


### Features

* **imdb_title:** support incremental sync ([d581386](https://github.com/MunifTanjim/stremthru/commit/d581386b9d2dd6c4de918fbb0c2ceecc05b262a6))
* **store/torbox:** add webdl endpoints to api client ([6e0e97f](https://github.com/MunifTanjim/stremthru/commit/6e0e97f47f20bc8b0a8b328b4dc45f670e698882))
* **stremio/sidekick:** increase session duration to 7 days ([4ec0c91](https://github.com/MunifTanjim/stremthru/commit/4ec0c918effef410fa814d5105d898b566a2cf22))
* **stremio/store:** add support for torbox webdl ([2f21977](https://github.com/MunifTanjim/stremthru/commit/2f2197787b757713683f63a03285b917c92512d6))
* **stremio/store:** make webdl opt-in ([d98dc94](https://github.com/MunifTanjim/stremthru/commit/d98dc944384c63057598ddd8b2d268c1de578bad))
* **stremio/torz:** initial implementation ([3bf342b](https://github.com/MunifTanjim/stremthru/commit/3bf342bbd1bc972e418ca4ad457bd9f5b8f35a83))
* **stremio:** do not show disabled addons in ui ([b1fd301](https://github.com/MunifTanjim/stremthru/commit/b1fd301251edf30a7ed08f6de91ebbe05c54c5ec))
* **stremio:** share admin authorization across addons ([19b2e01](https://github.com/MunifTanjim/stremthru/commit/19b2e01d45989e488e3d603d8db131474512e3b2))
* **torrent_info:** add parse method ([5705972](https://github.com/MunifTanjim/stremthru/commit/5705972718c4bc8d00494d76a80ea58dd318bdb9))
* **torrent_info:** upgrade go-ptt ([a6563e6](https://github.com/MunifTanjim/stremthru/commit/a6563e6856ccee72523b1ea447ab47bf2b117129))


### Bug Fixes

* **buddy:** handle upstream check manget limit ([0ae3a1d](https://github.com/MunifTanjim/stremthru/commit/0ae3a1ddffb58872f714764b74df5afe2b0f15ff))
* **store/torbox:** track uncached magnet in local db ([b7283c1](https://github.com/MunifTanjim/stremthru/commit/b7283c12004fd21d3c389c07e36b1fef7b34c55c))
* **stremio:** resolve some issues with transformer ([52aea2b](https://github.com/MunifTanjim/stremthru/commit/52aea2b74eb83d549cf64fc408b1d2a85bdc40e1))
* **torrent_stream:** ignore file with empty name ([c8295bf](https://github.com/MunifTanjim/stremthru/commit/c8295bf3137960111362df491bfd33029f1bde19))

## [0.71.0](https://github.com/MunifTanjim/stremthru/compare/0.70.10...0.71.0) (2025-05-10)


### Features

* **config:** add store client user agent ([84d8fe8](https://github.com/MunifTanjim/stremthru/commit/84d8fe86c334de82a13786c5c8264684b14fb997))
* **config:** allow unsetting peer uri ([07a96eb](https://github.com/MunifTanjim/stremthru/commit/07a96eb24581b35b4cf1a29eb6adc51d4ad59dd7))
* **config:** show details about tunnel ips ([58d8677](https://github.com/MunifTanjim/stremthru/commit/58d8677d538510c14a2c1ed6ac6694145b4ee5e8))
* **magnet_cache:** update stale time ([23f0ed1](https://github.com/MunifTanjim/stremthru/commit/23f0ed19f6f05405816bf7184fd9cd0b0f16febd))
* **proxy:** accept explicit filename for url ([21e104e](https://github.com/MunifTanjim/stremthru/commit/21e104efa61114482b94ae13327fc50f03504e28))
* **stremio/sidekick:** support renaming catalogs ([d746ac5](https://github.com/MunifTanjim/stremthru/commit/d746ac5744c8d52b2438da76e121fadf900dfd35))
* **stremio/store:** support hiding catalogs ([cd4bbb9](https://github.com/MunifTanjim/stremthru/commit/cd4bbb9956254688274d4962e46b474182c7ece0))
* **stremio/store:** support hiding streams ([cd5208e](https://github.com/MunifTanjim/stremthru/commit/cd5208eb23b25a9769a3ec094494bc75d446464c))
* **stremio/wrap:** document default stream sort config ([3eaa265](https://github.com/MunifTanjim/stremthru/commit/3eaa265b9e71686120698af57143cb1f6a2ae74f))
* **stremio:** revamp stream transformer ([0419abe](https://github.com/MunifTanjim/stremthru/commit/0419abebdc90d3840029589969ab5cb4e42c2716))


### Bug Fixes

* **stremio/wrap:** fix regression for resolution sort ([969522d](https://github.com/MunifTanjim/stremthru/commit/969522d3dba9e7912654f7a4f153359799e22bbc))
* **util:** fix ToSize conversion ([f485614](https://github.com/MunifTanjim/stremthru/commit/f48561496c5ceee31540ff87600a2438d44039fa))

## [0.70.10](https://github.com/MunifTanjim/stremthru/compare/0.70.9...0.70.10) (2025-05-06)


### Bug Fixes

* **stremio/sidekick:** extract disabled addon manifest url correctly ([005051a](https://github.com/MunifTanjim/stremthru/commit/005051a8766dfe240ea4b2414209bc8b0dd865b6))

## [0.70.9](https://github.com/MunifTanjim/stremthru/compare/0.70.8...0.70.9) (2025-05-05)


### Bug Fixes

* **stremio/wrap:** stop showing uncached content for easydebrid ([082c2cb](https://github.com/MunifTanjim/stremthru/commit/082c2cbbb8ade4da5b9ed0e870774ea0ca8f2213))

## [0.70.8](https://github.com/MunifTanjim/stremthru/compare/0.70.7...0.70.8) (2025-05-04)


### Bug Fixes

* **stremio/wrap:** fix typo in store limit check ([741ec4d](https://github.com/MunifTanjim/stremthru/commit/741ec4d541331612edf834bdb7a2a4d253855f17))

## [0.70.7](https://github.com/MunifTanjim/stremthru/compare/0.70.6...0.70.7) (2025-05-04)


### Bug Fixes

* **proxy:** move away from problematic Proxy-Authorization header ([845affd](https://github.com/MunifTanjim/stremthru/commit/845affdc0f07bec1ef6afab73d586dad75d41d64))

## [0.70.6](https://github.com/MunifTanjim/stremthru/compare/0.70.5...0.70.6) (2025-05-04)


### Bug Fixes

* **proxy:** accept req_headers for each url separately ([404a6bc](https://github.com/MunifTanjim/stremthru/commit/404a6bca85a9e0f99c6f9fbc72c4fa1fc9515f97))
* **proxy:** redact token in query param for request logs ([3e1e74e](https://github.com/MunifTanjim/stremthru/commit/3e1e74e6887122894dd1ace0dd40f4b4015f7ab1))

## [0.70.5](https://github.com/MunifTanjim/stremthru/compare/0.70.4...0.70.5) (2025-05-03)


### Bug Fixes

* **worker:** fix torrent_parser crash when dmm_hashlist is disabled ([9539c83](https://github.com/MunifTanjim/stremthru/commit/9539c8335c11c136a064f53ef19d8453f49a521b))

## [0.70.4](https://github.com/MunifTanjim/stremthru/compare/0.70.3...0.70.4) (2025-05-03)


### Bug Fixes

* **torrent_info:** upgrade go-ptt ([7472a07](https://github.com/MunifTanjim/stremthru/commit/7472a076fa8b91ca685b79fe9f634223c55f9459))
* **torznab:** do not send error for empty Search result ([b93aca6](https://github.com/MunifTanjim/stremthru/commit/b93aca67fe96a67043b64e0fa3359a042ec9db0c))

## [0.70.3](https://github.com/MunifTanjim/stremthru/compare/0.70.2...0.70.3) (2025-05-03)


### Bug Fixes

* **db:** escape double quote in PrepareFTS5Query ([944a165](https://github.com/MunifTanjim/stremthru/commit/944a165c6a5917f0ec1cc18046fbf087865dc40a))

## [0.70.2](https://github.com/MunifTanjim/stremthru/compare/0.70.1...0.70.2) (2025-05-03)


### Bug Fixes

* **torrent_info:** fix GetUnmappedHashes to only include parsed ones ([111fb56](https://github.com/MunifTanjim/stremthru/commit/111fb56019f9ffa801c2fb11edc58379caded6e1))

## [0.70.1](https://github.com/MunifTanjim/stremthru/compare/0.70.0...0.70.1) (2025-05-03)


### Bug Fixes

* **imdb_title:** fix SearchIds query for sqlite ([c8a9924](https://github.com/MunifTanjim/stremthru/commit/c8a9924a0b17915541b06f0e1e99f02f80bdadc8))

## [0.70.0](https://github.com/MunifTanjim/stremthru/compare/0.69.0...0.70.0) (2025-05-02)


### Features

* **config:** add config for data directory ([3f1cb0a](https://github.com/MunifTanjim/stremthru/commit/3f1cb0a5420ff6db2f21018c633c84c580df1b9e))
* **config:** add config to toggle specific feature ([0e54c2e](https://github.com/MunifTanjim/stremthru/commit/0e54c2e27615fe34acfe64ae4dcfba40c5a0deb8))
* **config:** add STREMTHRU_ENV config ([a4f8656](https://github.com/MunifTanjim/stremthru/commit/a4f86569ea9c0487f1cc27ff998b4edf40345624))
* **config:** merge STREMTHRU_STREMIO_ADDON into STREMTHRU_FEATURE ([3115fd6](https://github.com/MunifTanjim/stremthru/commit/3115fd6d216eaee5ece61594d17c8a7803902ade))
* **db:** do dialect detection eagerly at top level ([3e33088](https://github.com/MunifTanjim/stremthru/commit/3e33088b1e2ededf0aed30eff8d38f9c1b2a70be))
* **db:** increase sqlite busy timeout to 5s ([5667061](https://github.com/MunifTanjim/stremthru/commit/5667061524d9d12631d21c960eb887ccaa14fe18))
* **db:** retry Exec for sqlite3 busy error ([91f786b](https://github.com/MunifTanjim/stremthru/commit/91f786b1c5821c24d1e94f27e1d60de8bd291c49))
* **dmm_hashlist:** introduce dmm hashlist ([2f40853](https://github.com/MunifTanjim/stremthru/commit/2f408537165a65596803e637555ba2ddaf5755e7))
* **experiment:** add exclude_source param for zilean torrents endpoint ([33663ba](https://github.com/MunifTanjim/stremthru/commit/33663bafac005ca8a324d6ce0e53e3bf6ec2a3ec))
* **imdb_title:** add SearchIds function ([3c64469](https://github.com/MunifTanjim/stremthru/commit/3c644692572ef848630d57fb4a0831298b237e10))
* **imdb_title:** introduce imdb title ([c0d08ad](https://github.com/MunifTanjim/stremthru/commit/c0d08adaa7fd26ec80dd8a26aed6895f399b55da))
* **premiumize:** utilize torrent_info for name and size ([c9ed305](https://github.com/MunifTanjim/stremthru/commit/c9ed3054c3493714943437c5e4af0b8599a4be70))
* **proxy:** add endpoint to proxify links ([8db6b7c](https://github.com/MunifTanjim/stremthru/commit/8db6b7cebc5726a525d9c57fafcbe9f58173aa28))
* **proxy:** revamp endpoints ([20b01e7](https://github.com/MunifTanjim/stremthru/commit/20b01e728b74d7aa200d6370d5b165eb9fb4677f))
* **proxy:** support proxy link without encryption ([ef3c87a](https://github.com/MunifTanjim/stremthru/commit/ef3c87a912cf0b958ff060d3bfe6d3dff40e28bb))
* **shared/server:** respond for OPTIONS method in CORS middleware ([40c4e76](https://github.com/MunifTanjim/stremthru/commit/40c4e76034a2281f0b486d04c35057b169ad307f))
* **shared:** strip ip headers from req for proxy response ([aab82eb](https://github.com/MunifTanjim/stremthru/commit/aab82ebf3a738f730e1668e54e59ba331798e6b4))
* **shared:** support CreateProxyLink without expiration ([8e5450e](https://github.com/MunifTanjim/stremthru/commit/8e5450e7c42473b246f82c59796fe61a457c9149))
* **store/alldebrid:** update error codes ([6cdd911](https://github.com/MunifTanjim/stremthru/commit/6cdd9117debd2ce0e4582a3c50cb6c4a59976197))
* **store/debridlink:** update api client for latest updates ([c113560](https://github.com/MunifTanjim/stremthru/commit/c113560a302b5cdbdeb79d3c3f8c0a41ce33d3bc))
* **store/offcloud:** improve ListMagnets to go past first page ([e1e7876](https://github.com/MunifTanjim/stremthru/commit/e1e78769202fb3c357a4a6b5c8608c5d8842db52))
* **stremio/wrap:** log addon hostname for failed fetch streams ([eee0c1a](https://github.com/MunifTanjim/stremthru/commit/eee0c1a37506fa6d03edbe8f8ef646a911caf510))
* **stremio:** add support for semi-official verification ([154678e](https://github.com/MunifTanjim/stremthru/commit/154678ee6be022eb33c9ca792f1b67907c7845e7))
* **torrent_info:** update missing category in imdb torrent mapper worker ([4459aa8](https://github.com/MunifTanjim/stremthru/commit/4459aa8a962ae5583c36113382e904959636b828))
* **torrent_info:** use imdb torrent map for ListByStremId ([b182602](https://github.com/MunifTanjim/stremthru/commit/b18260255cc7ccbc79652eafe44aea2abaed32ad))
* **torznab:** add api endpoint ([0aa8c81](https://github.com/MunifTanjim/stremthru/commit/0aa8c8125d8c070a1d4112c38237bf680aa7bc00))
* **torznab:** support text search ([de7c401](https://github.com/MunifTanjim/stremthru/commit/de7c401297393ab1c8cc0557630c076588327011))
* **worker/store_crawler:** asynchronously crawl store when necessary ([c06215f](https://github.com/MunifTanjim/stremthru/commit/c06215fef9b844f549cc1c6dd88fe21a99a93ecf))
* **worker/torrent_parser:** do all unparsed items at once ([1f1112b](https://github.com/MunifTanjim/stremthru/commit/1f1112bbbbcb1287b6d5cd2a386978247865480a))
* **worker:** add mutual exclusion conditions for workers ([870c35d](https://github.com/MunifTanjim/stremthru/commit/870c35db169325b26b50ee23eb5d8ea21be297f6))
* **worker:** do not run sync_dmm_hashlist and torrent_parser together ([3004222](https://github.com/MunifTanjim/stremthru/commit/30042221946bc9864c58260d8955380191ad6cb2))
* **worker:** improve panic recovery ([7a1dcb9](https://github.com/MunifTanjim/stremthru/commit/7a1dcb90ea5dabb6b9613b1f6bb76333d6e7ab31))
* **worker:** map imdb tid to torrent hash ([723ed24](https://github.com/MunifTanjim/stremthru/commit/723ed24b4473c5bcd37e7fbdad1fc1c09ae74492))


### Bug Fixes

* **stremio/wrap:** decouple extractor from template ([9a59f9d](https://github.com/MunifTanjim/stremthru/commit/9a59f9d316b5802bfb8ae5c2c1cb0b8d5d364608))
* **stremio:** handle bad data type in Meta and MetaVideo ([c3381db](https://github.com/MunifTanjim/stremthru/commit/c3381db0d3464ae50ce311ea837486e9a3d03b4a))
* **stremio:** handle non-existent saved userdata id gracefully ([d1a0738](https://github.com/MunifTanjim/stremthru/commit/d1a07383cf16b55db5e873b1d979abdc0942405c))
* **torrent_info:** correct query in ListByStremId for series ([7ed8dde](https://github.com/MunifTanjim/stremthru/commit/7ed8dde36595604fb4bd513c4c0fc7626990b9d6))
* **worker/torrent_parser:** update go-ptt ([925b430](https://github.com/MunifTanjim/stremthru/commit/925b430638b46e9fd169390c3169f0fdc55d7ca6))
* **worker:** resolve nil pointer issue for JobTracker ([c66bea7](https://github.com/MunifTanjim/stremthru/commit/c66bea7e49a29b3fc447a2afbde8f4851f498483))


### Performance Improvements

* **stremio/store:** fetch streams in parallel ([2698c22](https://github.com/MunifTanjim/stremthru/commit/2698c2227bcbde5c0fb544f3973e3e1f5f9694bb))
* **stremio/store:** optimize meta fetching ([0eb4006](https://github.com/MunifTanjim/stremthru/commit/0eb4006517e7cd7da95c32bf8dc07f86fab00ef6))

## [0.69.0](https://github.com/MunifTanjim/stremthru/compare/0.68.1...0.69.0) (2025-04-23)


### Features

* **store:** log peer token validation failure ([bb1d774](https://github.com/MunifTanjim/stremthru/commit/bb1d774f7db4e166f8fa90a980b13707b50ae4f8))
* **stremio/sidekick:** display addon logo ([170e892](https://github.com/MunifTanjim/stremthru/commit/170e89274eaad065646c5f13254a9efb94495cfb))
* **stremio/sidekick:** support modifying logo ([8dee0a1](https://github.com/MunifTanjim/stremthru/commit/8dee0a1ab60dc66087b94177eb06c3a6a1d28bcf))
* **stremio/store:** add behaviorHints for streams ([42c2f15](https://github.com/MunifTanjim/stremthru/commit/42c2f155ce23416e0a04b577be146ba88ccd3118))
* **torrent_info:** support no_missing_size query param ([d6a4b89](https://github.com/MunifTanjim/stremthru/commit/d6a4b8962731241ea28d0b1100dbeb2d4474408f))


### Bug Fixes

* **core/error:** consistently add .request_id ([db7179b](https://github.com/MunifTanjim/stremthru/commit/db7179bf9da4258c45c8bba801086943eae62bf6))
* **core/error:** make .status_code consistent w/ .code ([84fd115](https://github.com/MunifTanjim/stremthru/commit/84fd115ab41ae19ba36a3d5fe81d4b5b1510eede))
* **store/torbox:** deal with inconsistent data type for error ([3f937df](https://github.com/MunifTanjim/stremthru/commit/3f937df3acc0dfae06917b306bf6f35447701996))
* **stremio/store:** explicitly set posterShape for catalog items ([58c3098](https://github.com/MunifTanjim/stremthru/commit/58c3098db56658f43313d934095254fc859625ae))
* **stremio/wrap:** handle empty store config gracefully ([f8902b3](https://github.com/MunifTanjim/stremthru/commit/f8902b3b083a5f595568d8f1e97518c2e020589b))
* **stremio:** remove stray event listener in htmx-modal ([dc29065](https://github.com/MunifTanjim/stremthru/commit/dc2906503ed1ac289df6aabb7077a2adc9a9fcdb))
* **stremio:** update type for MetaVideo.Rating ([8dcb423](https://github.com/MunifTanjim/stremthru/commit/8dcb423baa372f8e85cb43f03a7860f78cdcac26))
* **worker:** tweak torrent_pusher sid ([041808e](https://github.com/MunifTanjim/stremthru/commit/041808e07e567c4e63a3be154e0f02ca119bb61e))

## [0.68.1](https://github.com/MunifTanjim/stremthru/compare/0.68.0...0.68.1) (2025-04-22)


### Bug Fixes

* **torrent_info:** refine pull query ([7b6253d](https://github.com/MunifTanjim/stremthru/commit/7b6253d923924e893d5e65a6af4d3e35936ebf6a))

## [0.68.0](https://github.com/MunifTanjim/stremthru/compare/0.67.2...0.68.0) (2025-04-21)


### Features

* **store/torbox:** add usenet endpoints to api client ([80a2c79](https://github.com/MunifTanjim/stremthru/commit/80a2c798ad2c81001e975af3eb28be1cc3142c63))
* **store/torbox:** forward client ip ([6a42eea](https://github.com/MunifTanjim/stremthru/commit/6a42eea6f00841b5e592bc0954330b70a6b7966a))
* **stremio/store:** add support for torbox usenet ([5a4c9eb](https://github.com/MunifTanjim/stremthru/commit/5a4c9eb50b90de4a67a42125956502e0e05f3ea0))
* **stremio/usenet:** add torbox support ([51a6fa3](https://github.com/MunifTanjim/stremthru/commit/51a6fa302667f2ee1b861d7b777e9d41df8b4f8f))
* **stremio/wrap:** support copying saved userdata ([bfea0d5](https://github.com/MunifTanjim/stremthru/commit/bfea0d5decc0ad2460b0121011e3bcb562aab496))
* **torrent_info:** add endpoint for stats ([4e51c40](https://github.com/MunifTanjim/stremthru/commit/4e51c403d1f4ac76d4dc3a295ad15f7ef064e4f2))


### Bug Fixes

* **shared:** do not use url base as filename if missing extension ([1383686](https://github.com/MunifTanjim/stremthru/commit/13836868b7f663ab606360f1161280920641d9d9))
* **store/torbox:** fix total_items for list magnets ([c1d50d4](https://github.com/MunifTanjim/stremthru/commit/c1d50d4d3e9057dcbe19ad68b4b9ce9eff9140ac))

## [0.67.2](https://github.com/MunifTanjim/stremthru/compare/0.67.1...0.67.2) (2025-04-18)


### Bug Fixes

* **stremio/wrap:** fix stream endpoint for imdb ids ([4e6a702](https://github.com/MunifTanjim/stremthru/commit/4e6a702581c3826ebee5b04b03a932a5f74fade5))
* **torrent_info:** decrease trust for torrent title from torbox ([b7ef712](https://github.com/MunifTanjim/stremthru/commit/b7ef712853bc43e4c1db233b6a27e30d68d9a189))

## [0.67.1](https://github.com/MunifTanjim/stremthru/compare/0.67.0...0.67.1) (2025-04-18)


### Bug Fixes

* **stremio/wrap:** handle missing fields in built-in mediafusion extractor ([3541c31](https://github.com/MunifTanjim/stremthru/commit/3541c31e4319cb448451c3f0879612aa2c6421f0))

## [0.67.0](https://github.com/MunifTanjim/stremthru/compare/0.66.3...0.67.0) (2025-04-18)


### Features

* **experiment:** add torrents endpoint for zilean ingestion ([2a9a6e9](https://github.com/MunifTanjim/stremthru/commit/2a9a6e92ad6af28a00d0c355f68b3af52a57fb90))
* **stremio/wrap:** support empty extractor even with template ([084e0b8](https://github.com/MunifTanjim/stremthru/commit/084e0b888eb0b7c28973297f550433f367c5e8ff))


### Bug Fixes

* **stremio/wrap:** make default template consistent ([c461e9c](https://github.com/MunifTanjim/stremthru/commit/c461e9c043057b0ac8654d68519f7550191b68fb))

## [0.66.3](https://github.com/MunifTanjim/stremthru/compare/0.66.2...0.66.3) (2025-04-18)


### Bug Fixes

* **stremio/store:** support deprecated id format ([3ced695](https://github.com/MunifTanjim/stremthru/commit/3ced6952e12e174bb3185ccba48b08fff1f96d5b))

## [0.66.2](https://github.com/MunifTanjim/stremthru/compare/0.66.1...0.66.2) (2025-04-18)


### Bug Fixes

* **stremio/store:** fix id parser for backward compatibility ([146c78d](https://github.com/MunifTanjim/stremthru/commit/146c78d65aa6a8710211714c99b0fd71ef41c33c))

## [0.66.1](https://github.com/MunifTanjim/stremthru/compare/0.66.0...0.66.1) (2025-04-18)


### Bug Fixes

* **stremio/store:** add compatibility for older installation ([b69f905](https://github.com/MunifTanjim/stremthru/commit/b69f905977fc991655192103edbd1f9acd628c6f))

## [0.66.0](https://github.com/MunifTanjim/stremthru/compare/0.65.1...0.66.0) (2025-04-17)


### Features

* **store/premiumize:** improve error code detection ([1585318](https://github.com/MunifTanjim/stremthru/commit/1585318085783119fe9436cce4214b8e33fccc11))
* **stremio/store:** add fallback for parsed title ([7250851](https://github.com/MunifTanjim/stremthru/commit/725085197f98216865ccda50d091246075fc55e3))
* **stremio/store:** support multi store for st ([9ef6b37](https://github.com/MunifTanjim/stremthru/commit/9ef6b37bd0d238ca25841e9081ed23535408868f))


### Bug Fixes

* **store/alldebrid:** remove pointer from slice field ([0a1cbce](https://github.com/MunifTanjim/stremthru/commit/0a1cbce31a66bd8e0121c827a74123e1a0376d04))
* **torrent_info:** update go-ptt ([bfd57be](https://github.com/MunifTanjim/stremthru/commit/bfd57beca08d80460463f76af37dab12cbfaa7db))

## [0.65.1](https://github.com/MunifTanjim/stremthru/compare/0.65.0...0.65.1) (2025-04-17)


### Bug Fixes

* **torrent_stream:** fix query for GetStremIdByHashes ([115530e](https://github.com/MunifTanjim/stremthru/commit/115530e3e0300cbf13c7967afdc08f79fac031e2))

## [0.65.0](https://github.com/MunifTanjim/stremthru/compare/0.64.1...0.65.0) (2025-04-16)


### Features

* **torrent_info:** set up sharing ([a5f1e79](https://github.com/MunifTanjim/stremthru/commit/a5f1e79feaab76c08f87281bf0c341ecf5986e67))


### Bug Fixes

* **stremio/sidekick:** do not set configurable on reload ([a3d11fe](https://github.com/MunifTanjim/stremthru/commit/a3d11fe20b3241426b5c1b0dd0482d1ef8bc435b))
* **stremio:** normalize manifest url sceheme correctly ([2dbae6b](https://github.com/MunifTanjim/stremthru/commit/2dbae6b90d6e626a387fe381fa5b9ca970834d6c))

## [0.64.1](https://github.com/MunifTanjim/stremthru/compare/0.64.0...0.64.1) (2025-04-16)


### Bug Fixes

* **buddy:** skip bulk track for empty list ([44b90e6](https://github.com/MunifTanjim/stremthru/commit/44b90e61d2545227007659ba3d587876d3085c8a))
* **torrent_stream:** fix sql query to record streams ([2091966](https://github.com/MunifTanjim/stremthru/commit/2091966330c4d0268e3bb78ec300684e5ee488e6))

## [0.64.0](https://github.com/MunifTanjim/stremthru/compare/0.63.0...0.64.0) (2025-04-15)


### Features

* **store/pikpak:** add size to list magnet response when available ([bf94d30](https://github.com/MunifTanjim/stremthru/commit/bf94d30bfc579fcec1cc0dea96dc5e6bfdb2c8d3))
* **store/pikpak:** sort list magnet response by .addedAt ([ae7d37b](https://github.com/MunifTanjim/stremthru/commit/ae7d37bb91cb8fc3f6feba78d79df7eacde03aa5))
* **stremio/store:** add title, year, date in meta preview description ([f2fdd6f](https://github.com/MunifTanjim/stremthru/commit/f2fdd6fc0369e53d6cdf906bdc6e3af6d0e4435d))
* **stremio/store:** allow episode number without season ([887a00b](https://github.com/MunifTanjim/stremthru/commit/887a00b55a4e83ddec1466f73238d973653da7f9))
* **stremio/store:** always set parsed season and episode for videos ([494651a](https://github.com/MunifTanjim/stremthru/commit/494651a04bb5f43a7726f1a15f33ef8bd8e9448f))
* **stremio/store:** hide non-video files ([5eaba46](https://github.com/MunifTanjim/stremthru/commit/5eaba465d96a387ab4bff5801af6467d4217f770))
* **stremio/store:** increase catalog items cache time to 10m ([422e815](https://github.com/MunifTanjim/stremthru/commit/422e81576e9561c28f71a678625e678a0fce22ff))
* **torrent_info:** discard extracted data only for empty hash ([c73d9aa](https://github.com/MunifTanjim/stremthru/commit/c73d9aa8431ae48e7ab980bbf343adb9e9071789))
* **torrent_info:** extract data from mediafusion ([35f2d9f](https://github.com/MunifTanjim/stremthru/commit/35f2d9f81a651e7b832e7c519d908c4b2f1a8e19))


### Bug Fixes

* **config:** use default peer uri only if buddy uri is empty ([802c92b](https://github.com/MunifTanjim/stremthru/commit/802c92bf0b7c24320e4a604e0a9494da1fc92fbf))
* **torrent_info:** extract filename from title for torrentio correctly ([fb7cbcf](https://github.com/MunifTanjim/stremthru/commit/fb7cbcf2ee457310c9c10aeae626c40bf979b8c8))


### Performance Improvements

* **stremio/store:** cache response for fetch meta ([a9b14c7](https://github.com/MunifTanjim/stremthru/commit/a9b14c73c4d1ad0949d341c5641d50ee7d3060ef))

## [0.63.0](https://github.com/MunifTanjim/stremthru/compare/0.62.8...0.63.0) (2025-04-13)


### Features

* **store/debridlink:** forward client ip ([6561375](https://github.com/MunifTanjim/stremthru/commit/6561375509013908bec9588405e00f52388cc969))
* **stremio/store:** integrate directly into movie streams ([e632e17](https://github.com/MunifTanjim/stremthru/commit/e632e179bda3b73dec7575085dc7b292fe3034fb))
* **stremio/store:** integrate directly into series streams ([b82e978](https://github.com/MunifTanjim/stremthru/commit/b82e978fc6f2c841cdb44677373b30c1856f2783))
* **stremio/store:** support cinemeta metadata ([08305ee](https://github.com/MunifTanjim/stremthru/commit/08305eedf16a1567cce4b338cccbaa1a778788e7))
* **stremio/store:** support cinemeta metadata for series episodes ([ff100b2](https://github.com/MunifTanjim/stremthru/commit/ff100b28bf91cf935cef54b1f3ee69497435b06e))
* **stremio/store:** update catalog search strategy ([ffa6487](https://github.com/MunifTanjim/stremthru/commit/ffa6487a9b4630060e79cfec4df347ca028b182c))
* **stremio/store:** update templates for data ([d17cca7](https://github.com/MunifTanjim/stremthru/commit/d17cca783fce07e68eb0b618e26a604811816c20))


### Bug Fixes

* **stremio/store:** fix back button behavior ([cffba87](https://github.com/MunifTanjim/stremthru/commit/cffba87fd5b8f4cdcdad4557493c52fd54ee16cf))

## [0.62.8](https://github.com/MunifTanjim/stremthru/compare/0.62.7...0.62.8) (2025-04-11)


### Bug Fixes

* **torrent_info:** upgrade go-ptt ([04cc99b](https://github.com/MunifTanjim/stremthru/commit/04cc99bd4080439031b140f965999a426e9b93e8))

## [0.62.7](https://github.com/MunifTanjim/stremthru/compare/0.62.6...0.62.7) (2025-04-11)


### Bug Fixes

* **torrent_info:** upgrade go-ptt ([0654663](https://github.com/MunifTanjim/stremthru/commit/0654663dd7cf912db2fa2c142ad012e4ea3081ad))

## [0.62.6](https://github.com/MunifTanjim/stremthru/compare/0.62.5...0.62.6) (2025-04-11)


### Bug Fixes

* **torrent_info:** extract torrent info before transform ([89ec613](https://github.com/MunifTanjim/stremthru/commit/89ec613b6cae5e610ffd120d82385e01ed824746))

## [0.62.5](https://github.com/MunifTanjim/stremthru/compare/0.62.4...0.62.5) (2025-04-11)


### Bug Fixes

* **torrent_info:** log warning for failed to parse title ([21b74f4](https://github.com/MunifTanjim/stremthru/commit/21b74f400664791f0a534be0c9f419057b300cf8))

## [0.62.4](https://github.com/MunifTanjim/stremthru/compare/0.62.3...0.62.4) (2025-04-11)


### Bug Fixes

* **torrent_info:** discard bad torrent titles ([8ebd750](https://github.com/MunifTanjim/stremthru/commit/8ebd750cec6ce1fca0e213293797246035abb296))

## [0.62.3](https://github.com/MunifTanjim/stremthru/compare/0.62.2...0.62.3) (2025-04-11)


### Bug Fixes

* **torrent_info:** handle worker panic ([7748c97](https://github.com/MunifTanjim/stremthru/commit/7748c976bb3030eded9e1ef63d175a127f4932f1))

## [0.62.2](https://github.com/MunifTanjim/stremthru/compare/0.62.1...0.62.2) (2025-04-11)


### Bug Fixes

* upgrade Dockerfile golang version ([8767378](https://github.com/MunifTanjim/stremthru/commit/87673783ae0bf633f75721adfc0434b738fd6a0b))

## [0.62.1](https://github.com/MunifTanjim/stremthru/compare/0.62.0...0.62.1) (2025-04-11)


### Bug Fixes

* **store:** restore compatibility for older downstream version ([0e8923a](https://github.com/MunifTanjim/stremthru/commit/0e8923ab73a6cf46f4fe9014c3de6e033d8569ce))

## [0.62.0](https://github.com/MunifTanjim/stremthru/compare/0.61.1...0.62.0) (2025-04-11)


### Features

* make torrent_info and torrent_stream robust ([6a0aafc](https://github.com/MunifTanjim/stremthru/commit/6a0aafceaba5fd747c97f47bf0f38344dfab84e8))
* **store:** include magnet total size in response ([0230cc2](https://github.com/MunifTanjim/stremthru/commit/0230cc2c64e1cd1c47062cfa1398b3c1f267290a))
* **stremio/wrap:** add built-in peerflix extractor ([fc81408](https://github.com/MunifTanjim/stremthru/commit/fc814086f677e05bd2f08b60632e635e1aeefafb))
* **stremio/wrap:** improve built-in torrentio extractor ([91b2307](https://github.com/MunifTanjim/stremthru/commit/91b2307386b2e52f9c2f37120663c97e45708746))
* **stremio/wrap:** tag strem id for matched file ([0a4df3c](https://github.com/MunifTanjim/stremthru/commit/0a4df3c6713be92b39cf9a59d75b86258a955d27))
* **torrent_info:** add debug torrents endpoint ([173d10a](https://github.com/MunifTanjim/stremthru/commit/173d10a289027908bdfbded191a6773dc707cbba))
* **torrent_info:** collect torrent info from store operations ([5985d65](https://github.com/MunifTanjim/stremthru/commit/5985d65f5572908196c0d6ef41588e355ec69df6))
* **torrent_info:** collect torrent info from stremio/store ([0afc8e7](https://github.com/MunifTanjim/stremthru/commit/0afc8e732b7d11d7c17bf4d81c3139b05b7c81ef))
* **torrent_info:** collect torrent info from stremio/wrap ([bc5296f](https://github.com/MunifTanjim/stremthru/commit/bc5296f9b5dbb61c6b2111cdf285ee0ac16e1366))
* **torrent_info:** try to parse title at regular interval ([ca82daa](https://github.com/MunifTanjim/stremthru/commit/ca82daa96ad6a9474615f649f07c36a27b6df537))
* **torrent_stream:** rename magnet_cache_file to torrent_stream ([134316a](https://github.com/MunifTanjim/stremthru/commit/134316a032e84f20cf76ea837c163f30ab6ef9ef))
* upgrade to golang 1.24 ([80ca29e](https://github.com/MunifTanjim/stremthru/commit/80ca29ec7f9bd39f299a220e177c0cee1e61459c))


### Bug Fixes

* **magnet_cache:** do not track file with wrong sid ([b4d2892](https://github.com/MunifTanjim/stremthru/commit/b4d28921975738210b2cb3df671ea26b18299254))
* **store/debridlink:** add missing .path in add/get magnet response ([36867e0](https://github.com/MunifTanjim/stremthru/commit/36867e055b5b37dfed2f4254bff0d473ee901be1))
* **store/realdebrid:** add missing .name in add magnet response ([0c5d0aa](https://github.com/MunifTanjim/stremthru/commit/0c5d0aa0525ee06e0400b6126ef8fd22b8eee37e))
* **torrent_info:** resolve compatibility issues for postgresql ([0fca0b0](https://github.com/MunifTanjim/stremthru/commit/0fca0b08c95c3eb1639fcaa0cf371ba9010827f7))

## [0.61.1](https://github.com/MunifTanjim/stremthru/compare/0.61.0...0.61.1) (2025-03-31)


### Bug Fixes

* **stremio/wrap:** guard against invalid store config in userdata ([ecb489a](https://github.com/MunifTanjim/stremthru/commit/ecb489a65349aa01c73e191b9137ed2ebc14340d))

## [0.61.0](https://github.com/MunifTanjim/stremthru/compare/0.60.0...0.61.0) (2025-03-20)


### Features

* **stremio/wrap:** support saved userdata ([1e463d2](https://github.com/MunifTanjim/stremthru/commit/1e463d24cf1606eb5fab709caff7931b3ba6c23f))


### Bug Fixes

* **stremio/sidekick:** accept manifest url with query params ([ab6a9e8](https://github.com/MunifTanjim/stremthru/commit/ab6a9e8cea7a208325983bea7086972a486473cc))
* **stremio/wrap:** fix duplicate event listeners ([902e7f7](https://github.com/MunifTanjim/stremthru/commit/902e7f7d608865a8df3b10818f5ddb76a9d9fabc))

## [0.60.0](https://github.com/MunifTanjim/stremthru/compare/0.59.0...0.60.0) (2025-03-11)


### Features

* **stremio/sidekick:** support installing/uninstalling addon ([072a964](https://github.com/MunifTanjim/stremthru/commit/072a964e3220f6f967cefa54e2edfdecf83629f4))
* **stremio/sidekick:** support toggling configurable and protected ([d5dc313](https://github.com/MunifTanjim/stremthru/commit/d5dc313b0fa964da34b311b309d33e317d75726b))
* **stremio/wrap:** add built-in comet extractor ([eed8bb3](https://github.com/MunifTanjim/stremthru/commit/eed8bb33b7caa82ac5500ebf3a3839ab89739f09))
* **stremio/wrap:** expose no content proxy without explicit auth ([85f071f](https://github.com/MunifTanjim/stremthru/commit/85f071fe05d6362417fd9672c63093664e281b25))
* **stremio/wrap:** improve built-in torrentio extractor ([738c7bb](https://github.com/MunifTanjim/stremthru/commit/738c7bb9c2ba3c90c9d7c97d29d0ec68df89ceed))
* **stremio/wrap:** introduce admin user ([78e7346](https://github.com/MunifTanjim/stremthru/commit/78e7346758cb11cb30114cbc7c64cae4f8271fdf))
* **stremio:** update footer links ([a6b36da](https://github.com/MunifTanjim/stremthru/commit/a6b36da331d8b44c23f74872cefda27c3c11a721))


### Bug Fixes

* **stremio/wrap:** do not install without store configured ([109f07b](https://github.com/MunifTanjim/stremthru/commit/109f07b64ef76f05008de9c3094d6c2961e616fb))

## [0.59.0](https://github.com/MunifTanjim/stremthru/compare/0.58.0...0.59.0) (2025-03-09)


### Features

* **stremio/sidekick:** auto-correct manifest url on reload ([d925e51](https://github.com/MunifTanjim/stremthru/commit/d925e5115e1d62c47ea0b61d582e98899b8970df))
* **stremio/wrap:** improve some built-in extractors ([1f3ec8f](https://github.com/MunifTanjim/stremthru/commit/1f3ec8fb5626fe2ea203cf05005b5b5ba56b45e3))
* **stremio:** add common http headers for addon api calls ([6da69d4](https://github.com/MunifTanjim/stremthru/commit/6da69d4b72f36b44754688f66e244f5bc8b9f26b))

## [0.58.0](https://github.com/MunifTanjim/stremthru/compare/0.57.2...0.58.0) (2025-03-06)


### Features

* **config:** add default peer uri ([474d8e0](https://github.com/MunifTanjim/stremthru/commit/474d8e007a0c32300d9f5c3f93e796d12fa4df1c))
* **db:** add embedded schema migration ([af56775](https://github.com/MunifTanjim/stremthru/commit/af5677598bd1086994ba12175c183270bf591317))
* **db:** tweak schema migration logs ([ce2d188](https://github.com/MunifTanjim/stremthru/commit/ce2d188c9bba992f46730048de352505cf048e1d))
* **stremio/sidekick:** support name and description modification ([6927177](https://github.com/MunifTanjim/stremthru/commit/6927177f96d404053ba37d49af325296f5a95148))
* **stremio:** update links in ui ([0699c4b](https://github.com/MunifTanjim/stremthru/commit/0699c4bf2b8fd4b5bd17fe2b1aa416b4a9f0a5a1))


### Bug Fixes

* **buddy:** check only stale or missing hashes from peer ([041f484](https://github.com/MunifTanjim/stremthru/commit/041f484bb4812189a7714f196bc00c0a0db9a75c))

## [0.57.2](https://github.com/MunifTanjim/stremthru/compare/0.57.1...0.57.2) (2025-03-04)


### Bug Fixes

* **config:** do suffix match properly for STREMTHRU_TUNNEL ([89a4c6d](https://github.com/MunifTanjim/stremthru/commit/89a4c6dd1074a28d1fa1a5ba891d793d407556fa))

## [0.57.1](https://github.com/MunifTanjim/stremthru/compare/0.57.0...0.57.1) (2025-03-03)


### Bug Fixes

* **stremio/wrap:** escape filename in strem url ([cd93965](https://github.com/MunifTanjim/stremthru/commit/cd93965c3c2c56e433ef21618abc9ab83eb54e3b))

## [0.57.0](https://github.com/MunifTanjim/stremthru/compare/0.56.3...0.57.0) (2025-03-02)


### Features

* **kv:** support dynamic scope in type ([608a8a6](https://github.com/MunifTanjim/stremthru/commit/608a8a6dd5066b1bbc6db63f1ee1a7c3dbd7d2d8))
* **store:** add content proxy connection limit per user ([8a6fd00](https://github.com/MunifTanjim/stremthru/commit/8a6fd00e86251c4de6dc14762294fd77d9560f22))
* **store:** track content proxy connections per user ([2056b4b](https://github.com/MunifTanjim/stremthru/commit/2056b4b3045c7d73597e8f5707c3773a6c7983ea))


### Bug Fixes

* extract ip from r.RemoteAddr properly ([b2e40dc](https://github.com/MunifTanjim/stremthru/commit/b2e40dcb35a35e3c9b56c09d03a04b24bef9c18b))
* **store/premiumize:** handle not-premium error better ([494397b](https://github.com/MunifTanjim/stremthru/commit/494397b1b4d420942139a60aa5e8f1a791d52919))
* **store/premiumize:** isolate parent folder id cache properly ([634a0e1](https://github.com/MunifTanjim/stremthru/commit/634a0e10f67c8c6158df22b572c52bfe13676d27))

## [0.56.3](https://github.com/MunifTanjim/stremthru/compare/0.56.2...0.56.3) (2025-02-26)


### Bug Fixes

* **stremio/wrap:** set correct store hint in name ([d0e88e0](https://github.com/MunifTanjim/stremthru/commit/d0e88e0c93a836ccfa2c45165ca011690deff475))

## [0.56.2](https://github.com/MunifTanjim/stremthru/compare/0.56.1...0.56.2) (2025-02-24)


### Bug Fixes

* **stremio/wrap:** select store correctly in strem url ([81a11ee](https://github.com/MunifTanjim/stremthru/commit/81a11ee01de94ab8b9932342e3ec524f2aaabaf5))

## [0.56.1](https://github.com/MunifTanjim/stremthru/compare/0.56.0...0.56.1) (2025-02-23)


### Bug Fixes

* **stremio/wrap:** patch nil pointer dereference ([7e16ed2](https://github.com/MunifTanjim/stremthru/commit/7e16ed25a155551ced118fd55ce8bd9023b2ba23))

## [0.56.0](https://github.com/MunifTanjim/stremthru/compare/0.55.1...0.56.0) (2025-02-23)


### Features

* **store/easydebrid:** store magnet cache info in local db ([fede7bd](https://github.com/MunifTanjim/stremthru/commit/fede7bdcadab54a57a582edf2647f777d5c9e1ec))
* **stremio/wrap:** add raw template ([69707bb](https://github.com/MunifTanjim/stremthru/commit/69707bb4945c8d7eddae4b1b97f0770c3d991f92))
* **stremio/wrap:** include site in default template ([400bfe1](https://github.com/MunifTanjim/stremthru/commit/400bfe1848b42a05ca4eea28aedb39eb3f1be5bd))
* **stremio/wrap:** support multiple stores ([f0b791c](https://github.com/MunifTanjim/stremthru/commit/f0b791c87d6280c39426500f5b804cef90854fce))
* **stremio/wrap:** try to match series file using sid ([1bce88f](https://github.com/MunifTanjim/stremthru/commit/1bce88f1472be7f34131c5f4e21ead4568a41739))
* **stremio:** tweak manifest for addon catalog ([bb5bbfe](https://github.com/MunifTanjim/stremthru/commit/bb5bbfe85239f1b440b22e46aef3fb333a05bae2))
* use shorter request id ([ea4a83a](https://github.com/MunifTanjim/stremthru/commit/ea4a83a109a533df7a835a67572083a7cd18d10c))


### Bug Fixes

* **buddy:** always set local_only for peer check magnet ([968674b](https://github.com/MunifTanjim/stremthru/commit/968674b48b3fc69a9b93be7a623ce83e989c74b1))
* **stremio/wrap:** add missing space in cached stream name ([fadd97b](https://github.com/MunifTanjim/stremthru/commit/fadd97b0e3a25a809b194e8efe903f164e5cda4b))
* **stremio/wrap:** handle url encoded path in manifest url ([7ab2866](https://github.com/MunifTanjim/stremthru/commit/7ab2866e9169f4b9e38e394a6dded93484425a45))

## [0.55.1](https://github.com/MunifTanjim/stremthru/compare/0.55.0...0.55.1) (2025-02-16)


### Bug Fixes

* **cache:** make lru read thread-safe ([2c85350](https://github.com/MunifTanjim/stremthru/commit/2c8535095e2da9674d0a0d648f72708c296ea5b8))

## [0.55.0](https://github.com/MunifTanjim/stremthru/compare/0.54.2...0.55.0) (2025-02-15)


### Features

* **store/pikpak:** improve error construction ([3923386](https://github.com/MunifTanjim/stremthru/commit/3923386775ed34c8e1c5b08a111b8ec71828ea90))


### Bug Fixes

* **stremio/wrap:** do shallow copy before transform ([af676c1](https://github.com/MunifTanjim/stremthru/commit/af676c1c0c6f0c2f99f71b5652cc9e061a91c4a4))

## [0.54.2](https://github.com/MunifTanjim/stremthru/compare/0.54.1...0.54.2) (2025-02-14)


### Bug Fixes

* **stremio/wrap:** improve extractor for debridio ([e8c835d](https://github.com/MunifTanjim/stremthru/commit/e8c835de57657c43d8e79581e631d6583cb574ff))
* **stremio/wrap:** skip parsing catalog id for single addon ([41bec5b](https://github.com/MunifTanjim/stremthru/commit/41bec5b7afa51675abafc6537f2653dcc53e6250))

## [0.54.1](https://github.com/MunifTanjim/stremthru/compare/0.54.0...0.54.1) (2025-02-13)


### Bug Fixes

* **magnet_cache:** discard file idx for non-rd stores ([0881355](https://github.com/MunifTanjim/stremthru/commit/0881355825d016d34ff992ce48f8bf773a85cf38))

## [0.54.0](https://github.com/MunifTanjim/stremthru/compare/0.53.0...0.54.0) (2025-02-13)


### Features

* **buddy:** include store name in logs ([8db6749](https://github.com/MunifTanjim/stremthru/commit/8db6749eb5ace1ba1cfac891994ab7a7370e4f6d))


### Bug Fixes

* **store/torbox:** always do check magnet api call locally ([444839e](https://github.com/MunifTanjim/stremthru/commit/444839e8f9146ec6d0b1661cb78937f82a5e9819))

## [0.53.0](https://github.com/MunifTanjim/stremthru/compare/0.52.0...0.53.0) (2025-02-13)


### Features

* **stremio/sidekick:** add logo ([fd4f74c](https://github.com/MunifTanjim/stremthru/commit/fd4f74c977af5f06d83ca24ad639c93a52289ea8))
* **stremio/store:** add logo ([c990301](https://github.com/MunifTanjim/stremthru/commit/c9903010a3778de861229c22ee1440640c3ca146))
* **stremio/wrap:** add logo ([823c641](https://github.com/MunifTanjim/stremthru/commit/823c6417337241666a5e07d6c3bc1412a9567d9b))
* **stremio:** add logo for root addon ([3c02559](https://github.com/MunifTanjim/stremthru/commit/3c0255980e308ca83098f27537783cb011b9582b))


### Bug Fixes

* **stremio:** properly set root manifest id ([337defd](https://github.com/MunifTanjim/stremthru/commit/337defdcba1aa13562c4bf2999a2cbe08f64a879))

## [0.52.0](https://github.com/MunifTanjim/stremthru/compare/0.51.0...0.52.0) (2025-02-13)


### Features

* **stremio:** add addon catalog ([5967eb3](https://github.com/MunifTanjim/stremthru/commit/5967eb3ef9a8941d835dcc032874fa78390cc550))


### Bug Fixes

* **stremio/wrap:** allow clearing extractor/template ([9479cd5](https://github.com/MunifTanjim/stremthru/commit/9479cd5b4dde38fd368412e2c30ec1e7f3b312b1))

## [0.51.0](https://github.com/MunifTanjim/stremthru/compare/0.50.0...0.51.0) (2025-02-12)


### Features

* **store/torbox:** store magnet cache info in local db ([bb8ee7d](https://github.com/MunifTanjim/stremthru/commit/bb8ee7d10c0c9b24ef6ca7ac564886c62a03aa22))
* **stremio/sidekick:** add usage warning ([914a79d](https://github.com/MunifTanjim/stremthru/commit/914a79db7b572febef3335d5157caf83bfbccb4b))
* **stremio/sidekick:** support addons reset ([b8ba57b](https://github.com/MunifTanjim/stremthru/commit/b8ba57ba0c0ee3f9e125e4092c99d89f9fd962bc))
* **stremio/wrap:** add extractor for orion ([9e5a246](https://github.com/MunifTanjim/stremthru/commit/9e5a246faa91c6308f43db6798288eac88cb6832))
* **stremio/wrap:** always overwrite built-in transformer entitites ([6519ecb](https://github.com/MunifTanjim/stremthru/commit/6519ecb0e9d78c8095b20337a3ad4313f314999e))
* **stremio/wrap:** auto-correct manifest url suffix ([6e1a991](https://github.com/MunifTanjim/stremthru/commit/6e1a9911ea20054cd962ba50ed9839aa391ee408))
* **stremio/wrap:** improve mediafusion extractor ([78e68e1](https://github.com/MunifTanjim/stremthru/commit/78e68e155b2f7b76583a92ce556ad76fe45608ec))
* **stremio/wrap:** keep built-in transformer entities in-memory ([d210896](https://github.com/MunifTanjim/stremthru/commit/d2108965a2520ac75594731073a8de8a3814f495))


### Bug Fixes

* **buddy:** do not report unknown hashes as uncached ([a686ffa](https://github.com/MunifTanjim/stremthru/commit/a686ffada145d05745d1b1a74fc7edf306578696))
* **store/torbox:** extract file name from path ([6d030d2](https://github.com/MunifTanjim/stremthru/commit/6d030d23fadf5da27ce521e8f8604dc07e66637f))
* **stremio/sidekick:** fix response for addon move/reload ([49dc265](https://github.com/MunifTanjim/stremthru/commit/49dc2654788c2226663b0ab839aec0363d53152c))
* **stremio/wrap:** check for unconfigured addon ([880c17b](https://github.com/MunifTanjim/stremthru/commit/880c17b5ce38dc7688e666ea4b949795e70efb9d))
* **stremio/wrap:** do not use empty extracted values ([f142df0](https://github.com/MunifTanjim/stremthru/commit/f142df01fb2317818153e866f5dcb0caa42b27c3))

## [0.50.0](https://github.com/MunifTanjim/stremthru/compare/0.49.0...0.50.0) (2025-02-11)


### Features

* **server:** cleanup request logging ([5b82a27](https://github.com/MunifTanjim/stremthru/commit/5b82a277d4855b326306fa542f582aa686cd669a))
* **store/torbox:** chunk cached magnet check api call ([2059792](https://github.com/MunifTanjim/stremthru/commit/20597921613afb7e6b80167450e292555e6c2979))
* **store:** enforce max 500 items for check magnet ([6529f9d](https://github.com/MunifTanjim/stremthru/commit/6529f9de3e1cce876f08f1783f5e17724ea0ccbc))


### Bug Fixes

* **magnet_cache:** resolve too many sql variables issue ([8f2b41e](https://github.com/MunifTanjim/stremthru/commit/8f2b41e83682a7c408b580bbf60553e5f7a07d35))


### Performance Improvements

* **store:** improve check magnet performance for ad/dl/rd ([2ce8baa](https://github.com/MunifTanjim/stremthru/commit/2ce8baa13f68464b8f8ef3bab73cc08ad07e370b))

## [0.49.0](https://github.com/MunifTanjim/stremthru/compare/0.48.0...0.49.0) (2025-02-10)


### Features

* **store:** redact token from link access endpoint logss ([8c77a66](https://github.com/MunifTanjim/stremthru/commit/8c77a667912fe621953b0e28e294cc4701e49b43))
* **stremio/wrap:** update transformer seed entity ids ([2abd3b0](https://github.com/MunifTanjim/stremthru/commit/2abd3b03bac96e66fae3edb8c115146dc70e667a))
* **stremio:** change version placement in ui ([caa7f2c](https://github.com/MunifTanjim/stremthru/commit/caa7f2c0eb9954c7e6525b1af8c9b4ed6e8d7c73))


### Bug Fixes

* **stremio/sidekick:** fix response for addon toggle ([8431e5c](https://github.com/MunifTanjim/stremthru/commit/8431e5c39fd8eb83e785381f90fd9de5f3e647c8))
* **stremio/wrap:** fix file matching using pattern ([ab02b1c](https://github.com/MunifTanjim/stremthru/commit/ab02b1c599cf163031bd854d2251d65d817ed8b9))


### Performance Improvements

* **kv:** optimize queries ([6fdd737](https://github.com/MunifTanjim/stremthru/commit/6fdd737f5c97305b995d28e89693cb55084da155))

## [0.48.0](https://github.com/MunifTanjim/stremthru/compare/0.47.0...0.48.0) (2025-02-09)


### Features

* add better logging ([55b1419](https://github.com/MunifTanjim/stremthru/commit/55b14190b7cefc2126f42ab9cef25293d3cc864f))
* **stremio/wrap:** support multiple upstream addons for public instance ([4147999](https://github.com/MunifTanjim/stremthru/commit/41479999baa9f3a1cf3c649aeda33e6f1f489b92))
* **stremio/wrap:** update addon description in manifest ([ab1b672](https://github.com/MunifTanjim/stremthru/commit/ab1b6728b05cf8b7252b3ba35ca59cf631288fc7))
* **stremio/wrap:** use extracted data for cached streams grouping ([0252fd4](https://github.com/MunifTanjim/stremthru/commit/0252fd42c4d1ae36a61c675a9cf5f1710cd67314))
* **stremio:** add missing request_id in logs ([1ed2da2](https://github.com/MunifTanjim/stremthru/commit/1ed2da2b1825b913433378f6ad25fd86c350e48b))

## [0.47.0](https://github.com/MunifTanjim/stremthru/compare/0.46.3...0.47.0) (2025-02-08)


### Features

* **stremio/wrap:** allow addon w/ direct link w/o content proxy ([0832fa4](https://github.com/MunifTanjim/stremthru/commit/0832fa4279912cabd267c778a927a8e4118b345e))
* **stremio/wrap:** keep some manifest fields for single addon ([1df96dd](https://github.com/MunifTanjim/stremthru/commit/1df96dda68885b13c017457c3061ec48f40e9f45))
* **stremio/wrap:** support reconfiguring store ([d11271a](https://github.com/MunifTanjim/stremthru/commit/d11271a011c964ac96f64e6f877cf2d14ce4697b))


### Bug Fixes

* **stremio/wrap:** adjust spacing styles on configure page ([4aae7ca](https://github.com/MunifTanjim/stremthru/commit/4aae7ca4db32f76f0c7b670d87f1327907da27a7))

## [0.46.3](https://github.com/MunifTanjim/stremthru/compare/0.46.2...0.46.3) (2025-02-06)


### Bug Fixes

* **stremio/wrap:** fix public instance usage ([a8b6fda](https://github.com/MunifTanjim/stremthru/commit/a8b6fda65cfb06dfa35404cbe72547217fb9e4aa))

## [0.46.2](https://github.com/MunifTanjim/stremthru/compare/0.46.1...0.46.2) (2025-02-06)


### Bug Fixes

* **stremio/wrap:** handle transform failure gracefully ([1924c00](https://github.com/MunifTanjim/stremthru/commit/1924c004033cf465b88faf62b0770a379b05e70f))

## [0.46.1](https://github.com/MunifTanjim/stremthru/compare/0.46.0...0.46.1) (2025-02-06)


### Bug Fixes

* **stremio/wrap:** surface error for failed database operation ([594409e](https://github.com/MunifTanjim/stremthru/commit/594409e1d34102d8d9c30f4e9fa02114a39a261c))

## [0.46.0](https://github.com/MunifTanjim/stremthru/compare/0.45.1...0.46.0) (2025-02-06)


### Features

* **stremio/wrap:** add support for stream sort ([0d173a2](https://github.com/MunifTanjim/stremthru/commit/0d173a2ade440cd996f2cd95a082ad450ebd5921))

## [0.45.1](https://github.com/MunifTanjim/stremthru/compare/0.45.0...0.45.1) (2025-02-06)


### Bug Fixes

* **stremio/wrap:** cleanup dev codes ([881a861](https://github.com/MunifTanjim/stremthru/commit/881a86160824e87e634873af414837879514ee89))

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
