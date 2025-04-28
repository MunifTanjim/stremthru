-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `imdb_torrent` (
    `tid` varchar NOT NULL,
    `hash` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`tid`, `hash`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `imdb_torrent`;
-- +goose StatementEnd
