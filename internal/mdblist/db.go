package mdblist

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "mdblist_list"

type MDBListList struct {
	Id          int       `json:"id"`
	UserId      int       `json:"user_id"`
	UserName    string    `json:"user_name"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Mediatype   MediaType `json:"mediatype"`
	Dynamic     bool      `json:"dynamic"`
	Likes       int       `json:"likes"`
	UpdatedAt   string    `json:"uat"`

	Items []MDBListItem `json:"-"`
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
		return 0, err
	}
	if id == 0 {
		return 0, nil
	}
	return id, nil
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s JOIN WHERE %s = ?`,
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
	`SELECT %s, %s(ig.%s) AS genre FROM %s li JOIN %s i ON i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s WHERE li.%s = ? ORDER BY i.%s ASC`,
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
	`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = %s`,
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
	ListColumn.Likes,
	ListColumn.Likes,
	ListColumn.UpdatedAt,
	db.CurrentTimestamp,
)

func UpsertList(list *MDBListList) error {
	if list.Id == 0 {
		return errors.New("list id is missing")
	}

	_, err := db.Exec(
		query_upsert_list,
		list.Id,
		list.UserId,
		list.UserName,
		list.Name,
		list.Slug,
		list.Description,
		list.Mediatype,
		list.Dynamic,
		list.Likes,
	)
	return err
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

func UpsertItems(items []MDBListItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}

	query := query_upsert_items +
		util.RepeatJoin(query_upsert_items_placeholder, count, ",") +
		query_upsert_items_after_values
	columnCount := len(ItemColumns)
	args := make([]any, count*columnCount)
	for i, item := range items {
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

	_, err := db.Exec(query, args...)
	return err
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

func SetListItems(listId int, itemIds []int) error {
	count := len(itemIds)

	cleanupArgs := make([]any, count)
	for i, itemId := range itemIds {
		cleanupArgs[i] = itemId
	}
	if _, err := db.Exec(query_cleanup_list_items+"("+util.RepeatJoin("?", count, ",")+")", cleanupArgs...); err != nil {
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

	if _, err := db.Exec(query, args...); err != nil {
		return err
	}

	return nil
}
