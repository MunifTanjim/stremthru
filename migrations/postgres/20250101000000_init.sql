-- +goose Up
-- +goose StatementBegin

CREATE SCHEMA IF NOT EXISTS "public";

CREATE TABLE IF NOT EXISTS "public"."magnet_cache" (
    "store" character varying NOT NULL,
    "hash" character varying NOT NULL,
    "is_cached" boolean NOT NULL DEFAULT false,
    "modified_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "files" json NOT NULL DEFAULT '[]',
    PRIMARY KEY ("store", "hash")
);

CREATE TABLE IF NOT EXISTS "public"."magnet_cache_file" (
    "h" character varying NOT NULL,
    "n" character varying NOT NULL,
    "i" integer NOT NULL DEFAULT -1,
    "s" bigint NOT NULL DEFAULT -1,
    "sid" character varying NOT NULL DEFAULT '',
    PRIMARY KEY ("h", "n")
);

CREATE TABLE IF NOT EXISTS "public"."peer_token" (
    "id" character varying NOT NULL,
    "name" character varying NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."kv" (
    "k" text NOT NULL,
    "v" text NOT NULL,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "t" text NOT NULL DEFAULT '',
    PRIMARY KEY ("t", "k")
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "public"."kv";

DROP TABLE IF EXISTS "public"."peer_token";

DROP TABLE IF EXISTS "public"."magnet_cache_file";

DROP TABLE IF EXISTS "public"."magnet_cache";

-- +goose StatementEnd
