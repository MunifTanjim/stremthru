-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_info` ADD COLUMN `indexer` varchar NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torrent_info` DROP COLUMN `indexer`;
-- +goose StatementEnd
