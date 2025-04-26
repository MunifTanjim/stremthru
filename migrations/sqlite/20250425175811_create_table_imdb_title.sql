-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `imdb_title` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `tid` varchar NOT NULL,
  `type` varchar NOT NULL,
  `title` varchar NOT NULL,
  `orig_title` varchar NOT NULL,
  `year` int NOT NULL,
  `is_adult` bool NOT NULL
);

CREATE UNIQUE INDEX `imdb_title_uidx_tid`
  ON `imdb_title` (`tid`);

CREATE VIRTUAL TABLE IF NOT EXISTS `imdb_title_fts` USING fts5(
  `title`, `orig_title`, `year`, `type`, content='imdb_title', content_rowid='id'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `imdb_title_fts`;
DROP TABLE IF EXISTS `imdb_title`;
-- +goose StatementEnd
