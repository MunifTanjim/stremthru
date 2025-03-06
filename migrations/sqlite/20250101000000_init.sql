-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `magnet_cache` (
    `store` varchar NOT NULL,
    `hash` varchar NOT NULL,
    `is_cached` bool NOT NULL DEFAULT false,
    `modified_at` datetime NOT NULL DEFAULT (unixepoch()),
    `files` json NOT NULL DEFAULT (json('[]')),
    PRIMARY KEY (`store`, `hash`)
);

CREATE TABLE IF NOT EXISTS `magnet_cache_file` (
    `h` varchar NOT NULL,
    `n` varchar NOT NULL,
    `i` int NOT NULL DEFAULT -1,
    `s` int NOT NULL DEFAULT -1,
    `sid` varchar NOT NULL DEFAULT '',
    PRIMARY KEY (`h`, `n`)
);

CREATE TABLE IF NOT EXISTS `peer_token` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `created_at` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `kv` (
    `t` varchar NOT NULL DEFAULT '',
    `k` varchar NOT NULL,
    `v` varchar NOT NULL,
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`t`, `k`)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS `kv`;

DROP TABLE IF EXISTS `peer_token`;

DROP TABLE IF EXISTS `magnet_cache_file`;

DROP TABLE IF EXISTS `magnet_cache`;

-- +goose StatementEnd
