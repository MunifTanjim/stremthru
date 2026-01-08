-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `rate_limit_config` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `limit` int NOT NULL,
    `window` varchar NOT NULL,
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `rate_limit_config`;
-- +goose StatementEnd
