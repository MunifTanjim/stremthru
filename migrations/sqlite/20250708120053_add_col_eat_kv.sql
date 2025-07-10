-- +goose Up
-- +goose StatementBegin
ALTER TABLE `kv` ADD COLUMN `eat` datetime;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `kv` DROP COLUMN `eat` datetime;
-- +goose StatementEnd
