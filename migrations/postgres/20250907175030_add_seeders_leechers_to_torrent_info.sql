-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" ADD COLUMN "seeders" int NOT NULL DEFAULT 0;
ALTER TABLE "public"."torrent_info" ADD COLUMN "leechers" int NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torrent_info" DROP COLUMN "leechers";
ALTER TABLE "public"."torrent_info" DROP COLUMN "seeders";
-- +goose StatementEnd
