-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `tmdb_list` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `description` varchar NOT NULL,
    `private` bool NOT NULL,
    `account_id` varchar NOT NULL,
    `username` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `tmdb_item` (
    `id` int NOT NULL,
    `type` varchar NOT NULL,
    `is_partial` bool NOT NULL,
    `title` varchar NOT NULL,
    `orig_title` varchar NOT NULL,
    `overview` varchar NOT NULL,
    `release_date` date,
    `is_adult` bool NOT NULL,
    `backdrop` varchar NOT NULL,
    `poster` varchar NOT NULL,
    `popularity` real NOT NULL,
    `vote_average` real NOT NULL,
    `vote_count` int NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`, `type`)
);

CREATE TABLE IF NOT EXISTS `tmdb_item_genre` (
    `item_id` int NOT NULL,
    `item_type` varchar NOT NULL,
    `genre_id` int NOT NULL,
    PRIMARY KEY (`item_id`, `item_type`, `genre_id`)
);

CREATE TABLE IF NOT EXISTS `tmdb_list_item` (
    `list_id` varchar NOT NULL,
    `item_id` int NOT NULL,
    `item_type` varchar NOT NULL,
    `idx` int NOT NULL,
    PRIMARY KEY (`list_id`, `item_id`, `item_type`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `tmdb_list_item`;
DROP TABLE IF EXISTS `tmdb_item_genre`;
DROP TABLE IF EXISTS `tmdb_item`;
DROP TABLE IF EXISTS `tmdb_list`;
-- +goose StatementEnd
