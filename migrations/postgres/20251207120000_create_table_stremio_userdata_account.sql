-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."stremio_userdata_account" (
  "addon" text NOT NULL,
  "key" text NOT NULL,
  "account_id" text NOT NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("addon", "key", "account_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."stremio_userdata_account";
-- +goose StatementEnd
