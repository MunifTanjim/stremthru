-- +goose Up
-- +goose StatementBegin
ALTER TABLE `oauth_token` RENAME TO `oauth_token_old`;

CREATE TABLE IF NOT EXISTS `oauth_token` (
  `id` varchar NOT NULL,
  `provider` varchar NOT NULL,
  `user_id` varchar NOT NULL,
  `user_name` varchar NOT NULL,
  `token_type` varchar NOT NULL,
  `access_token` varchar NOT NULL,
  `refresh_token` varchar NOT NULL,
  `expires_at` datetime,
  `scope` varchar NOT NULL,
  `v` int NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`),
  UNIQUE (`provider`, `user_id`)
);

INSERT INTO `oauth_token` (`id`, `provider`, `user_id`, `user_name`, `token_type`, `access_token`, `refresh_token`, `expires_at`, `scope`, `v`, `cat`, `uat`)
SELECT `id`, `provider`, `user_id`, `user_name`, `token_type`, `access_token`, `refresh_token`, `expires_at`, `scope`, `v`, `cat`, `uat` FROM `oauth_token_old`;

DROP TABLE `oauth_token_old`;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `oauth_token` RENAME TO `oauth_token_old`;

CREATE TABLE IF NOT EXISTS `oauth_token` (
  `id` varchar NOT NULL,
  `provider` varchar NOT NULL,
  `user_id` varchar NOT NULL,
  `user_name` varchar NOT NULL,
  `token_type` varchar NOT NULL,
  `access_token` varchar NOT NULL,
  `refresh_token` varchar NOT NULL,
  `expires_at` datetime NOT NULL,
  `scope` varchar NOT NULL,
  `v` int NOT NULL,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`id`),
  UNIQUE (`provider`, `user_id`)
);

INSERT INTO `oauth_token` (`id`, `provider`, `user_id`, `user_name`, `token_type`, `access_token`, `refresh_token`, `expires_at`, `scope`, `v`, `cat`, `uat`)
SELECT `id`, `provider`, `user_id`, `user_name`, `token_type`, `access_token`, `refresh_token`, COALESCE(`expires_at`, 0), `scope`, `v`, `cat`, `uat` FROM `oauth_token_old`;

DROP TABLE `oauth_token_old`;
-- +goose StatementEnd
