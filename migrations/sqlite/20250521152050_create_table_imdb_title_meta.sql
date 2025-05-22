-- +goose Up
-- +goose StatementBegin
ALTER TABLE `imdb_title_map`
    ADD COLUMN `mal` varchar NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS `imdb_title_meta` (
    `tid` varchar NOT NULL,
    `description` varchar NOT NULL,
    `runtime` int NOT NULL,
    `poster` varchar NOT NULL,
    `backdrop` varchar NOT NULL,
    `trailer` varchar NOT NULL,
    `rating` int  NOT NULL,
    `mpa_rating` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`tid`)
);

CREATE TABLE IF NOT EXISTS `imdb_title_genre` (
    `tid` varchar NOT NULL,
    `genre` varchar NOT NULL,

    PRIMARY KEY (`tid`, `genre`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `imdb_title_genre`;

DROP TABLE IF EXISTS `imdb_title_meta`;

ALTER TABLE `imdb_title_map`
    DROP COLUMN `mal`;
-- +goose StatementEnd
