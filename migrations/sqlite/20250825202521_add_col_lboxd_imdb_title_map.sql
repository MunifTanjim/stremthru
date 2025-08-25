-- +goose Up
-- +goose StatementBegin
ALTER TABLE `imdb_title_map` ADD COLUMN `lboxd` varchar NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `imdb_title_map` DROP COLUMN `lboxd`;
-- +goose StatementEnd
