-- +goose Up
-- +goose StatementBegin

-- 1. Rename existing tables
ALTER TABLE `torznab_indexer` RENAME TO `_torznab_indexer_old`;
ALTER TABLE `torznab_indexer_syncinfo` RENAME TO `_torznab_indexer_syncinfo_old`;

DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_queued_at_synced_at`;
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_status`;

-- 2. Create new torznab_indexer with auto-increment integer PK
CREATE TABLE IF NOT EXISTS `torznab_indexer` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `type` varchar NOT NULL,
  `old_id` varchar NULL,
  `name` varchar NOT NULL,
  `url` varchar NOT NULL,
  `api_key` varchar NOT NULL,
  `rate_limit_config_id` varchar NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch())
);

-- 3. Migrate data from old torznab_indexer
INSERT INTO `torznab_indexer` (`type`, `old_id`, `name`, `url`, `api_key`, `rate_limit_config_id`, `cat`, `uat`)
  SELECT `type`, `id`, `name`, `url`, `api_key`, `rate_limit_config_id`, `cat`, `uat`
  FROM `_torznab_indexer_old`;

-- 4. Create new torznab_indexer_syncinfo with indexer_id FK
CREATE TABLE IF NOT EXISTS `torznab_indexer_syncinfo` (
  `indexer_id` integer NOT NULL,
  `sid` varchar NOT NULL,
  `queued_at` datetime,
  `synced_at` datetime,
  `error` text,
  `result_count` integer,
  `status` varchar NOT NULL DEFAULT 'queued',
  `queries` json,
  PRIMARY KEY (`indexer_id`, `sid`)
);

CREATE INDEX `torznab_indexer_syncinfo_idx_queued_at_synced_at`
  ON `torznab_indexer_syncinfo` (`queued_at`, `synced_at`);
CREATE INDEX `torznab_indexer_syncinfo_idx_status`
  ON `torznab_indexer_syncinfo` (`status`);

-- 5. Migrate syncinfo data, joining to get new integer id
INSERT INTO `torznab_indexer_syncinfo` (`indexer_id`, `sid`, `queued_at`, `synced_at`, `error`, `result_count`, `status`, `queries`)
  SELECT ti.`id`, osi.`sid`, osi.`queued_at`, osi.`synced_at`, osi.`error`, osi.`result_count`, osi.`status`, osi.`queries`
  FROM `_torznab_indexer_syncinfo_old` osi
  INNER JOIN `torznab_indexer` ti ON osi.`type` = ti.`type` AND osi.`id` = ti.`old_id`;

-- 6. Drop old tables
DROP TABLE `_torznab_indexer_syncinfo_old`;
DROP TABLE `_torznab_indexer_old`;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 1. Rename new tables
ALTER TABLE `torznab_indexer` RENAME TO `_torznab_indexer_new`;
ALTER TABLE `torznab_indexer_syncinfo` RENAME TO `_torznab_indexer_syncinfo_new`;

DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_queued_at_synced_at`;
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_status`;

-- 2. Recreate original torznab_indexer with composite PK
CREATE TABLE IF NOT EXISTS `torznab_indexer` (
  `type` varchar NOT NULL,
  `id` varchar NOT NULL,
  `name` varchar NOT NULL,
  `url` varchar NOT NULL,
  `api_key` varchar NOT NULL,
  `rate_limit_config_id` varchar NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),
  PRIMARY KEY (`type`, `id`)
);

-- 3. Migrate data back using old_id
INSERT INTO `torznab_indexer` (`type`, `id`, `name`, `url`, `api_key`, `rate_limit_config_id`, `cat`, `uat`)
  SELECT `type`, `old_id`, `name`, `url`, `api_key`, `rate_limit_config_id`, `cat`, `uat`
  FROM `_torznab_indexer_new`
  WHERE `old_id` IS NOT NULL;

-- 4. Recreate original torznab_indexer_syncinfo with composite PK
CREATE TABLE IF NOT EXISTS `torznab_indexer_syncinfo` (
  `type` varchar NOT NULL,
  `id` varchar NOT NULL,
  `sid` varchar NOT NULL,
  `queued_at` datetime,
  `synced_at` datetime,
  `error` text,
  `result_count` integer,
  `status` varchar NOT NULL DEFAULT 'queued',
  `queries` json,
  PRIMARY KEY (`type`, `id`, `sid`)
);

CREATE INDEX `torznab_indexer_syncinfo_idx_queued_at_synced_at`
  ON `torznab_indexer_syncinfo` (`queued_at`, `synced_at`);
CREATE INDEX `torznab_indexer_syncinfo_idx_status`
  ON `torznab_indexer_syncinfo` (`status`);

-- 5. Migrate syncinfo data back, joining to map integer id to original (type, old_id)
INSERT INTO `torznab_indexer_syncinfo` (`type`, `id`, `sid`, `queued_at`, `synced_at`, `error`, `result_count`, `status`, `queries`)
  SELECT ti.`type`, ti.`old_id`, osi.`sid`, osi.`queued_at`, osi.`synced_at`, osi.`error`, osi.`result_count`, osi.`status`, osi.`queries`
  FROM `_torznab_indexer_syncinfo_new` osi
  INNER JOIN `_torznab_indexer_new` ti ON osi.`indexer_id` = ti.`id`
  WHERE ti.`old_id` IS NOT NULL;

-- 6. Drop new tables
DROP TABLE `_torznab_indexer_syncinfo_new`;
DROP TABLE `_torznab_indexer_new`;

-- +goose StatementEnd
