-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."job_queue" (
    "name" text NOT NULL,
    "key" text NOT NULL,
    "payload" jsonb,
    "status" text NOT NULL DEFAULT 'queued',
    "error" jsonb NOT NULL DEFAULT '[]',
    "priority" integer NOT NULL DEFAULT 0,
    "process_after" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("name", "key")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."job_queue";
-- +goose StatementEnd
