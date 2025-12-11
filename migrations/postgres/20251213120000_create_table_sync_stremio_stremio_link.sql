-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."sync_stremio_stremio_link" (
  "account_a_id" varchar NOT NULL,
  "account_b_id" varchar NOT NULL,
  "sync_config" jsonb NOT NULL DEFAULT '{"watched":{"dir":"none","ids":[]}}',
  "sync_state" jsonb NOT NULL DEFAULT '{}',
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("account_a_id", "account_b_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."sync_stremio_stremio_link";
-- +goose StatementEnd
