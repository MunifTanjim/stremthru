-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_stream" ADD COLUMN "mi" jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torrent_stream" DROP COLUMN "mi";
-- +goose StatementEnd
