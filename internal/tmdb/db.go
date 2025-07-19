package tmdb

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "tmdb_list"

type TMDBList struct {
	Id          string
	Name        string
	Description string
	Private     bool
	AccountId   string
	Username    string
	UpdatedAt   db.Timestamp

	Items []TMDBItem `json:"-"`
}

func (l *TMDBList) GetURL() string {
	if l.IsUserSpecific() {
		return "https://www.themoviedb.org/u/" + l.Username + "/" + strings.TrimPrefix(l.Id, ID_PREFIX_DYNAMIC_USER_SPECIFIC)
	}
	if l.IsDynamic() {
		return "https://www.themoviedb.org/" + strings.TrimPrefix(l.Id, ID_PREFIX_DYNAMIC)
	}
	return "https://www.themoviedb.org/list/" + l.Id
}

const ID_PREFIX_DYNAMIC = "~:"
const ID_PREFIX_DYNAMIC_USER_SPECIFIC = ID_PREFIX_DYNAMIC + "u:"

func (l *TMDBList) IsDynamic() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_DYNAMIC)
}

func (l *TMDBList) IsUserSpecific() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_DYNAMIC_USER_SPECIFIC)
}

func (l *TMDBList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.TMDB.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

func (l *TMDBList) ShouldPersist() bool {
	if l.IsUserSpecific() {
		return false
	}
	return !l.Private
}

var ListColumn = struct {
	Id          string
	Name        string
	Description string
	Private     string
	AccountId   string
	Username    string
	UpdatedAt   string
}{
	Id:          "id",
	Name:        "name",
	Description: "description",
	Private:     "private",
	AccountId:   "account_id",
	Username:    "username",
	UpdatedAt:   "uat",
}

var ListColumns = []string{
	ListColumn.Id,
	ListColumn.Name,
	ListColumn.Description,
	ListColumn.Private,
	ListColumn.AccountId,
	ListColumn.Username,
	ListColumn.UpdatedAt,
}

const ItemTableName = "tmdb_item"

type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeTVShow MediaType = "tv"
)

type TMDBItem struct {
	Id            int
	Type          MediaType
	IsPartial     bool
	Title         string
	OriginalTitle string
	Overview      string
	ReleaseDate   db.DateOnly
	IsAdult       bool
	Backdrop      string
	Poster        string
	Popularity    float64
	VoteAverage   float64
	VoteCount     int
	UpdatedAt     db.Timestamp

	Idx    int            `json:"-"`
	Genres db.JSONIntList `json:"-"`
}

func (item *TMDBItem) BackdropURL(size BackdropSize) string {
	return IMAGE_BASE_URL + string(size) + item.Backdrop
}

func (item *TMDBItem) PosterURL(size PosterSize) string {
	return IMAGE_BASE_URL + string(size) + item.Poster
}

func (item *TMDBItem) GenreNames() []string {
	genres := make([]string, len(item.Genres))
	for i, genreId := range item.Genres {
		genres[i] = string(genreNameById[genreId])
	}
	return genres
}

var ItemColumn = struct {
	Id            string
	Type          string
	IsPartial     string
	Title         string
	OriginalTitle string
	Overview      string
	ReleaseDate   string
	IsAdult       string
	Backdrop      string
	Poster        string
	Popularity    string
	VoteAverage   string
	VoteCount     string
	UpdatedAt     string
}{
	Id:            "id",
	Type:          "type",
	IsPartial:     "is_partial",
	Title:         "title",
	OriginalTitle: "orig_title",
	Overview:      "overview",
	ReleaseDate:   "release_date",
	IsAdult:       "is_adult",
	Backdrop:      "backdrop",
	Poster:        "poster",
	Popularity:    "popularity",
	VoteAverage:   "vote_average",
	VoteCount:     "vote_count",
	UpdatedAt:     "uat",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Type,
	ItemColumn.IsPartial,
	ItemColumn.Title,
	ItemColumn.OriginalTitle,
	ItemColumn.Overview,
	ItemColumn.ReleaseDate,
	ItemColumn.IsAdult,
	ItemColumn.Backdrop,
	ItemColumn.Poster,
	ItemColumn.Popularity,
	ItemColumn.VoteAverage,
	ItemColumn.VoteCount,
	ItemColumn.UpdatedAt,
}

const ItemGenreTableName = "tmdb_item_genre"

type TMDBItemGenre struct {
	ItemId   int
	ItemType MediaType
	GenreId  int
}

var ItemGenreColumn = struct {
	ItemId   string
	ItemType string
	GenreId  string
}{
	ItemId:   "item_id",
	ItemType: "item_type",
	GenreId:  "genre_id",
}

var ItemGenreColumns = []string{
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.GenreId,
}

const ListItemTableName = "tmdb_list_item"

type TMDBListItem struct {
	ListId int
	ItemId int
	Idx    int
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
	ListItemColumn.Idx,
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*TMDBList, error) {
	row := db.QueryRow(query_get_list_by_id, id)
	list := &TMDBList{}
	if err := row.Scan(
		&list.Id,
		&list.Name,
		&list.Description,
		&list.Private,
		&list.AccountId,
		&list.Username,
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
	ItemGenreColumn.GenreId,
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

func GetListItems(listId string) ([]TMDBItem, error) {
	var items []TMDBItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item TMDBItem
		if err := rows.Scan(
			&item.Id,
			&item.Type,
			&item.IsPartial,
			&item.Title,
			&item.OriginalTitle,
			&item.Overview,
			&item.ReleaseDate,
			&item.IsAdult,
			&item.Backdrop,
			&item.Poster,
			&item.Popularity,
			&item.VoteAverage,
			&item.VoteCount,
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

var query_upsert_list = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s`,
	ListTableName,
	strings.Join(ListColumns[:len(ListColumns)-1], ", "),
	util.RepeatJoin("?", len(ListColumns)-1, ", "),
	ListColumn.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Name, ListColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Description, ListColumn.Description),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Private, ListColumn.Private),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.AccountId, ListColumn.AccountId),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Username, ListColumn.Username),
		fmt.Sprintf(`%s = %s`, ListColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertList(list *TMDBList) (err error) {
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
			list.Name,
			list.Description,
			list.Private,
			list.AccountId,
			list.Username,
		)
		if err != nil {
			return err
		}
	}

	list.UpdatedAt = db.Timestamp{Time: time.Now()}

	err = UpsertItems(tx, list.Items)
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
	`INSERT INTO %s AS li (%s) VALUES `,
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
		fmt.Sprintf(`%s = CASE WHEN li.%s = %s THEN li.%s ELSE EXCLUDED.%s END`, ItemColumn.IsPartial, ItemColumn.IsPartial, db.BooleanTrue, ItemColumn.IsPartial, ItemColumn.IsPartial),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Title, ItemColumn.Title),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.OriginalTitle, ItemColumn.OriginalTitle),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Overview, ItemColumn.Overview),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.ReleaseDate, ItemColumn.ReleaseDate),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.IsAdult, ItemColumn.IsAdult),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Backdrop, ItemColumn.Backdrop),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Poster, ItemColumn.Poster),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Popularity, ItemColumn.Popularity),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.VoteAverage, ItemColumn.VoteAverage),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.VoteCount, ItemColumn.VoteCount),
		fmt.Sprintf(`%s = %s`, ItemColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertItems(tx db.Executor, items []TMDBItem) error {
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
			args[i*columnCount+2] = item.IsPartial
			args[i*columnCount+3] = item.Title
			args[i*columnCount+4] = item.OriginalTitle
			args[i*columnCount+5] = item.Overview
			args[i*columnCount+6] = item.ReleaseDate
			args[i*columnCount+7] = item.IsAdult
			args[i*columnCount+8] = item.Backdrop
			args[i*columnCount+9] = item.Poster
			args[i*columnCount+10] = item.Popularity
			args[i*columnCount+11] = item.VoteAverage
			args[i*columnCount+12] = item.VoteCount
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}

		for _, item := range cItems {
			if err := setItemGenre(tx, item.Id, item.Type, item.Genres); err != nil {
				return err
			}
		}
	}

	return nil
}

var query_set_item_genre_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s,%s) VALUES `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.GenreId,
)
var query_set_item_genre_values_placeholder = `(?,?,?)`
var query_set_item_genre_after_values = ` ON CONFLICT DO NOTHING`
var query_cleanup_item_genre = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ? AND %s NOT IN `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.ItemType,
	ItemGenreColumn.GenreId,
)

func setItemGenre(tx db.Executor, itemId int, itemType MediaType, genres []int) error {
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

func setListItems(tx db.Executor, listId string, items []TMDBItem) error {
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
	args := make([]any, count*4)
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
