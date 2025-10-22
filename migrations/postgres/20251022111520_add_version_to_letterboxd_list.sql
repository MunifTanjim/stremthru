-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."letterboxd_list" ADD COLUMN "v" bigint NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."letterboxd_list" DROP COLUMN "v";
-- +goose StatementEnd
