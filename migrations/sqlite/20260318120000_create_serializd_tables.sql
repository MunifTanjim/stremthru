-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `serializd_list` (
    `id` varchar NOT NULL,
    `name` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `serializd_item` (
    `id` int NOT NULL,
    `name` varchar NOT NULL,
    `banner_image` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`id`)
);

CREATE TABLE IF NOT EXISTS `serializd_list_item` (
    `list_id` varchar NOT NULL,
    `item_id` int NOT NULL,
    `idx` int NOT NULL,
    PRIMARY KEY (`list_id`, `item_id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `serializd_list_item`;
DROP TABLE IF EXISTS `serializd_item`;
DROP TABLE IF EXISTS `serializd_list`;
-- +goose StatementEnd
