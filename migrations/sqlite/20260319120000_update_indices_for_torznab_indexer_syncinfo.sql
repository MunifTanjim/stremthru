-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_status`;

DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_queued_at_synced_at`;

CREATE INDEX `torznab_indexer_syncinfo_idx_status_queued_at` ON `torznab_indexer_syncinfo` (`status`, `queued_at` DESC);

CREATE INDEX `torznab_indexer_syncinfo_idx_indexer_id_status_queued_at` ON `torznab_indexer_syncinfo` (`indexer_id`, `status`, `queued_at` DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_indexer_id_status_queued_at`;

DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_status_queued_at`;

CREATE INDEX `torznab_indexer_syncinfo_idx_queued_at_synced_at` ON `torznab_indexer_syncinfo` (`queued_at`, `synced_at`);

CREATE INDEX `torznab_indexer_syncinfo_idx_status` ON `torznab_indexer_syncinfo` (`status`);
-- +goose StatementEnd
