-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "trakt_account" (
  "id" varchar NOT NULL,
  "oauth_token_id" varchar NOT NULL,
  "cat" timestamp NOT NULL DEFAULT NOW(),
  "uat" timestamp NOT NULL DEFAULT NOW(),

  PRIMARY KEY ("id"),
  UNIQUE ("oauth_token_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "trakt_account";
-- +goose StatementEnd
