-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "public"."usenet_server" (
  "id" text NOT NULL,
  "name" text NOT NULL,
  "host" text NOT NULL,
  "port" integer NOT NULL DEFAULT 563,
  "username" text NOT NULL,
  "password" text NOT NULL,
  "tls" boolean NOT NULL DEFAULT true,
  "tls_skip_verify" boolean NOT NULL DEFAULT false,
  "priority" integer NOT NULL DEFAULT 0,
  "is_backup" boolean NOT NULL DEFAULT false,
  "max_conn" integer NOT NULL,
  "cat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "uat" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY ("id")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "public"."usenet_server";
-- +goose StatementEnd
