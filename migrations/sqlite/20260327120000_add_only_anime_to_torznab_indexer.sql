-- +goose Up
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  ADD COLUMN `only_anime` bool NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `torznab_indexer`
  DROP COLUMN `only_anime`;
-- +goose StatementEnd
