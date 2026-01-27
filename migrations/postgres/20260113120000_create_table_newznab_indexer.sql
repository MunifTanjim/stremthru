-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."newznab_indexer" (
  "id" serial NOT NULL PRIMARY KEY,
  "type" text NOT NULL,
  "name" text NOT NULL,
  "url" text NOT NULL,
  "api_key" text NOT NULL,
  "rate_limit_config_id" text NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."newznab_indexer";
-- +goose StatementEnd
