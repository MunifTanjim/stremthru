-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."kv" ADD COLUMN "eat" timestamptz;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."kv" DROP COLUMN "eat" timestamptz;
-- +goose StatementEnd
