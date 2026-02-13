-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."nzb_info" (
    "id" text PRIMARY KEY,
    "hash" text NOT NULL,
    "name" text NOT NULL DEFAULT '',
    "size" bigint NOT NULL DEFAULT 0,
    "file_count" integer NOT NULL DEFAULT 0,
    "password" text NOT NULL DEFAULT '',
    "url" text NOT NULL DEFAULT '',
    "files" jsonb,
    "streamable" boolean NOT NULL DEFAULT false,
    "user" text NOT NULL DEFAULT '',
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX nzb_info_uidx_hash ON "public"."nzb_info" ("hash");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."nzb_info";
-- +goose StatementEnd
