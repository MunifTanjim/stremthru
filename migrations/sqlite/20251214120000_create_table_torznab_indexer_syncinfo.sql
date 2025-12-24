-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torznab_indexer_syncinfo` (
  `type` varchar NOT NULL,
  `id` varchar NOT NULL,
  `sid` varchar NOT NULL,
  `queued_at` datetime,
  `synced_at` datetime,
  `error` text,
  `result_count` integer,
  PRIMARY KEY (`type`, `id`, `sid`)
);

CREATE INDEX `torznab_indexer_syncinfo_idx_queued_at_synced_at` ON `torznab_indexer_syncinfo` (`queued_at`, `synced_at`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_queued_at_synced_at`;
DROP TABLE IF EXISTS `torznab_indexer_syncinfo`;
-- +goose StatementEnd
