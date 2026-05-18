-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_mapping_review` ADD COLUMN `status` varchar NOT NULL DEFAULT 'pending';
ALTER TABLE `torrent_mapping_review` ADD COLUMN `resolved_at` datetime;
CREATE INDEX torrent_mapping_review_idx_status ON `torrent_mapping_review` (`status`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS torrent_mapping_review_idx_status;
ALTER TABLE `torrent_mapping_review` DROP COLUMN `resolved_at`;
ALTER TABLE `torrent_mapping_review` DROP COLUMN `status`;
-- +goose StatementEnd
