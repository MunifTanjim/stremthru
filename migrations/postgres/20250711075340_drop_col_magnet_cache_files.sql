-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."magnet_cache" DROP COLUMN "files";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."magnet_cache" ADD COLUMN "files" json NOT NULL DEFAULT '[]';
-- +goose StatementEnd
