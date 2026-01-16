-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `newznab_indexer` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `type` varchar NOT NULL,
  `name` varchar NOT NULL,
  `url` varchar NOT NULL,
  `api_key` varchar NOT NULL,
  `rate_limit_config_id` varchar NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch())
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `newznab_indexer`;
-- +goose StatementEnd
