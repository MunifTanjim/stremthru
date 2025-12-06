-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."stremio_account" (
  "id" text NOT NULL,
  "email" text NOT NULL,
  "password" text NOT NULL,
  "token" text NOT NULL DEFAULT '',
  "token_eat" timestamptz,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("id"),
  UNIQUE ("email")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."stremio_account";
-- +goose StatementEnd
