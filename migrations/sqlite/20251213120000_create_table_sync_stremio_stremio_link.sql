-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `sync_stremio_stremio_link` (
  `account_a_id` varchar NOT NULL,
  `account_b_id` varchar NOT NULL,
  `sync_config` json NOT NULL DEFAULT '{"watched":{"dir":"none","ids":[]}}',
  `sync_state` json NOT NULL DEFAULT '{}',
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`account_a_id`, `account_b_id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `sync_stremio_stremio_link`;
-- +goose StatementEnd
