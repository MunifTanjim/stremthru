-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "public"."torrent_review_request" (
    "id" bigserial PRIMARY KEY,
    "hash" text NOT NULL,
    "reason" text NOT NULL,
    "prev_imdb_id" text NOT NULL DEFAULT '',
    "imdb_id" text NOT NULL DEFAULT '',
    "files" jsonb,
    "comment" text NOT NULL DEFAULT '',
    "ip" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "public"."torrent_review_request";

-- +goose StatementEnd
