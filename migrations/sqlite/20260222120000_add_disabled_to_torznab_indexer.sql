-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  ADD COLUMN `disabled` bool NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  DROP COLUMN `disabled`;
-- +goose StatementEnd
