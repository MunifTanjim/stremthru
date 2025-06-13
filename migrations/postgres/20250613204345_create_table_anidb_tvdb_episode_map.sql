-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."anidb_tvdb_episode_map" (
    "anidb_id" text NOT NULL,
    "tvdb_id" text NOT NULL,
    "anidb_season" int NOT NULL,
    "tvdb_season" int NOT NULL,
    "start" int NOT NULL,
    "end" int NOT NULL,
    "offset" int NOT NULL,
    "before" text NOT NULL,
    "map" text NOT NULL,

    PRIMARY KEY ("anidb_id", "tvdb_id", "anidb_season", "tvdb_season")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."anidb_tvdb_episode_map";
-- +goose StatementEnd
