-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `trakt_account` (
  `id` varchar NOT NULL,
  `oauth_token_id` varchar NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`),
  UNIQUE (`oauth_token_id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `trakt_account`;
-- +goose StatementEnd
