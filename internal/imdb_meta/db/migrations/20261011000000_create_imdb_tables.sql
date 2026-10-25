-- +goose Up
CREATE TABLE IF NOT EXISTS title (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL DEFAULT '',
  is_series INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL,
  orig_name TEXT NOT NULL DEFAULT '',
  year_start INTEGER NOT NULL DEFAULT 0,
  year_end INTEGER,
  release_date TEXT NOT NULL DEFAULT '',
  release_country TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  runtime INTEGER NOT NULL DEFAULT 0,
  image_id TEXT NOT NULL DEFAULT '',
  rating INTEGER NOT NULL DEFAULT 0,
  votes INTEGER NOT NULL DEFAULT 0,
  certificate TEXT NOT NULL DEFAULT '',
  -- 1 for episodes stored from a series payload, 0 once the title itself has
  -- been fetched. uat only moves on a full upsert, never on a partial one.
  partial INTEGER NOT NULL DEFAULT 0,
  uat INTEGER NOT NULL DEFAULT (unixepoch()),
  -- Last refresh of the fast-changing part (ratings, the episode list); 0 if
  -- never. A full upsert moves it too.
  fast_uat INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS title_idx_image_id ON title(image_id);

CREATE TABLE IF NOT EXISTS title_genre (
  id TEXT NOT NULL,
  genre TEXT NOT NULL,
  PRIMARY KEY (id, genre)
);

CREATE TABLE IF NOT EXISTS title_trailer (
  id TEXT NOT NULL,
  vi_id TEXT NOT NULL,
  pos INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (id, vi_id)
);

CREATE TABLE IF NOT EXISTS title_aka (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title_id TEXT NOT NULL,
  text TEXT NOT NULL,
  country TEXT NOT NULL DEFAULT '',
  language TEXT NOT NULL DEFAULT '',
  UNIQUE (title_id, text, country, language)
);
CREATE INDEX IF NOT EXISTS title_aka_idx_title_id ON title_aka(title_id);

CREATE TABLE IF NOT EXISTS title_person (
  id TEXT NOT NULL,
  role TEXT NOT NULL,
  pos INTEGER NOT NULL DEFAULT 0,
  nmid TEXT NOT NULL,
  name TEXT NOT NULL,
  character_name TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (id, role, pos)
);

CREATE TABLE IF NOT EXISTS title_episode (
  id TEXT PRIMARY KEY,
  title_id TEXT NOT NULL,
  season INTEGER NOT NULL,
  -- NULL when IMDb gives the episode no number.
  episode INTEGER
);
CREATE INDEX IF NOT EXISTS title_episode_idx_title_id_season_episode ON title_episode(title_id, season, episode);

CREATE TABLE IF NOT EXISTS missing (
  id TEXT PRIMARY KEY,
  uat INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS missing_idx_uat ON missing(uat);

-- Image data is shared: IMDb reuses an image across titles (a series and its
-- episodes). title_image lists which images a title shows, in order.
CREATE TABLE IF NOT EXISTS image (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0,
  caption TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS title_image (
  title_id TEXT NOT NULL,
  image_id TEXT NOT NULL,
  ep_id TEXT NOT NULL DEFAULT '',
  pos INTEGER NOT NULL,
  PRIMARY KEY (title_id, image_id)
);
CREATE INDEX IF NOT EXISTS title_image_idx_title_id_pos ON title_image(title_id, pos);
CREATE INDEX IF NOT EXISTS title_image_idx_image_id ON title_image(image_id);

-- Set once a title's image gallery has been fully synced.
CREATE TABLE IF NOT EXISTS title_gallery (
  title_id TEXT PRIMARY KEY,
  uat INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down
DROP TABLE IF EXISTS title_gallery;
DROP TABLE IF EXISTS title_image;
DROP TABLE IF EXISTS image;
DROP TABLE IF EXISTS missing;
DROP TABLE IF EXISTS title_episode;
DROP TABLE IF EXISTS title_person;
DROP TABLE IF EXISTS title_aka;
DROP TABLE IF EXISTS title_trailer;
DROP TABLE IF EXISTS title_genre;
DROP TABLE IF EXISTS title;
