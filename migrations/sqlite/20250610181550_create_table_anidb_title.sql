-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `anidb_title` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `tid` varchar NOT NULL,
  `ttype` varchar NOT NULL,
  `tlang` varchar NOT NULL,
  `value` varchar NOT NULL,
  `season` varchar NOT NULL,
  `year` varchar NOT NULL,
  `type` varchar NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX `anidb_title_uidx_tid_ttype_tlang`
  ON `anidb_title` (`tid`, `ttype`, `tlang`);

CREATE VIRTUAL TABLE IF NOT EXISTS `anidb_title_fts` USING fts5(
  `value`, `season`, `year`, `type`, content='anidb_title', content_rowid='id'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `anidb_title_fts`;
DROP TABLE IF EXISTS `anidb_title`;
-- +goose StatementEnd
