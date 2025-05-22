-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."imdb_title_map"
    ADD COLUMN "mal" text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS "public"."imdb_title_meta" (
    "tid" text NOT NULL,
    "description" text NOT NULL,
    "runtime" int NOT NULL,
    "poster" text NOT NULL,
    "backdrop" text NOT NULL,
    "trailer" text NOT NULL,
    "rating" int  NOT NULL,
    "mpa_rating" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("tid")
);

CREATE TABLE IF NOT EXISTS "public"."imdb_title_genre" (
    "tid" text NOT NULL,
    "genre" text NOT NULL,

    PRIMARY KEY ("tid", "genre")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."imdb_title_genre";

DROP TABLE IF EXISTS "public"."imdb_title_meta";

ALTER TABLE "public"."imdb_title_map"
    DROP COLUMN "mal";
-- +goose StatementEnd
