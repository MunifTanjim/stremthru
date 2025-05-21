-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."mdblist_list" (
    "id" int NOT NULL,
    "user_id" int NOT NULL,
    "user_name" text NOT NULL,
    "name" text NOT NULL,
    "slug" text NOT NULL,
    "description" text NOT NULL DEFAULT '',
    "mediatype" text NOT NULL,
    "dynamic" boolean NOT NULL DEFAULT false,
    "private" boolean NOT NULL DEFAULT false,
    "likes" int NOT NULL DEFAULT 0,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."mdblist_item" (
    "id" int NOT NULL,
    "rank" int NOT NULL,
    "adult" boolean NOT NULL DEFAULT false,
    "title" text NOT NULL,
    "poster" text NOT NULL,
    "imdb_id" text NOT NULL,
    "tvdb_id" int NOT NULL,
    "language" text NOT NULL,
    "mediatype" text NOT NULL,
    "release_year" int NOT NULL,
    "spoken_language" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."mdblist_list_item" (
    "list_id" int NOT NULL,
    "item_id" int NOT NULL,

    PRIMARY KEY ("list_id", "item_id")
);

CREATE TABLE IF NOT EXISTS "public"."mdblist_item_genre" (
    "item_id" int NOT NULL,
    "genre" text NOT NULL,

    PRIMARY KEY ("item_id", "genre")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."mdblist_item_genre";
DROP TABLE IF EXISTS "public"."mdblist_list_item";
DROP TABLE IF EXISTS "public"."mdblist_item";
DROP TABLE IF EXISTS "public"."mdblist_list";
-- +goose StatementEnd
