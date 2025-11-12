-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" ADD COLUMN "indexer" text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" DROP COLUMN "indexer";
-- +goose StatementEnd
