-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS "anime_id_map_idx_imdb" ON "public"."anime_id_map" ("imdb");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."anime_id_map_idx_imdb";
-- +goose StatementEnd
