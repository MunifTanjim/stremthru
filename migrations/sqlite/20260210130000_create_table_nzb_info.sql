-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `nzb_info` (
    `id` varchar PRIMARY KEY,
    `hash` varchar NOT NULL,
    `name` varchar NOT NULL DEFAULT '',
    `size` int NOT NULL DEFAULT 0,
    `file_count` integer NOT NULL DEFAULT 0,
    `password` varchar NOT NULL DEFAULT '',
    `url` varchar NOT NULL DEFAULT '',
    `files` jsonb,
    `streamable` boolean NOT NULL DEFAULT false,
    `user` varchar NOT NULL DEFAULT '',
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch())
);

CREATE UNIQUE INDEX nzb_info_uidx_hash ON `nzb_info` (`hash`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `nzb_info`;
-- +goose StatementEnd
