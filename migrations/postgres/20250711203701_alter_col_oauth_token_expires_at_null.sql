-- +goose Up
-- +goose StatementBegin
ALTER TABLE "public"."oauth_token" ALTER COLUMN "expires_at" DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "public"."oauth_token"
  ALTER COLUMN "expires_at" TYPE timestamptz USING COALESCE("expires_at", 'epoch'),
  ALTER COLUMN "expires_at" SET NOT NULL;
-- +goose StatementEnd
