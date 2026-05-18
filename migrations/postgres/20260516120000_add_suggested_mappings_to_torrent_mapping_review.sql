-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_mapping_review` ADD COLUMN `suggested_mappings` text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torrent_mapping_review` DROP COLUMN `suggested_mappings`;
-- +goose StatementEnd
