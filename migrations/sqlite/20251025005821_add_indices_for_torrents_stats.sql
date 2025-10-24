-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS `torrent_info_idx_src` ON `torrent_info` (`src`);
CREATE INDEX IF NOT EXISTS `torrent_stream_idx_sid_src` ON `torrent_stream` (`sid`, `src`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `torrent_stream_idx_sid_src`;
DROP INDEX IF EXISTS `torrent_info_idx_src`;
-- +goose StatementEnd
