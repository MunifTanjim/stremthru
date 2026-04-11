-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `usenet_server_stats` (
    `id` varchar PRIMARY KEY,
    `server_id` varchar NOT NULL DEFAULT '',
    `nzb_hash` varchar NOT NULL DEFAULT '',
    `segments_fetched` int NOT NULL DEFAULT 0,
    `bytes_downloaded` int NOT NULL DEFAULT 0,
    `article_not_found` int NOT NULL DEFAULT 0,
    `connection_errors` int NOT NULL DEFAULT 0,
    `latency_samples` jsonb NOT NULL DEFAULT '[]',
    `wall_clock_ms` real NOT NULL DEFAULT 0,
    `cat` datetime NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX usenet_server_stats_idx_server_cat ON `usenet_server_stats` (`server_id`, `cat`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `usenet_server_stats`;
-- +goose StatementEnd
