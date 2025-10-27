-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_info` ADD COLUMN `private` bool NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torrent_info` DROP COLUMN `private`;
-- +goose StatementEnd
