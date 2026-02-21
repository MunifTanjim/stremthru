-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  ADD COLUMN "disabled" boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  DROP COLUMN IF EXISTS "disabled";
-- +goose StatementEnd
