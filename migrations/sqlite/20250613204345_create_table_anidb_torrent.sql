-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `anidb_torrent` (
    `tid` varchar NOT NULL,
    `hash` varchar NOT NULL,
    `s_type` varchar NOT NULL,
    `s` int NOT NULL,
    `ep_start` int NOT NULL,
    `ep_end` int NOT NULL,
    `eps` varchar NOT NULL,
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`tid`, `hash`, `s_type`, `s`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `anidb_torrent`;
-- +goose StatementEnd
