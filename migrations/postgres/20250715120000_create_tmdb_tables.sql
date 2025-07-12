-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "public"."tmdb_list" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "description" text NOT NULL,
    "private" boolean NOT NULL,
    "account_id" text NOT NULL,
    "username" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."tmdb_item" (
    "id" int NOT NULL,
    "type" text NOT NULL,
    "is_partial" boolean NOT NULL,
    "title" text NOT NULL,
    "orig_title" text NOT NULL,
    "overview" text NOT NULL,
    "release_date" date,
    "is_adult" boolean NOT NULL,
    "backdrop" text NOT NULL,
    "poster" text NOT NULL,
    "popularity" real NOT NULL,
    "vote_average" real NOT NULL,
    "vote_count" int NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id", "type")
);

CREATE TABLE IF NOT EXISTS "public"."tmdb_item_genre" (
    "item_id" int NOT NULL,
    "item_type" text NOT NULL,
    "genre_id" int NOT NULL,
    PRIMARY KEY ("item_id", "item_type", "genre_id")
);

CREATE TABLE IF NOT EXISTS "public"."tmdb_list_item" (
    "list_id" text NOT NULL,
    "item_id" int NOT NULL,
    "item_type" text NOT NULL,
    "idx" int NOT NULL,
    PRIMARY KEY ("list_id", "item_id", "item_type")
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "public"."tmdb_list_item";
DROP TABLE IF EXISTS "public"."tmdb_item_genre";
DROP TABLE IF EXISTS "public"."tmdb_item";
DROP TABLE IF EXISTS "public"."tmdb_list";

-- +goose StatementEnd