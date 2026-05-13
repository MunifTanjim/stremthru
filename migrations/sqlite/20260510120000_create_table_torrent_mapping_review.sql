-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torrent_mapping_review` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `hash` varchar NOT NULL,
    `target` varchar NOT NULL,
    `reason` varchar NOT NULL,
    `prev_id` varchar,
    `mapping_id` varchar,
    `files` text,
    `comment` varchar NOT NULL DEFAULT '',
    `ip` varchar NOT NULL,
    `created_at` datetime NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX torrent_mapping_review_idx_hash ON `torrent_mapping_review` (`hash`);
CREATE INDEX torrent_mapping_review_idx_target ON `torrent_mapping_review` (`target`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `torrent_mapping_review`;
-- +goose StatementEnd
