-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."anidb_torrent" (
    "tid" text NOT NULL,
    "hash" text NOT NULL,
    "s_type" text NOT NULL,
    "s" int NOT NULL,
    "ep_start" int NOT NULL,
    "ep_end" int NOT NULL,
    "eps" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("tid", "hash", "s_type", "s")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."anidb_torrent";
-- +goose StatementEnd
