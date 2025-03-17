-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `stremio_userdata` (
    `addon` varchar NOT NULL,
    `key` varchar NOT NULL,
    `value` json NOT NULL,
    `name` varchar NOT NULL,
    `disabled` bool NOT NULL DEFAULT false,
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`addon`, `key`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `stremio_userdata`;
-- +goose StatementEnd
