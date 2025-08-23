package mdblist

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "mdblist_list"

type MDBListList struct {
	Id          string       `json:"id"`
	UserId      int          `json:"user_id"`
	UserName    string       `json:"user_name"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	Mediatype   MediaType    `json:"mediatype"`
	Dynamic     bool         `json:"dynamic"`
	Private     bool         `json:"private"`
	Likes       int          `json:"likes"`
	UpdatedAt   db.Timestamp `json:"uat"`

	Items []MDBListItem `json:"-"`
}

const ID_PREFIX_USER_WATCHLIST = "~:watchlist:"

func (l *MDBListList) IsWatchlist() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_USER_WATCHLIST)
}

func (l *MDBListList) GetURL() string {
	if l.IsWatchlist() {
		return "https://mdblist.com/" + l.Slug
	}
	if l.UserName != "" && l.Slug != "" {
		return "https://mdblist.com/lists/" + l.UserName + "/" + l.Slug
	}
	if l.Id != "" {
		return "https://mdblist.com/?list=" + l.Id
	}
	return ""
}

func (l *MDBListList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.MDBList.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

type ListColumnStruct struct {
	Id          string
	UserId      string
	UserName    string
	Name        string
	Slug        string
	Description string
	Mediatype   string
	Dynamic     string
	Private     string
	Likes       string
	UpdatedAt   string
}

var ListColumn = ListColumnStruct{
	Id:          "id",
	UserId:      "user_id",
	UserName:    "user_name",
	Name:        "name",
	Slug:        "slug",
	Description: "description",
	Mediatype:   "mediatype",
	Dynamic:     "dynamic",
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
	ListColumn.Mediatype,
	ListColumn.Dynamic,
	ListColumn.Private,
	ListColumn.Likes,
	ListColumn.UpdatedAt,
}

const ItemTableName = "mdblist_item"

type genreList []Genre

func (genre genreList) Value() (driver.Value, error) {
	return json.Marshal(genre)
}

func (files *genreList) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, files)
}

type MDBListItem struct {
	IMDBId         string    `json:"imdb_id"`
	Adult          bool      `json:"adult"`
	Title          string    `json:"title"`
	Poster         string    `json:"poster"`
	Language       string    `json:"language"`
	Mediatype      MediaType `json:"mediatype"`
	ReleaseYear    int       `json:"release_year"`
	SpokenLanguage string    `json:"spoken_language"`

	Genre  genreList `json:"-"`
	Rank   int       `json:"-"`
	TmdbId string    `json:"-"`
	TvdbId string    `json:"-"`
}

func (li MDBListItem) GenreNames() []string {
	genres := make([]string, len(li.Genre))
	for i, genreId := range li.Genre {
		genres[i] = string(genreNameById[genreId])
	}
	return genres
}

type ItemColumnStruct struct {
	IMDBId         string
	Adult          string
	Title          string
	Poster         string
	Language       string
	Mediatype      string
	ReleaseYear    string
	SpokenLanguage string
}

var ItemColumn = ItemColumnStruct{
	IMDBId:         "imdb_id",
	Adult:          "adult",
	Title:          "title",
	Poster:         "poster",
	Language:       "language",
	Mediatype:      "mediatype",
	ReleaseYear:    "release_year",
	SpokenLanguage: "spoken_language",
}

var ItemColumns = []string{
	ItemColumn.IMDBId,
	ItemColumn.Adult,
	ItemColumn.Title,
	ItemColumn.Poster,
	ItemColumn.Language,
	ItemColumn.Mediatype,
	ItemColumn.ReleaseYear,
	ItemColumn.SpokenLanguage,
}

const ListItemTableName = "mdblist_list_item"

type MDBListListItem struct {
	ListId string `json:"list_id"`
	ItemId int    `json:"item_id"`
	Rank   int    `json:"rank"`
}

type ListItemColumnStruct struct {
	ListId string
	ItemId string
	Rank   string
}

var ListItemColumn = ListItemColumnStruct{
	ListId: "list_id",
	ItemId: "item_id",
	Rank:   "rank",
}

var ListItemColumns = []string{
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Rank,
}

const ItemGenreTableName = "mdblist_item_genre"

type MDBListItemGenre struct {
	ItemId string `json:"item_id"`
	Genre  string `json:"genre"`
}

type ItemGenreColumnStruct struct {
	ItemId string
	Genre  string
}

var ItemGenreColumn = ItemGenreColumnStruct{
	ItemId: "item_id",
	Genre:  "genre",
}

var ItemGenreColumns = []string{
	ItemGenreColumn.ItemId,
	ItemGenreColumn.Genre,
}

var query_get_id_by_name = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	ListColumn.Id,
	ListTableName,
	ListColumn.UserName,
	ListColumn.Slug,
)

func GetListIdByName(userName, slug string) (string, error) {
	var id string
	row := db.QueryRow(query_get_id_by_name, userName, slug)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*MDBListList, error) {
	var list MDBListList
	row := db.QueryRow(query_get_list_by_id, id)
	if err := row.Scan(
		&list.Id,
		&list.UserId,
		&list.UserName,
		&list.Name,
		&list.Slug,
		&list.Description,
		&list.Mediatype,
		&list.Dynamic,
		&list.Private,
		&list.Likes,
		&list.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	items, err := GetListItems(list.Id)
	if err != nil {
		return nil, err
	}
	list.Items = items
	return &list, nil
}

var query_get_list_by_name = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.UserName,
	ListColumn.Slug,
)

func GetListByName(userName, slug string) (*MDBListList, error) {
	var list MDBListList
	row := db.QueryRow(query_get_list_by_name, userName, slug)
	if err := row.Scan(
		&list.Id,
		&list.UserId,
		&list.UserName,
		&list.Name,
		&list.Slug,
		&list.Description,
		&list.Mediatype,
		&list.Dynamic,
		&list.Private,
		&list.Likes,
		&list.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	items, err := GetListItems(list.Id)
	if err != nil {
		return nil, err
	}
	list.Items = items
	return &list, nil
}

var query_get_list_items = fmt.Sprintf(
	`SELECT %s, %s(ig.%s) AS genre FROM %s li JOIN %s i ON i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s WHERE li.%s = ? GROUP BY i.%s ORDER BY min(li.%s) ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	db.FnJSONGroupArray,
	ItemGenreColumn.Genre,
	ListItemTableName,
	ItemTableName,
	ItemColumn.IMDBId,
	ListItemColumn.ItemId,
	ItemGenreTableName,
	ItemColumn.IMDBId,
	ItemGenreColumn.ItemId,
	ListItemColumn.ListId,
	ItemColumn.IMDBId,
	ListItemColumn.Rank,
)

func GetListItems(listId string) ([]MDBListItem, error) {
	var items []MDBListItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MDBListItem
		if err := rows.Scan(
			&item.IMDBId,
			&item.Adult,
			&item.Title,
			&item.Poster,
			&item.Language,
			&item.Mediatype,
			&item.ReleaseYear,
			&item.SpokenLanguage,
			&item.Genre,
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
	`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = %s`,
	ListTableName,
	db.JoinColumnNames(ListColumns[:len(ListColumns)-1]...),
	util.RepeatJoin("?", len(ListColumns)-1, ","),
	ListColumn.Id,
	ListColumn.UserId,
	ListColumn.UserId,
	ListColumn.UserName,
	ListColumn.UserName,
	ListColumn.Name,
	ListColumn.Name,
	ListColumn.Slug,
	ListColumn.Slug,
	ListColumn.Description,
	ListColumn.Description,
	ListColumn.Mediatype,
	ListColumn.Mediatype,
	ListColumn.Dynamic,
	ListColumn.Dynamic,
	ListColumn.Private,
	ListColumn.Private,
	ListColumn.Likes,
	ListColumn.Likes,
	ListColumn.UpdatedAt,
	db.CurrentTimestamp,
)

func UpsertList(list *MDBListList) (err error) {
	if list.Id == "" {
		return errors.New("list id is missing")
	}

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

	_, err = tx.Exec(
		query_upsert_list,
		list.Id,
		list.UserId,
		list.UserName,
		list.Name,
		list.Slug,
		list.Description,
		list.Mediatype,
		list.Dynamic,
		list.Private,
		list.Likes,
	)
	if err != nil {
		return err
	}

	list.UpdatedAt = db.Timestamp{Time: time.Now()}

	err = upsertItems(tx, list.Items)
	if err != nil {
		return err
	}

	err = setListItems(tx, list.Id, list.Items)
	if err != nil {
		return err
	}

	return nil
}

var query_upsert_items = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	ItemTableName,
	db.JoinColumnNames(ItemColumns...),
)
var query_upsert_items_placeholder = fmt.Sprintf(
	"(%s)",
	util.RepeatJoin("?", len(ItemColumns), ","),
)
var query_upsert_items_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s`,
	ItemColumn.IMDBId,
	ItemColumn.Adult,
	ItemColumn.Adult,
	ItemColumn.Title,
	ItemColumn.Title,
	ItemColumn.Poster,
	ItemColumn.Poster,
	ItemColumn.Language,
	ItemColumn.Language,
	ItemColumn.Mediatype,
	ItemColumn.Mediatype,
	ItemColumn.ReleaseYear,
	ItemColumn.ReleaseYear,
	ItemColumn.SpokenLanguage,
	ItemColumn.SpokenLanguage,
)

func upsertItems(tx *db.Tx, items []MDBListItem) error {
	if len(items) == 0 {
		return nil
	}

	for cItems := range slices.Chunk(items, 500) {
		count := len(cItems)

		query := query_upsert_items +
			util.RepeatJoin(query_upsert_items_placeholder, count, ",") +
			query_upsert_items_after_values

		columnCount := len(ItemColumns)
		args := make([]any, count*columnCount)
		for i, item := range cItems {
			args[i*columnCount+0] = item.IMDBId
			args[i*columnCount+1] = item.Adult
			args[i*columnCount+2] = item.Title
			args[i*columnCount+3] = item.Poster
			args[i*columnCount+4] = item.Language
			args[i*columnCount+5] = item.Mediatype
			args[i*columnCount+6] = item.ReleaseYear
			args[i*columnCount+7] = item.SpokenLanguage
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}

		for _, item := range cItems {
			err = setItemGenre(tx, item.IMDBId, item.Genre)
			if err != nil {
				return err
			}
			err = imdb_title.RecordMappingFromMDBList(tx, item.IMDBId, item.TmdbId, item.TvdbId, "", "")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

var query_set_item_genre_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s, %s) VALUES `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.Genre,
)
var query_set_item_genre_values_placeholder = "(?, ?)"
var query_set_item_genre_after_values = fmt.Sprintf(
	` ON CONFLICT DO NOTHING`,
)
var query_cleanup_item_genre = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s NOT IN `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.Genre,
)

func setItemGenre(tx *db.Tx, itemTId string, genres []Genre) error {
	count := len(genres)

	cleanupArgs := make([]any, 1+count)
	cleanupArgs[0] = itemTId
	for i, genre := range genres {
		cleanupArgs[1+i] = genre
	}
	cleanupQuery := query_cleanup_item_genre + "(" + util.RepeatJoin("?", count, ",") + ")"
	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	query := query_set_item_genre_before_values +
		util.RepeatJoin(query_set_item_genre_values_placeholder, count, ",") +
		query_set_item_genre_after_values
	args := make([]any, count*2)
	for i, genre := range genres {
		args[i*2] = itemTId
		args[i*2+1] = genre
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

var query_set_list_items_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s, %s, %s) VALUES `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Rank,
)
var query_set_list_items_values_placeholder = "(?,?,?)"
var query_set_list_items_after_values = fmt.Sprintf(
	` ON CONFLICT (%s, %s) DO UPDATE SET %s = EXCLUDED.%s`,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Rank,
	ListItemColumn.Rank,
)
var query_cleanup_list_items = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s NOT IN `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
)

func setListItems(tx *db.Tx, listId string, items []MDBListItem) error {
	count := len(items)

	cleanupArgs := make([]any, 1+count)
	cleanupArgs[0] = listId
	for i := range items {
		cleanupArgs[1+i] = items[i].IMDBId
	}
	cleanupQuery := query_cleanup_list_items + "(" + util.RepeatJoin("?", count, ",") + ")"
	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	query := query_set_list_items_before_values +
		util.RepeatJoin(query_set_list_items_values_placeholder, count, ",") +
		query_set_list_items_after_values
	args := make([]any, count*3)
	for i := range items {
		item := &items[i]
		args[i*3+0] = listId
		args[i*3+1] = item.IMDBId
		args[i*3+2] = item.Rank
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
