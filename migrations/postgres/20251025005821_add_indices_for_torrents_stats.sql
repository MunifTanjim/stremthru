-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS "torrent_info_idx_src" ON "torrent_info" ("src");
CREATE INDEX IF NOT EXISTS "torrent_stream_idx_src_sid" ON "torrent_stream" ("src", "sid");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS "torrent_stream_idx_src_sid";
DROP INDEX IF EXISTS "torrent_info_idx_src";
-- +goose StatementEnd
