-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  ADD COLUMN "only_anime" boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."torznab_indexer"
  DROP COLUMN IF EXISTS "only_anime";
-- +goose StatementEnd
