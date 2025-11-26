-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `torrent_review_request` (
    `id` integer PRIMARY KEY AUTOINCREMENT,
    `hash` varchar NOT NULL,
    `reason` varchar NOT NULL,
    `prev_imdb_id` varchar NOT NULL DEFAULT '',
    `imdb_id` varchar NOT NULL DEFAULT '',
    `files` json,
    `comment` varchar NOT NULL DEFAULT '',
    `ip` varchar NOT NULL,
    `created_at` datetime NOT NULL DEFAULT (unixepoch())
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS `torrent_review_request`;

-- +goose StatementEnd
