-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."trakt_list_item" ADD COLUMN "added_at" timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."trakt_list_item" DROP COLUMN "added_at";
-- +goose StatementEnd
