-- +goose Up
-- +goose StatementBegin
ALTER TABLE `anime_id_map` ADD COLUMN `lboxd` varchar;
ALTER TABLE `anime_id_map` ADD COLUMN `tmdb_season_id` int NOT NULL DEFAULT 0;
ALTER TABLE `anime_id_map` ADD COLUMN `tvdb_season_id` int NOT NULL DEFAULT 0;
ALTER TABLE `anime_id_map` ADD COLUMN `trakt` varchar;
ALTER TABLE `anime_id_map` ADD COLUMN `trakt_season` int NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE `anime_id_map` DROP COLUMN `trakt_season`;
ALTER TABLE `anime_id_map` DROP COLUMN `trakt`;
ALTER TABLE `anime_id_map` DROP COLUMN `tvdb_season_id`;
ALTER TABLE `anime_id_map` DROP COLUMN `tmdb_season_id`;
ALTER TABLE `anime_id_map` DROP COLUMN `lboxd`;
-- +goose StatementEnd
