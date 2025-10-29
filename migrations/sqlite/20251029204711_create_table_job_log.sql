-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `job_log` (
    `name` varchar NOT NULL,
    `id` varchar NOT NULL,
    `status` varchar NOT NULL,
    `data` json,
    `error` varchar,
    `created_at` datetime NOT NULL DEFAULT (unixepoch()),
    `updated_at` datetime NOT NULL DEFAULT (unixepoch()),
    `expires_at` datetime,
    PRIMARY KEY (`name`, `id`)
);

INSERT INTO job_log (name, id, status, data, error, created_at, updated_at, expires_at)
SELECT substring(t, 5) AS name,
       k               AS id,
       v ->> 'status'  AS status,
       null            AS data,
       v ->> 'err'     AS error,
       cat             AS created_at,
       uat             AS updated_at,
       eat             AS expires_at
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
SELECT concat('job:', name)                                      AS t,
       id                                                        AS k,
       json_object('status', status, 'err', error, 'data', data) AS v,
       created_at                                                AS cat,
       updated_at                                                AS uat,
       expires_at                                                AS eat
FROM job_log

DROP TABLE IF EXISTS `job_log`;

-- +goose StatementEnd
