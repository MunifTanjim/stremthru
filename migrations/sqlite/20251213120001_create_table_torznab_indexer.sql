-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torznab_indexer` (
  `type` varchar NOT NULL,
  `id` varchar NOT NULL,
  `name` varchar NOT NULL,
  `url` varchar NOT NULL,
  `api_key` varchar NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`type`, `id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `torznab_indexer`;
-- +goose StatementEnd
