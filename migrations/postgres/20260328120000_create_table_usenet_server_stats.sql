-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."usenet_server_stats" (
    "id" text PRIMARY KEY,
    "server_id" text NOT NULL DEFAULT '',
    "nzb_hash" text NOT NULL DEFAULT '',
    "segments_fetched" bigint NOT NULL DEFAULT 0,
    "bytes_downloaded" bigint NOT NULL DEFAULT 0,
    "article_not_found" bigint NOT NULL DEFAULT 0,
    "connection_errors" bigint NOT NULL DEFAULT 0,
    "latency_samples" jsonb NOT NULL DEFAULT '[]',
    "wall_clock_ms" double precision NOT NULL DEFAULT 0,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX usenet_server_stats_idx_server_cat ON "public"."usenet_server_stats" ("server_id", "cat");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."usenet_server_stats";
-- +goose StatementEnd
