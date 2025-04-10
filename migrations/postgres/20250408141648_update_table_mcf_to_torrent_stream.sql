-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."magnet_cache_file"
  RENAME TO "torrent_stream";

ALTER TABLE "public"."torrent_stream"
  ADD COLUMN "src" text NOT NULL DEFAULT '',
  ADD COLUMN "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
