-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_status_queued_at";

DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_indexer_id_status_queued_at";

CREATE INDEX "torznab_indexer_syncinfo_idx_status_queued_at"
  ON "public"."torznab_indexer_syncinfo" ("queued_at" DESC)
  WHERE "status" IN ('queued', 'syncing');

CREATE INDEX "torznab_indexer_syncinfo_idx_indexer_id_status_queued_at"
  ON "public"."torznab_indexer_syncinfo" ("indexer_id", (("synced_at" IS NULL)) DESC, "queued_at" DESC)
  WHERE "status" IN ('queued', 'syncing');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_indexer_id_status_queued_at";

DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_status_queued_at";

CREATE INDEX "torznab_indexer_syncinfo_idx_status_queued_at"
  ON "public"."torznab_indexer_syncinfo" ("status", "queued_at" DESC);

CREATE INDEX "torznab_indexer_syncinfo_idx_indexer_id_status_queued_at"
  ON "public"."torznab_indexer_syncinfo" ("indexer_id", "status", "queued_at" DESC);
-- +goose StatementEnd
