-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  ADD COLUMN "rate_limit_config_id" text NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  DROP COLUMN IF EXISTS "rate_limit_config_id";
-- +goose StatementEnd
