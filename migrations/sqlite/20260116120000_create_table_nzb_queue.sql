-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `nzb_queue` (
  `id` varchar NOT NULL,
  `name` varchar NOT NULL,
  `url` varchar NOT NULL,
  `category` varchar NOT NULL DEFAULT '',
  `priority` integer NOT NULL DEFAULT 0,
  `password` varchar NOT NULL DEFAULT '',
  `status` varchar NOT NULL,
  `error` varchar,
  `user` varchar NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`)
);
CREATE INDEX IF NOT EXISTS `nzb_queue_idx_status` ON `nzb_queue` (`status`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `nzb_queue_idx_status`;
DROP TABLE IF EXISTS `nzb_queue`;
-- +goose StatementEnd
