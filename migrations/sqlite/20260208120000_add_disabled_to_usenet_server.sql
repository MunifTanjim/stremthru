-- +goose Up
-- +goose StatementBegin
ALTER TABLE `usenet_server`
  ADD COLUMN `disabled` bool NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `usenet_server`
  DROP COLUMN `disabled`;
-- +goose StatementEnd
