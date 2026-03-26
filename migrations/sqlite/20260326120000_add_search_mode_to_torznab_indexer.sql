-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  ADD COLUMN `search_mode` text NOT NULL DEFAULT 'auto';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  DROP COLUMN `search_mode`;
-- +goose StatementEnd
