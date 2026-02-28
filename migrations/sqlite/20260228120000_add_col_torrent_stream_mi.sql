-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_stream` ADD COLUMN `mi` jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torrent_stream` DROP COLUMN `mi`;
-- +goose StatementEnd
