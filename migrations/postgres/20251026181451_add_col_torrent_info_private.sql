-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" ADD COLUMN "private" boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" DROP COLUMN "private";
-- +goose StatementEnd
