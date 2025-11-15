-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "public"."job_log" (
    "name" text NOT NULL,
    "id" text NOT NULL,
    "status" text NOT NULL,
    "data" jsonb,
    "error" text,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expires_at" timestamptz,
    PRIMARY KEY ("name", "id")
);

INSERT INTO job_log (name, id, status, data, error, created_at, updated_at, expires_at)
SELECT ltrim(t, 'job:')     AS name,
       k                    AS id,
       v::json ->> 'status' AS status,
       null                 AS data,
       v::json ->> 'err'    AS error,
       cat                  AS created_at,
       uat                  AS updated_at,
       eat                  AS expires_at
FROM kv
WHERE t LIKE 'job:%'
  AND t NOT LIKE 'job:%:%';

DELETE
FROM kv
WHERE t LIKE 'job:%'
  AND t NOT LIKE 'job:%:%';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

INSERT INTO kv (t, k, v, cat, uat, eat)
SELECT concat('job:', name)                                            AS t,
       id                                                              AS k,
       json_build_object('status', status, 'err', error, 'data', data) AS v,
       created_at                                                      AS cat,
       updated_at                                                      AS uat,
       expires_at                                                      AS eat
FROM job_log;

DROP TABLE IF EXISTS "public"."job_log";

-- +goose StatementEnd
