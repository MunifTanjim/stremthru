-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `anidb_tvdb_episode_map` (
    `anidb_id` varchar NOT NULL,
    `tvdb_id` varchar NOT NULL,
    `anidb_season` int NOT NULL,
    `tvdb_season` int NOT NULL,
    `start` int NOT NULL,
    `end` int NOT NULL,
    `offset` int NOT NULL,
    `before` varchar NOT NULL,
    `map` varchar NOT NULL,

    PRIMARY KEY (`anidb_id`, `tvdb_id`, `anidb_season`, `tvdb_season`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `anidb_tvdb_episode_map`;
-- +goose StatementEnd
