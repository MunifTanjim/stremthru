-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `torrent_info` (
  `hash` varchar NOT NULL,
  `t_title` varchar NOT NULL,

  `src` varchar NOT NULL,
  `category` varchar NOT NULL DEFAULT '',

  `created_at` datetime NOT NULL DEFAULT (unixepoch()),
  `updated_at` datetime NOT NULL DEFAULT (unixepoch()),
  `parsed_at` datetime,
  `parser_version` int NOT NULL DEFAULT 0,
  `sent_at` datetime,

  `audio` varchar NOT NULL DEFAULT '',
  `bit_depth` varchar NOT NULL DEFAULT '',
  `channels` varchar NOT NULL DEFAULT '',
  `codec` varchar NOT NULL DEFAULT '',
  `complete` bool NOT NULL DEFAULT false,
  `container` varchar NOT NULL DEFAULT '',
  `convert` bool NOT NULL DEFAULT false,
  `date` date NOT NULL DEFAULT '',
  `documentary` bool NOT NULL DEFAULT false,
  `dubbed` bool NOT NULL DEFAULT false,
  `edition` varchar NOT NULL DEFAULT '',
  `episode_code` varchar NOT NULL DEFAULT '',
  `episodes` varchar NOT NULL DEFAULT '',
  `extended` bool NOT NULL DEFAULT false,
  `extension` varchar NOT NULL DEFAULT '',
  `group` varchar NOT NULL DEFAULT '',
  `hdr` varchar NOT NULL DEFAULT '',
  `hardcoded` bool NOT NULL DEFAULT false,
  `languages` varchar NOT NULL DEFAULT '',
  `network` varchar NOT NULL DEFAULT '',
  `proper` bool NOT NULL DEFAULT false,
  `quality` varchar NOT NULL DEFAULT '',
  `region` varchar NOT NULL DEFAULT '',
  `remastered` bool NOT NULL DEFAULT false,
  `repack` bool NOT NULL DEFAULT false,
  `resolution` varchar NOT NULL DEFAULT '',
  `retail` bool NOT NULL DEFAULT false,
  `seasons` varchar NOT NULL DEFAULT '',
  `site` varchar NOT NULL DEFAULT '',
  `size` int NOT NULL DEFAULT -1,
  `subbed` bool NOT NULL DEFAULT false,
  `three_d` varchar NOT NULL DEFAULT '',
  `title` varchar NOT NULL DEFAULT '',
  `unrated` bool NOT NULL DEFAULT false,
  `upscaled` bool NOT NULL DEFAULT false,
  `volumes` varchar NOT NULl DEFAULT '',
  `year` int NOT NULL DEFAULT 0,
  `year_end` int NOT NULL DEFAULT 0,

  PRIMARY KEY (`hash`)
);

CREATE TABLE IF NOT EXISTS `torrent_stream` (
  `hash` varchar NOT NULL,
  `sid` varchar NOT NULL,
  `src` varchar NOT NULL,
  `f_idx` int NOT NULL,
  `f_name` varchar NOT NULL DEFAULT '',
  `f_size` int NOT NULL DEFAULT -1,
  `cat` datetime NOT NULL DEFAULT (unixepoch()),
  `uat` datetime NOT NULL DEFAULT (unixepoch()),

  PRIMARY KEY (`hash`, `sid`)
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `torrent_stream`;

DROP TABLE IF EXISTS `torrent_info`;
-- +goose StatementEnd
