-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `dmm_hashlist` (
    `id` varchar NOT NULL,
    `entry_count` int NOT NULL,
    `cat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `dmm_hashlist`;
-- +goose StatementEnd
