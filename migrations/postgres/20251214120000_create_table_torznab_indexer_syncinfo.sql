-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer_syncinfo" (
  "type" text NOT NULL,
  "id" text NOT NULL,
  "sid" text NOT NULL,
  "queued_at" timestamptz,
  "synced_at" timestamptz,
  "error" text,
  "result_count" integer,
  PRIMARY KEY ("type", "id", "sid")
);

CREATE INDEX "torznab_indexer_syncinfo_idx_queued_at_synced_at" ON "public"."torznab_indexer_syncinfo" ("queued_at", "synced_at");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_queued_at_synced_at";
DROP TABLE IF EXISTS "public"."torznab_indexer_syncinfo";
-- +goose StatementEnd
