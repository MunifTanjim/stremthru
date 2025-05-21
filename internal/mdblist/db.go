package mdblist

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "mdblist_list"

type MDBListList struct {
	Id          int          `json:"id"`
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

func (l *MDBListList) GetURL() string {
	if l.UserName != "" && l.Slug != "" {
		return "https://mdblist.com/lists/" + l.UserName + "/" + l.Slug
	}
	if l.Id != 0 {
		return "https://mdblist.com/?list=" + strconv.Itoa(l.Id)
	}
	return ""
}

func (l *MDBListList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(12 * time.Hour))
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
	Id             int       `json:"id"`
	Rank           int       `json:"rank"`
	Adult          bool      `json:"adult"`
	Title          string    `json:"title"`
	Poster         string    `json:"poster"`
	ImdbId         string    `json:"imdb_id"`
	TvdbId         int       `json:"tvdb_id"`
	Language       string    `json:"language"`
	Mediatype      MediaType `json:"mediatype"`
	ReleaseYear    int       `json:"release_year"`
	SpokenLanguage string    `json:"spoken_language"`
	Genre          genreList `json:"-"`
}

type ItemColumnStruct struct {
	Id             string
	Rank           string
	Adult          string
	Title          string
	Poster         string
	ImdbId         string
	TvdbId         string
	Language       string
	Mediatype      string
	ReleaseYear    string
	SpokenLanguage string
}

var ItemColumn = ItemColumnStruct{
	Id:             "id",
	Rank:           "rank",
	Adult:          "adult",
	Title:          "title",
	Poster:         "poster",
	ImdbId:         "imdb_id",
	TvdbId:         "tvdb_id",
	Language:       "language",
	Mediatype:      "mediatype",
	ReleaseYear:    "release_year",
	SpokenLanguage: "spoken_language",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Rank,
	ItemColumn.Adult,
	ItemColumn.Title,
	ItemColumn.Poster,
	ItemColumn.ImdbId,
	ItemColumn.TvdbId,
	ItemColumn.Language,
	ItemColumn.Mediatype,
	ItemColumn.ReleaseYear,
	ItemColumn.SpokenLanguage,
}

const ListItemTableName = "mdblist_list_item"

type MDBListListItem struct {
	ListId int `json:"list_id"`
	ItemId int `json:"item_id"`
}

type ListItemColumnStruct struct {
	ListId string
	ItemId string
}

var ListItemColumn = ListItemColumnStruct{
	ListId: "list_id",
	ItemId: "item_id",
}

var ListItemColumns = []string{
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
}

const ItemGenreTableName = "mdblist_item_genre"

type MDBListItemGenre struct {
	ItemId int    `json:"item_id"`
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

func GetListIdByName(userName, slug string) (int, error) {
	var id int
	row := db.QueryRow(query_get_id_by_name, userName, slug)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id int) (*MDBListList, error) {
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
	`SELECT %s, %s(ig.%s) AS genre FROM %s li JOIN %s i ON i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s WHERE li.%s = ? GROUP BY i.%s ORDER BY i.%s ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	db.FnJSONGroupArray,
	ItemGenreColumn.Genre,
	ListItemTableName,
	ItemTableName,
	ItemColumn.Id,
	ListItemColumn.ItemId,
	ItemGenreTableName,
	ItemColumn.Id,
	ItemGenreColumn.ItemId,
	ListItemColumn.ListId,
	ItemColumn.Id,
	ItemColumn.Rank,
)

func GetListItems(listId int) ([]MDBListItem, error) {
	var items []MDBListItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MDBListItem
		if err := rows.Scan(
			&item.Id,
			&item.Rank,
			&item.Adult,
			&item.Title,
			&item.Poster,
			&item.ImdbId,
			&item.TvdbId,
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
	if list.Id == 0 {
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

	itemIds := make([]int, 0, len(list.Items))
	for i := range list.Items {
		if id := list.Items[i].Id; id != 0 {
			itemIds = append(itemIds, id)
		}
	}

	err = setListItems(tx, list.Id, itemIds)
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
	` ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s`,
	ItemColumn.Id,
	ItemColumn.Rank,
	ItemColumn.Rank,
	ItemColumn.Adult,
	ItemColumn.Adult,
	ItemColumn.Title,
	ItemColumn.Title,
	ItemColumn.Poster,
	ItemColumn.Poster,
	ItemColumn.ImdbId,
	ItemColumn.ImdbId,
	ItemColumn.TvdbId,
	ItemColumn.TvdbId,
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
	filteredItems := make([]MDBListItem, 0, len(items))
	for i := range items {
		if item := &items[i]; item.Id != 0 {
			filteredItems = append(filteredItems, *item)
		}
	}

	if len(filteredItems) == 0 {
		return nil
	}

	for cItems := range slices.Chunk(filteredItems, 500) {
		count := len(cItems)

		query := query_upsert_items +
			util.RepeatJoin(query_upsert_items_placeholder, count, ",") +
			query_upsert_items_after_values

		columnCount := len(ItemColumns)
		args := make([]any, count*columnCount)
		for i, item := range cItems {
			args[i*columnCount] = item.Id
			args[i*columnCount+1] = item.Rank
			args[i*columnCount+2] = item.Adult
			args[i*columnCount+3] = item.Title
			args[i*columnCount+4] = item.Poster
			args[i*columnCount+5] = item.ImdbId
			args[i*columnCount+6] = item.TvdbId
			args[i*columnCount+7] = item.Language
			args[i*columnCount+8] = item.Mediatype
			args[i*columnCount+9] = item.ReleaseYear
			args[i*columnCount+10] = item.SpokenLanguage
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}

		for _, item := range cItems {
			err = setItemGenre(tx, item.Id, item.Genre)
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

func setItemGenre(tx *db.Tx, itemId int, genres []Genre) error {
	count := len(genres)

	cleanupArgs := make([]any, 1+count)
	cleanupArgs[0] = itemId
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
		args[i*2] = itemId
		args[i*2+1] = genre
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

var query_set_list_items_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s, %s) VALUES `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
)
var query_set_list_items_values_placeholder = "(?, ?)"
var query_set_list_items_after_values = fmt.Sprintf(
	` ON CONFLICT DO NOTHING`,
)
var query_cleanup_list_items = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s NOT IN `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
)

func setListItems(tx *db.Tx, listId int, itemIds []int) error {
	count := len(itemIds)

	cleanupArgs := make([]any, 1+count)
	cleanupArgs[0] = listId
	for i, itemId := range itemIds {
		cleanupArgs[1+i] = itemId
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
	args := make([]any, count*2)
	for i, itemId := range itemIds {
		args[i*2] = listId
		args[i*2+1] = itemId
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
