-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `stremio_userdata_account` (
  `addon` varchar NOT NULL,
  `key` varchar NOT NULL,
  `account_id` varchar NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`addon`, `key`, `account_id`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `stremio_userdata_account`;
-- +goose StatementEnd
