-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."imdb_title" (
  "id" serial NOT NULL PRIMARY KEY,
  "tid" text NOT NULL,
  "type" text NOT NULL,
  "title" text NOT NULL,
  "orig_title" text NOT NULL,
  "year" int NOT NULL,
  "is_adult" boolean NOT NULL
);

CREATE UNIQUE INDEX "imdb_title_uidx_tid"
  ON "public"."imdb_title" ("tid");

ALTER TABLE "public"."imdb_title" ADD COLUMN "search_vector" tsvector;

CREATE INDEX "imdb_title_idx_search_vector" ON "public"."imdb_title" USING GIN ("search_vector");

CREATE OR REPLACE FUNCTION imdb_title_update_search_vector() RETURNS trigger AS $$
BEGIN
  NEW."search_vector" := to_tsvector(
    'english',
    coalesce(NEW."title", '') || ' ' ||
    coalesce(NEW."orig_title", '') || ' ' ||
    coalesce(NEW."type", '') || ' ' ||
    NEW."year"::text
  );
  RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER "imdb_title_trigger_on_upsert"
  BEFORE INSERT OR UPDATE
  ON "public"."imdb_title"
FOR EACH ROW EXECUTE FUNCTION imdb_title_update_search_vector();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."imdb_title";
-- +goose StatementEnd
