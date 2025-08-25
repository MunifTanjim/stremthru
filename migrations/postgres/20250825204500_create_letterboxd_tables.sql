-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "public"."letterboxd_list" (
    "id" text NOT NULL,
    "user_id" text NOT NULL,
    "user_name" text NOT NULL,
    "name" text NOT NULL,
    "slug" text NOT NULL,
    "description" text NOT NULL,
    "private" boolean NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."letterboxd_item" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "release_year" int NOT NULL,
    "runtime" int NOT NULL,
    "rating" int NOT NULL,
    "adult" boolean NOT NULL,
    "poster" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."letterboxd_item_genre" (
    "item_id" text NOT NULL,
    "genre_id" text NOT NULL,
    PRIMARY KEY ("item_id", "genre_id")
);

CREATE TABLE IF NOT EXISTS "public"."letterboxd_list_item" (
    "list_id" text NOT NULL,
    "item_id" text NOT NULL,
    "rank" int NOT NULL,
    PRIMARY KEY ("list_id", "item_id")
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS "public"."letterboxd_list_item";
DROP TABLE IF EXISTS "public"."letterboxd_item_genre";
DROP TABLE IF EXISTS "public"."letterboxd_item";
DROP TABLE IF EXISTS "public"."letterboxd_list";

-- +goose StatementEnd
