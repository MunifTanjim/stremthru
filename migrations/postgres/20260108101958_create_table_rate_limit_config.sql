-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."rate_limit_config" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "limit" int NOT NULL,
    "window" text NOT NULL,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."rate_limit_config";
-- +goose StatementEnd
