-- +goose Up
-- +goose StatementBegin
DELETE FROM `torznab_indexer_syncinfo`;
ALTER TABLE `torznab_indexer_syncinfo` ADD COLUMN `status` varchar NOT NULL DEFAULT 'queued';
ALTER TABLE `torznab_indexer_syncinfo` ADD COLUMN `queries` json;
CREATE INDEX `torznab_indexer_syncinfo_idx_status` ON `torznab_indexer_syncinfo` (`status`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `torznab_indexer_syncinfo_idx_status`;
ALTER TABLE `torznab_indexer_syncinfo` DROP COLUMN `queries`;
ALTER TABLE `torznab_indexer_syncinfo` DROP COLUMN `status`;
-- +goose StatementEnd
