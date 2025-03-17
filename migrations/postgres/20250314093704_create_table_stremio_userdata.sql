-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."stremio_userdata" (
    "addon" text NOT NULL,
    "key" text NOT NULL,
    "value" json NOT NULL,
    "name" text NOT NULL,
    "disabled" boolean NOT NULL DEFAULT false,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("addon", "key")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."stremio_userdata";
-- +goose StatementEnd
