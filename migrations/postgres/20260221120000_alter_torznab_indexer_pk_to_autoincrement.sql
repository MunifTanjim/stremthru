-- +goose Up
-- +goose StatementBegin

-- 1. Rename old tables
ALTER TABLE "public"."torznab_indexer" RENAME TO "_torznab_indexer_old";
ALTER TABLE "public"."torznab_indexer_syncinfo" RENAME TO "_torznab_indexer_syncinfo_old";

DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_queued_at_synced_at";
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_status";

-- 2. Create new torznab_indexer with serial PK
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer" (
  "id" serial NOT NULL PRIMARY KEY,
  "type" text NOT NULL,
  "old_id" text NULL,
  "name" text NOT NULL,
  "url" text NOT NULL,
  "api_key" text NOT NULL,
  "rate_limit_config_id" text NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Migrate data
INSERT INTO "public"."torznab_indexer" ("type", "old_id", "name", "url", "api_key", "rate_limit_config_id", "cat", "uat")
  SELECT "type", "id", "name", "url", "api_key", "rate_limit_config_id", "cat", "uat"
  FROM "public"."_torznab_indexer_old";

-- 4. Create new syncinfo table
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer_syncinfo" (
  "indexer_id" integer NOT NULL,
  "sid" text NOT NULL,
  "queued_at" timestamptz,
  "synced_at" timestamptz,
  "error" text,
  "result_count" integer,
  "status" text NOT NULL DEFAULT 'queued',
  "queries" jsonb,
  PRIMARY KEY ("indexer_id", "sid")
);

CREATE INDEX "torznab_indexer_syncinfo_idx_queued_at_synced_at"
  ON "public"."torznab_indexer_syncinfo" ("queued_at", "synced_at");
CREATE INDEX "torznab_indexer_syncinfo_idx_status"
  ON "public"."torznab_indexer_syncinfo" ("status");

-- 5. Migrate syncinfo data
INSERT INTO "public"."torznab_indexer_syncinfo" ("indexer_id", "sid", "queued_at", "synced_at", "error", "result_count", "status", "queries")
  SELECT ti."id", osi."sid", osi."queued_at", osi."synced_at", osi."error", osi."result_count", osi."status", osi."queries"
  FROM "public"."_torznab_indexer_syncinfo_old" osi
  INNER JOIN "public"."torznab_indexer" ti ON osi."type" = ti."type" AND osi."id" = ti."old_id";

-- 6. Drop old tables
DROP TABLE "public"."_torznab_indexer_syncinfo_old";
DROP TABLE "public"."_torznab_indexer_old";

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- 1. Rename new tables
ALTER TABLE "public"."torznab_indexer" RENAME TO "_torznab_indexer_new";
ALTER TABLE "public"."torznab_indexer_syncinfo" RENAME TO "_torznab_indexer_syncinfo_new";

DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_queued_at_synced_at";
DROP INDEX IF EXISTS "public"."torznab_indexer_syncinfo_idx_status";

-- 2. Recreate original torznab_indexer with composite PK
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer" (
  "type" text NOT NULL,
  "id" text NOT NULL,
  "name" text NOT NULL,
  "url" text NOT NULL,
  "api_key" text NOT NULL,
  "rate_limit_config_id" text NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("type", "id")
);

-- 3. Migrate data back using old_id
INSERT INTO "public"."torznab_indexer" ("type", "id", "name", "url", "api_key", "rate_limit_config_id", "cat", "uat")
  SELECT "type", "old_id", "name", "url", "api_key", "rate_limit_config_id", "cat", "uat"
  FROM "public"."_torznab_indexer_new"
  WHERE "old_id" IS NOT NULL;

-- 4. Recreate original torznab_indexer_syncinfo with composite PK
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer_syncinfo" (
  "type" text NOT NULL,
  "id" text NOT NULL,
  "sid" text NOT NULL,
  "queued_at" timestamptz,
  "synced_at" timestamptz,
  "error" text,
  "result_count" integer,
  "status" text NOT NULL DEFAULT 'queued',
  "queries" jsonb,
  PRIMARY KEY ("type", "id", "sid")
);

CREATE INDEX "torznab_indexer_syncinfo_idx_queued_at_synced_at"
  ON "public"."torznab_indexer_syncinfo" ("queued_at", "synced_at");
CREATE INDEX "torznab_indexer_syncinfo_idx_status"
  ON "public"."torznab_indexer_syncinfo" ("status");

-- 5. Migrate syncinfo data back, joining to map integer id to original (type, old_id)
INSERT INTO "public"."torznab_indexer_syncinfo" ("type", "id", "sid", "queued_at", "synced_at", "error", "result_count", "status", "queries")
  SELECT ti."type", ti."old_id", osi."sid", osi."queued_at", osi."synced_at", osi."error", osi."result_count", osi."status", osi."queries"
  FROM "public"."_torznab_indexer_syncinfo_new" osi
  INNER JOIN "public"."_torznab_indexer_new" ti ON osi."indexer_id" = ti."id"
  WHERE ti."old_id" IS NOT NULL;

-- 6. Drop new tables and sequence
DROP TABLE "public"."_torznab_indexer_syncinfo_new";
DROP TABLE "public"."_torznab_indexer_new";

-- +goose StatementEnd
