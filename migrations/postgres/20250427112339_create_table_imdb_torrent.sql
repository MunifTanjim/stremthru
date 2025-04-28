-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."imdb_torrent" (
    "tid" text NOT NULL,
    "hash" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("tid", "hash")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."imdb_torrent";
-- +goose StatementEnd
