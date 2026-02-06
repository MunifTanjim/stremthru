-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."newznab_indexer"
  ADD COLUMN "disabled" boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."newznab_indexer"
  DROP COLUMN IF EXISTS "disabled";
-- +goose StatementEnd
