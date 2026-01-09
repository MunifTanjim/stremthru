-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  ADD COLUMN `rate_limit_config_id` varchar NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  DROP COLUMN `rate_limit_config_id`;
-- +goose StatementEnd
