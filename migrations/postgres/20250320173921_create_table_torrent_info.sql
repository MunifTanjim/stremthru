-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."torrent_info" (
  "hash" text NOT NULL,
  "t_title" text NOT NULL,

  "src" text NOT NULL,
  "category" text NOT NULL DEFAULT '',

  "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "parsed_at" timestamptz,
  "parser_version" int NOT NULL DEFAULT 0,
  "sent_at" timestamptz,

  "audio" text NOT NULL DEFAULT '',
  "bit_depth" text NOT NULL DEFAULT '',
  "channels" text NOT NULL DEFAULT '',
  "codec" text NOT NULL DEFAULT '',
  "complete" boolean NOT NULL DEFAULT false,
  "container" text NOT NULL DEFAULT '',
  "convert" boolean NOT NULL DEFAULT false,
  "date" date NOT NULL DEFAULT '',
  "documentary" boolean NOT NULL DEFAULT false,
  "dubbed" boolean NOT NULL DEFAULT false,
  "edition" text NOT NULL DEFAULT '',
  "episode_code" text NOT NULL DEFAULT '',
  "episodes" text NOT NULL DEFAULT '',
  "extended" boolean NOT NULL DEFAULT false,
  "extension" text NOT NULL DEFAULT '',
  "group" text NOT NULL DEFAULT '',
  "hdr" text NOT NULL DEFAULT '',
  "hardcoded" boolean NOT NULL DEFAULT false,
  "languages" text NOT NULL DEFAULT '',
  "network" text NOT NULL DEFAULT '',
  "proper" boolean NOT NULL DEFAULT false,
  "quality" text NOT NULL DEFAULT '',
  "region" text NOT NULL DEFAULT '',
  "remastered" boolean NOT NULL DEFAULT false,
  "repack" boolean NOT NULL DEFAULT false,
  "resolution" text NOT NULL DEFAULT '',
  "retail" boolean NOT NULL DEFAULT false,
  "seasons" text NOT NULL DEFAULT '',
  "site" text NOT NULL DEFAULT '',
  "size" bigint NOT NULL DEFAULT -1,
  "subbed" boolean NOT NULL DEFAULT false,
  "three_d" text NOT NULL DEFAULT '',
  "title" text NOT NULL DEFAULT '',
  "unrated" boolean NOT NULL DEFAULT false,
  "upscaled" boolean NOT NULL DEFAULT false,
  "volumes" text NOT NULl DEFAULT '',
  "year" int NOT NULL DEFAULT 0,
  "year_end" int NOT NULL DEFAULT 0,

  PRIMARY KEY ("hash")
);

CREATE TABLE IF NOT EXISTS "public"."torrent_stream" (
  "hash" text NOT NULL,
  "sid" text NOT NULL,
  "src" text NOT NULL,
  "f_idx" int NOT NULL,
  "f_name" varchar NOT NULL DEFAULT '',
  "f_size" int NOT NULL DEFAULT -1,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("hash", "sid")
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."torrent_stream";

DROP TABLE IF EXISTS "public"."torrent_info";
-- +goose StatementEnd
