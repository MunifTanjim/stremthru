-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."tvdb_list" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "slug" text NOT NULL,
    "overview" text NOT NULL,
    "is_official" boolean NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."tvdb_item" (
    "id" int NOT NULL,
    "type" text NOT NULL,
    "name" text NOT NULL,
    "overview" text NOT NULL,
    "year" int NOT NULL,
    "runtime" int NOT NULL,
    "poster" text NOT NULL,
    "background" text NOT NULL,
    "trailer" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id", "type")
);

CREATE TABLE IF NOT EXISTS "public"."tvdb_item_genre" (
    "item_id" int NOT NULL,
    "item_type" text NOT NULL,
    "genre" int NOT NULL,
    PRIMARY KEY ("item_id", "item_type", "genre")
);

CREATE TABLE IF NOT EXISTS "public"."tvdb_list_item" (
    "list_id" text NOT NULL,
    "item_id" int NOT NULL,
    "item_type" text NOT NULL,
    "order" int NOT NULL,
    PRIMARY KEY ("list_id", "item_id", "item_type")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."tvdb_list_item";
DROP TABLE IF EXISTS "public"."tvdb_item_genre";
DROP TABLE IF EXISTS "public"."tvdb_item";
DROP TABLE IF EXISTS "public"."tvdb_list";
-- +goose StatementEnd
