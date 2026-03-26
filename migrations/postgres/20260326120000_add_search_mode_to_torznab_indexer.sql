-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  ADD COLUMN "search_mode" text NOT NULL DEFAULT 'auto';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  DROP COLUMN IF EXISTS "search_mode";
-- +goose StatementEnd
