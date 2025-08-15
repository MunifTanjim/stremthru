package trakt

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "trakt_list"

type TraktList struct {
	Id          string
	UserId      string
	UserName    string
	Name        string
	Slug        string
	Description string
	Private     bool
	Likes       int
	UpdatedAt   db.Timestamp

	Items []TraktItem `json:"-"`
}

const ID_PREFIX_DYNAMIC = "~:"
const ID_PREFIX_USER_FAVORITES = ID_PREFIX_DYNAMIC + "favorites:"
const ID_PREFIX_USER_WATCHLIST = ID_PREFIX_DYNAMIC + "watchlist:"

const ID_PREFIX_DYNAMIC_USER_SPECIFIC = ID_PREFIX_DYNAMIC + "u:"
const USER_MOVIES_RECOMMENDATIONS_ID = ID_PREFIX_DYNAMIC_USER_SPECIFIC + "movies/recommendations"
const USER_SHOWS_RECOMMENDATIONS_ID = ID_PREFIX_DYNAMIC_USER_SPECIFIC + "shows/recommendations"

func (l *TraktList) GetURL() string {
	if !strings.HasPrefix(l.Id, ID_PREFIX_DYNAMIC) {
		return "https://trakt.tv/users/" + l.UserId + "/lists/" + l.Slug
	}

	if l.IsStandard() {
		slug, _, _ := strings.Cut(strings.TrimPrefix(l.Id, ID_PREFIX_DYNAMIC), ":")
		return "https://trakt.tv/users/" + l.UserId + "/" + slug
	}

	return "https://trakt.tv/" + l.Slug
}

func (l *TraktList) IsDynamic() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_DYNAMIC)
}

func (l *TraktList) IsStandard() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_USER_FAVORITES) ||
		strings.HasPrefix(l.Id, ID_PREFIX_USER_WATCHLIST)
}

func (l *TraktList) IsUserRecommendations() bool {
	return l.Id == USER_MOVIES_RECOMMENDATIONS_ID ||
		l.Id == USER_SHOWS_RECOMMENDATIONS_ID
}

func (l *TraktList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.Trakt.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

func (l *TraktList) ShouldPersist() bool {
	if !l.IsDynamic() {
		return true
	}
	if l.IsUserRecommendations() {
		return false
	}
	if l.IsStandard() {
		return true
	}
	return !l.Private
}

var ListColumn = struct {
	Id          string
	UserId      string
	UserName    string
	Name        string
	Slug        string
	Description string
	Private     string
	Likes       string
	UpdatedAt   string
}{
	Id:          "id",
	UserId:      "user_id",
	UserName:    "user_name",
	Name:        "name",
	Slug:        "slug",
	Description: "description",
	Private:     "private",
	Likes:       "likes",
	UpdatedAt:   "uat",
}

var ListColumns = []string{
	ListColumn.Id,
	ListColumn.UserId,
	ListColumn.UserName,
	ListColumn.Name,
	ListColumn.Slug,
	ListColumn.Description,
	ListColumn.Private,
	ListColumn.Likes,
	ListColumn.UpdatedAt,
}

const ItemTableName = "trakt_item"

type TraktItem struct {
	Id        int
	Type      ItemType
	Title     string
	Year      int
	Overview  string
	Runtime   int
	Poster    string
	Fanart    string
	Trailer   string
	Rating    int
	MPARating string
	UpdatedAt db.Timestamp

	Idx    int               `json:"-"`
	Genres db.JSONStringList `json:"-"`
	Ids    ListItemIds       `json:"-"`
}

var ItemColumn = struct {
	Id        string
	Type      string
	Title     string
	Year      string
	Overview  string
	Runtime   string
	Poster    string
	Fanart    string
	Trailer   string
	Rating    string
	MPARating string
	UpdatedAt string
}{
	Id:        "id",
	Type:      "type",
	Title:     "title",
	Year:      "year",
	Overview:  "overview",
	Runtime:   "runtime",
	Poster:    "poster",
	Fanart:    "fanart",
	Trailer:   "trailer",
	Rating:    "rating",
	MPARating: "mpa_rating",
	UpdatedAt: "uat",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Type,
	ItemColumn.Title,
	ItemColumn.Year,
	ItemColumn.Overview,
	ItemColumn.Runtime,
	ItemColumn.Poster,
	ItemColumn.Fanart,
	ItemColumn.Trailer,
	ItemColumn.Rating,
	ItemColumn.MPARating,
	ItemColumn.UpdatedAt,
}

const ItemGenreTableName = "trakt_item_genre"

type TraktItemGenre struct {
	ItemId   int
	ItemType ItemType
	Genre    string
}

var ItemGenreColumn = struct {
	ItemId   string
	ItemType string
	Genre    string
}{
	ItemId:   "item_id",
	ItemType: "item_type",
	Genre:    "genre",
}

var ItemGenreColumns = []string{
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.Genre,
}

const ListItemTableName = "trakt_list_item"

type TraktListItem struct {
	ListId   string
	ItemId   int
	ItemType ItemType
	Idx      int
}

var ListItemColumn = struct {
	ListId   string
	ItemId   string
	ItemType string
	Idx      string
}{
	ListId:   "list_id",
	ItemId:   "item_id",
	ItemType: "item_type",
	Idx:      "idx",
}

var ListItemColumns = []string{
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.ItemType,
	ListItemColumn.Idx,
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*TraktList, error) {
	row := db.QueryRow(query_get_list_by_id, id)
	list := &TraktList{}
	if err := row.Scan(
		&list.Id,
		&list.UserId,
		&list.UserName,
		&list.Name,
		&list.Slug,
		&list.Description,
		&list.Private,
		&list.Likes,
		&list.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	items, err := GetListItems(id)
	if err != nil {
		return nil, err
	}
	list.Items = items
	return list, nil
}

var query_get_list_items = fmt.Sprintf(
	`SELECT %s, min(li.%s), %s(ig.%s) AS genres FROM %s li JOIN %s i ON i.%s = li.%s AND i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s AND i.%s = ig.%s WHERE li.%s = ? GROUP BY i.%s, i.%s ORDER BY min(li.%s) ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	ListItemColumn.Idx,
	db.FnJSONGroupArray,
	ItemGenreColumn.Genre,
	ListItemTableName,
	ItemTableName,
	ItemColumn.Id,
	ListItemColumn.ItemId,
	ItemColumn.Type,
	ListItemColumn.ItemType,
	ItemGenreTableName,
	ItemColumn.Id,
	ItemGenreColumn.ItemId,
	ItemColumn.Type,
	ItemGenreColumn.ItemType,
	ListItemColumn.ListId,
	ItemColumn.Id,
	ItemColumn.Type,
	ListItemColumn.Idx,
)

func GetListItems(listId string) ([]TraktItem, error) {
	var items []TraktItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item TraktItem
		if err := rows.Scan(
			&item.Id,
			&item.Type,
			&item.Title,
			&item.Year,
			&item.Overview,
			&item.Runtime,
			&item.Poster,
			&item.Fanart,
			&item.Trailer,
			&item.Rating,
			&item.MPARating,
			&item.UpdatedAt,
			&item.Idx,
			&item.Genres,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

var query_get_list_id_by_slug = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	ListColumn.Id,
	ListTableName,
	ListColumn.UserId,
	ListColumn.Slug,
)

func GetListIdBySlug(userId, slug string) (string, error) {
	var id string
	row := db.QueryRow(query_get_list_id_by_slug, userId, slug)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

var query_upsert_list = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s`,
	ListTableName,
	strings.Join(ListColumns[:len(ListColumns)-1], ", "),
	util.RepeatJoin("?", len(ListColumns)-1, ", "),
	ListColumn.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.UserId, ListColumn.UserId),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.UserName, ListColumn.UserName),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Name, ListColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Slug, ListColumn.Slug),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Description, ListColumn.Description),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Private, ListColumn.Private),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Likes, ListColumn.Likes),
		fmt.Sprintf(`%s = %s`, ListColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertList(list *TraktList) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		tErr := tx.Rollback()
		err = errors.Join(tErr, err)
	}()

	if list.ShouldPersist() {
		_, err = tx.Exec(
			query_upsert_list,
			list.Id,
			list.UserId,
			list.UserName,
			list.Name,
			list.Slug,
			list.Description,
			list.Private,
			list.Likes,
		)
		if err != nil {
			return err
		}
	}

	list.UpdatedAt = db.Timestamp{Time: time.Now()}

	err = upsertItems(tx, list.Items)
	if err != nil {
		return err
	}

	if list.ShouldPersist() {
		err = setListItems(tx, list.Id, list.Items)
		if err != nil {
			return err
		}
	}

	return nil
}

var query_upsert_items_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	ItemTableName,
	strings.Join(ItemColumns[:len(ItemColumns)-1], ", "),
)
var query_upsert_items_values_placholder = fmt.Sprintf(
	`(%s)`,
	util.RepeatJoin("?", len(ItemColumns)-1, ","),
)
var query_upsert_items_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s) DO UPDATE SET %s`,
	ItemColumn.Id,
	ItemColumn.Type,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Type, ItemColumn.Type),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Title, ItemColumn.Title),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Year, ItemColumn.Year),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Overview, ItemColumn.Overview),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Runtime, ItemColumn.Runtime),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Poster, ItemColumn.Poster),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Fanart, ItemColumn.Fanart),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Trailer, ItemColumn.Trailer),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Rating, ItemColumn.Rating),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.MPARating, ItemColumn.MPARating),
		fmt.Sprintf(`%s = %s`, ItemColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func upsertItems(tx db.Executor, items []TraktItem) error {
	if len(items) == 0 {
		return nil
	}

	for cItems := range slices.Chunk(items, 500) {
		count := len(cItems)

		query := query_upsert_items_before_values +
			util.RepeatJoin(query_upsert_items_values_placholder, count, ",") +
			query_upsert_items_after_values

		columnCount := len(ItemColumns) - 1
		args := make([]any, count*columnCount)
		for i, item := range cItems {
			args[i*columnCount+0] = item.Id
			args[i*columnCount+1] = item.Type
			args[i*columnCount+2] = item.Title
			args[i*columnCount+3] = item.Year
			args[i*columnCount+4] = item.Overview
			args[i*columnCount+5] = item.Runtime
			args[i*columnCount+6] = item.Poster
			args[i*columnCount+7] = item.Fanart
			args[i*columnCount+8] = item.Trailer
			args[i*columnCount+9] = item.Rating
			args[i*columnCount+10] = item.MPARating
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}

		idMaps := make([]meta.IdMap, 0, count)
		for _, item := range cItems {
			if err := setItemGenre(tx, item.Id, item.Type, item.Genres); err != nil {
				return err
			}

			if item.Ids.IMDB != "" {
				idMapType := meta.IdTypeUnknown
				switch item.Type {
				case ItemTypeMovie:
					idMapType = meta.IdTypeMovie
				case ItemTypeShow:
					idMapType = meta.IdTypeShow
				}

				idMaps = append(idMaps, meta.IdMap{
					Type:  idMapType,
					IMDB:  item.Ids.IMDB,
					TMDB:  strconv.Itoa(item.Ids.TMDB),
					TVDB:  strconv.Itoa(item.Ids.TVDB),
					Trakt: strconv.Itoa(item.Ids.Trakt),
				})
			}
		}
		go util.LogError(log, meta.SetIdMaps(idMaps, meta.IdProviderIMDB), "failed to set id maps")
	}

	return nil
}

var query_set_item_genre_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s,%s) VALUES `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.Genre,
)
var query_set_item_genre_values_placeholder = `(?,?,?)`
var query_set_item_genre_after_values = ` ON CONFLICT DO NOTHING`
var query_cleanup_item_genre = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ? AND %s NOT IN `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.Genre,
)

func setItemGenre(tx db.Executor, itemId int, itemType ItemType, genres []string) error {
	count := len(genres)

	if count == 0 {
		return nil
	}

	cleanupQuery := query_cleanup_item_genre + "(" + util.RepeatJoin("?", count, ",") + ")"
	cleanupArgs := make([]any, 2+count)
	cleanupArgs[0] = itemId
	cleanupArgs[1] = itemType
	for i, genre := range genres {
		cleanupArgs[2+i] = genre
	}
	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	query := query_set_item_genre_before_values +
		util.RepeatJoin(query_set_item_genre_values_placeholder, count, ",") +
		query_set_item_genre_after_values
	args := make([]any, len(genres)*3)
	for i, genre := range genres {
		args[i*3+0] = itemId
		args[i*3+1] = itemType
		args[i*3+2] = genre
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

var query_set_list_item_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s,%s,%s) VALUES `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.ItemType,
	ListItemColumn.Idx,
)
var query_set_list_item_values_placeholder = `(?,?,?,?)`
var query_set_list_item_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s,%s) DO UPDATE SET %s = EXCLUDED.%s`,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.ItemType,
	ListItemColumn.Idx,
	ListItemColumn.Idx,
)
var query_cleanup_list_item = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	ListItemTableName,
	ListItemColumn.ListId,
)

func setListItems(tx db.Executor, listId string, items []TraktItem) error {
	count := len(items)

	if count == 0 {
		return nil
	}

	cleanupQuery := query_cleanup_list_item
	cleanupArgs := []any{listId}
	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	query := query_set_list_item_before_values +
		util.RepeatJoin(query_set_list_item_values_placeholder, count, ",") +
		query_set_list_item_after_values
	args := make([]any, len(items)*4)
	for i, item := range items {
		args[i*4+0] = listId
		args[i*4+1] = item.Id
		args[i*4+2] = item.Type
		args[i*4+3] = item.Idx
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	return nil
}
