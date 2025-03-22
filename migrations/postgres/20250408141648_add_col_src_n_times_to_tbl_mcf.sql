-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."magnet_cache_file"
  ADD COLUMN "src" text NOT NULL DEFAULT '',
  ADD COLUMN "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN "sat" timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
