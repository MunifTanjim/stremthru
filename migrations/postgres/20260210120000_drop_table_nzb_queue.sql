-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS "public"."nzb_queue_idx_status";
DROP TABLE IF EXISTS "public"."nzb_queue";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."nzb_queue" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "url" text NOT NULL,
  "category" text NOT NULL DEFAULT '',
  "priority" integer NOT NULL DEFAULT 0,
  "password" text NOT NULL DEFAULT '',
  "status" text NOT NULL,
  "error" text,
  "user" text NOT NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "nzb_queue_idx_status" ON "public"."nzb_queue" ("status");
-- +goose StatementEnd
