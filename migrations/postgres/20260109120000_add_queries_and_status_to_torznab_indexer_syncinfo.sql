-- +goose Up
-- +goose StatementBegin
DELETE FROM "public"."torznab_indexer_syncinfo";
ALTER TABLE "public"."torznab_indexer_syncinfo" ADD COLUMN "status" text NOT NULL DEFAULT 'queued';
ALTER TABLE "public"."torznab_indexer_syncinfo" ADD COLUMN "queries" jsonb;
CREATE INDEX "torznab_indexer_syncinfo_idx_status" ON "public"."torznab_indexer_syncinfo" ("status");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_status";
ALTER TABLE "public"."torznab_indexer_syncinfo" DROP COLUMN "queries";
ALTER TABLE "public"."torznab_indexer_syncinfo" DROP COLUMN "status";
-- +goose StatementEnd
