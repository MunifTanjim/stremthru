-- +goose Up
-- +goose StatementBegin
ALTER TABLE `letterboxd_list` ADD COLUMN `v` int NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `letterboxd_list` DROP COLUMN `v`;
-- +goose StatementEnd
