-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torrent_stream` (
    `h` varchar NOT NULL,
    `n` varchar NOT NULL,
    `i` int NOT NULL DEFAULT -1,
    `s` int NOT NULL DEFAULT -1,
    `sid` varchar NOT NULL DEFAULT '',
    `src` varchar NOT NULL DEFAULT '',
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),

    PRIMARY KEY (`h`, `n`)
);

INSERT INTO `torrent_stream` (h, n, i, s, sid)
SELECT h, n, i, s, sid FROM `magnet_cache_file`;

DROP TABLE `magnet_cache_file`;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
