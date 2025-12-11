-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "sync_stremio_trakt_link" (
  "stremio_account_id" varchar NOT NULL,
  "trakt_account_id" varchar NOT NULL,
  "sync_config" jsonb NOT NULL DEFAULT '{"watched":{"dir":"none"}}',
  "sync_state" jsonb NOT NULL DEFAULT '{}',
  "cat" timestamp NOT NULL DEFAULT NOW(),
  "uat" timestamp NOT NULL DEFAULT NOW(),

  PRIMARY KEY ("stremio_account_id", "trakt_account_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "sync_stremio_trakt_link";
-- +goose StatementEnd
