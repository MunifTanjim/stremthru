-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torznab_indexer_syncinfo` (
  `type` varchar NOT NULL,
  `id` varchar NOT NULL,
  `sid` varchar NOT NULL,
  `queued_at` datetime,
  `synced_at` datetime,
  PRIMARY KEY (`type`, `id`, `sid`)
);

CREATE INDEX `idx_torznab_indexer_syncinfo_pending` ON `torznab_indexer_syncinfo` (`queued_at`, `synced_at`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_torznab_indexer_syncinfo_pending`;
DROP TABLE IF EXISTS `torznab_indexer_syncinfo`;
-- +goose StatementEnd
