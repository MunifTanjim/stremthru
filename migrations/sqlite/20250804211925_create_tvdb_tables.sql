-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `tvdb_list` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `slug` varchar NOT NULL,
    `overview` varchar NOT NULL,
    `is_official` bool NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `tvdb_item` (
    `id` int NOT NULL,
    `type` varchar NOT NULL,
    `name` varchar NOT NULL,
    `overview` varchar NOT NULL,
    `year` int NOT NULL,
    `runtime` int NOT NULL,
    `poster` varchar NOT NULL,
    `background` varchar NOT NULL,
    `trailer` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`, `type`)
);

CREATE TABLE IF NOT EXISTS `tvdb_item_genre` (
    `item_id` int NOT NULL,
    `item_type` varchar NOT NULL,
    `genre` int NOT NULL,
    PRIMARY KEY (`item_id`, `item_type`, `genre`)
);

CREATE TABLE IF NOT EXISTS `tvdb_list_item` (
    `list_id` varchar NOT NULL,
    `item_id` int NOT NULL,
    `item_type` varchar NOT NULL,
    `order` int NOT NULL,
    PRIMARY KEY (`list_id`, `item_id`, `item_type`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `tvdb_list_item`;
DROP TABLE IF EXISTS `tvdb_item_genre`;
DROP TABLE IF EXISTS `tvdb_item`;
DROP TABLE IF EXISTS `tvdb_list`;
-- +goose StatementEnd
