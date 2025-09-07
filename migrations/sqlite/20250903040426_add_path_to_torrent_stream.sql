-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torrent_stream` RENAME COLUMN `n` TO `p`;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torrent_stream` RENAME COLUMN `p` TO `n`;
-- +goose StatementEnd
