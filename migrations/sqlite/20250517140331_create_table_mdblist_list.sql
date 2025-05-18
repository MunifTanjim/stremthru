-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `mdblist_list` (
    `id` int NOT NULL,
    `user_id` int NOT NULL,
    `user_name` varchar NOT NULL,
    `name` varchar NOT NULL,
    `slug` varchar NOT NULL,
    `description` varchar NOT NULL DEFAULT '',
    `mediatype` varchar NOT NULL,
    `dynamic` bool NOT NULL DEFAULT false,
    `private` bool NOT NULL DEFAULT false,
    `likes` int NOT NULL DEFAULT 0,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `mdblist_item` (
    `id` int NOT NULL,
    `rank` int NOT NULL,
    `adult` bool NOT NULL DEFAULT false,
    `title` varchar NOT NULL,
    `poster` varchar NOT NULL,
    `imdb_id` varchar NOT NULL,
    `tvdb_id` int NOT NULL,
    `language` varchar NOT NULL,
    `mediatype` varchar NOT NULL,
    `release_year` int NOT NULL,
    `spoken_language` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `mdblist_list_item` (
    `list_id` int NOT NULL,
    `item_id` int NOT NULL,

    PRIMARY KEY (`list_id`, `item_id`)
);

CREATE TABLE IF NOT EXISTS `mdblist_item_genre` (
    `item_id` int NOT NULL,
    `genre` varchar NOT NULL,

    PRIMARY KEY (`item_id`, `genre`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `mdblist_item_genre`;
DROP TABLE IF EXISTS `mdblist_list_item`;
DROP TABLE IF EXISTS `mdblist_item`;
DROP TABLE IF EXISTS `mdblist_list`;
-- +goose StatementEnd
