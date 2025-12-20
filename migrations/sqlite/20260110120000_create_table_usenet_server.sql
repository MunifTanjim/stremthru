-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `usenet_server` (
  `id` varchar NOT NULL,
  `name` varchar NOT NULL,
  `host` varchar NOT NULL,
  `port` integer NOT NULL DEFAULT 563,
  `username` varchar NOT NULL,
  `password` varchar NOT NULL,
  `tls` boolean NOT NULL DEFAULT 1,
  `tls_skip_verify` boolean NOT NULL DEFAULT 0,
  `priority` integer NOT NULL DEFAULT 0,
  `is_backup` boolean NOT NULL DEFAULT 0,
  `max_conn` integer NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `usenet_server`;
-- +goose StatementEnd
