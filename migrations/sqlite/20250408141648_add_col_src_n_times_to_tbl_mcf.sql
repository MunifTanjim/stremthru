-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `magnet_cache_file_tmp_updated` (
    `h` varchar NOT NULL,
    `n` varchar NOT NULL,
    `i` int NOT NULL DEFAULT -1,
    `s` int NOT NULL DEFAULT -1,
    `sid` varchar NOT NULL DEFAULT '',
    `src` varchar NOT NULL DEFAULT '',
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    `sat` datetime,

    PRIMARY KEY (`h`, `n`)
);

INSERT INTO `magnet_cache_file_tmp_updated` (h, n, i, s, sid)
SELECT h, n, i, s, sid FROM `magnet_cache_file`;

DROP TABLE `magnet_cache_file`;

ALTER TABLE `magnet_cache_file_tmp_updated`
    RENAME TO `magnet_cache_file`;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
