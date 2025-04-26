-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."dmm_hashlist" (
    "id" text NOT NULL,
    "entry_count" int NOT NULL,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."dmm_hashlist";
-- +goose StatementEnd
