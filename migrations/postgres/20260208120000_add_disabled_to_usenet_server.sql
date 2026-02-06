-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."usenet_server"
  ADD COLUMN "disabled" boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."usenet_server"
  DROP COLUMN IF EXISTS "disabled";
-- +goose StatementEnd
