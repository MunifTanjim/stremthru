-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "sync_stremio_trakt_link" (
  "stremio_account_id" varchar NOT NULL,
  "trakt_account_id" varchar NOT NULL,
  "sync_config" json NOT NULL DEFAULT '{"watched":{"dir":"none"}}',
  "sync_state" json NOT NULL DEFAULT '{}',
  "cat" datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("stremio_account_id", "trakt_account_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "sync_stremio_trakt_link";
-- +goose StatementEnd
