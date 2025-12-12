-- +goose Up
-- +goose StatementBegin
ALTER TABLE `trakt_list_item` ADD COLUMN `added_at` datetime;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `trakt_list_item` DROP COLUMN `added_at`;
-- +goose StatementEnd
