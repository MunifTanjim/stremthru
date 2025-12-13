-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."torznab_indexer" (
  "type" text NOT NULL,
  "id" text NOT NULL,
  "name" text NOT NULL,
  "url" text NOT NULL,
  "api_key" text NOT NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("type", "id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."torznab_indexer";
-- +goose StatementEnd
