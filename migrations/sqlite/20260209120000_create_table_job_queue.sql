-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS `job_queue` (
    `name` varchar NOT NULL,
    `key` varchar NOT NULL,
    `payload` jsonb,
    `status` varchar NOT NULL DEFAULT 'queued',
    `error` jsonb NOT NULL DEFAULT '[]',
    `priority` integer NOT NULL DEFAULT 0,
    `process_after` datetime NOT NULL DEFAULT (unixepoch()),
    `cat` datetime NOT NULL DEFAULT (unixepoch()),
    `uat` datetime NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (`name`, `key`)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `job_queue`;
-- +goose StatementEnd
