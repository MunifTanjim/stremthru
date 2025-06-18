-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."anidb_title" (
  "id" serial NOT NULL PRIMARY KEY,
  "tid" text NOT NULL,
  "ttype" text NOT NULL,
  "tlang" text NOT NULL,
  "value" text NOT NULL,
  "season" text NOT NULL,
  "year" text NOT NULL,
  "type" text NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX "anidb_title_uidx_tid_ttype_tlang"
  ON "public"."anidb_title" ("tid", "ttype", "tlang");

ALTER TABLE "public"."anidb_title" ADD COLUMN "search_vector" tsvector;

CREATE INDEX "anidb_title_idx_search_vector" ON "public"."anidb_title" USING GIN ("search_vector");

CREATE OR REPLACE FUNCTION anidb_title_update_search_vector() RETURNS trigger AS $$
BEGIN
  NEW."search_vector" := to_tsvector(
    'simple',
    coalesce(NEW."value", '') || ' ' ||
    NEW."year"
  );
  RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

CREATE TRIGGER "anidb_title_trigger_on_upsert"
  BEFORE INSERT OR UPDATE
  ON "public"."anidb_title"
FOR EACH ROW EXECUTE FUNCTION anidb_title_update_search_vector();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."anidb_title";
-- +goose StatementEnd
