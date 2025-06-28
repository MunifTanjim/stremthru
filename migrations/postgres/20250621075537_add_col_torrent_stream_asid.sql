-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_stream" ADD COLUMN "asid" text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torrent_stream" DROP COLUMN "asid";
-- +goose StatementEnd
