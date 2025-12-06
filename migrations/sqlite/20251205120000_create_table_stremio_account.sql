-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `stremio_account` (
  `id` varchar NOT NULL,
  `email` varchar NOT NULL,
  `password` varchar NOT NULL,
  `token` varchar NOT NULL DEFAULT '',
  `token_eat` datetime,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`),
  UNIQUE (`email`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `stremio_account`;
-- +goose StatementEnd
