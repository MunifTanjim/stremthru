-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."torrent_mapping_review" (
    "id" SERIAL PRIMARY KEY,
    "hash" text NOT NULL,
    "target" text NOT NULL,
    "reason" text NOT NULL,
    "prev_id" text,
    "mapping_id" text,
    "files" text,
    "comment" text NOT NULL DEFAULT '',
    "ip" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX torrent_mapping_review_idx_hash ON "public"."torrent_mapping_review" ("hash");
CREATE INDEX torrent_mapping_review_idx_target ON "public"."torrent_mapping_review" ("target");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."torrent_mapping_review";
-- +goose StatementEnd
