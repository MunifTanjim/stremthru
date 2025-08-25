-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."imdb_title_map" ADD COLUMN "lboxd" text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."imdb_title_map" DROP COLUMN "lboxd";
-- +goose StatementEnd
