-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."serializd_list" (
    "id" text NOT NULL,
    "name" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."serializd_item" (
    "id" int NOT NULL,
    "name" text NOT NULL,
    "banner_image" text NOT NULL,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "public"."serializd_list_item" (
    "list_id" text NOT NULL,
    "item_id" int NOT NULL,
    "idx" int NOT NULL,
    PRIMARY KEY ("list_id", "item_id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."serializd_list_item";
DROP TABLE IF EXISTS "public"."serializd_item";
DROP TABLE IF EXISTS "public"."serializd_list";
-- +goose StatementEnd
