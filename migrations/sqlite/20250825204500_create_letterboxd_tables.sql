-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `letterboxd_list` (
    `id` varchar NOT NULL,
    `user_id` varchar NOT NULL,
    `user_name` varchar NOT NULL,
    `name` varchar NOT NULL,
    `slug` varchar NOT NULL,
    `description` varchar NOT NULL,
    `private` bool NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `letterboxd_item` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `release_year` int NOT NULL,
    `runtime` int NOT NULL,
    `rating` int NOT NULL,
    `adult` bool NOT NULL,
    `poster` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `letterboxd_item_genre` (
    `item_id` varchar NOT NULL,
    `genre_id` varchar NOT NULL,
    PRIMARY KEY (`item_id`, `genre_id`)
);

CREATE TABLE IF NOT EXISTS `letterboxd_list_item` (
    `list_id` varchar NOT NULL,
    `item_id` varchar NOT NULL,
    `rank` int NOT NULL,
    PRIMARY KEY (`list_id`, `item_id`)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS `letterboxd_list_item`;
DROP TABLE IF EXISTS `letterboxd_item_genre`;
DROP TABLE IF EXISTS `letterboxd_item`;
DROP TABLE IF EXISTS `letterboxd_list`;

-- +goose StatementEnd
