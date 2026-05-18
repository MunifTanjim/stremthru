-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."torrent_mapping_review" ADD COLUMN "status" text NOT NULL DEFAULT 'pending';
ALTER TABLE "public"."torrent_mapping_review" ADD COLUMN "resolved_at" timestamptz;
CREATE INDEX torrent_mapping_review_idx_status ON "public"."torrent_mapping_review" ("status");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS torrent_mapping_review_idx_status;
ALTER TABLE "public"."torrent_mapping_review" DROP COLUMN IF EXISTS "resolved_at";
ALTER TABLE "public"."torrent_mapping_review" DROP COLUMN IF EXISTS "status";
-- +goose StatementEnd
