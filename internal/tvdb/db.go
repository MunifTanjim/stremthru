package tvdb

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

const ListTableName = "tvdb_list"

type TVDBList struct {
	Id         string
	Name       string
	Slug       string
	Overview   string
	IsOfficial bool
	UpdatedAt  db.Timestamp

	Items []TVDBItem `json:"-"`
}

func (l *TVDBList) GetURL() string {
	return "https://www.thetvdb.com/lists/" + l.Slug
}

func (l *TVDBList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.TVDB.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

var ListColumn = struct {
	Id         string
	Name       string
	Slug       string
	Overview   string
	IsOfficial string
	UpdatedAt  string
}{
	Id:         "id",
	Name:       "name",
	Slug:       "slug",
	Overview:   "overview",
	IsOfficial: "is_official",
	UpdatedAt:  "uat",
}

var ListColumns = []string{
	ListColumn.Id,
	ListColumn.Name,
	ListColumn.Slug,
	ListColumn.Overview,
	ListColumn.IsOfficial,
	ListColumn.UpdatedAt,
}

const ItemTableName = "tvdb_item"

type TVDBItemType string

const (
	TVDBItemTypeMovie  TVDBItemType = "movie"
	TVDBItemTypeSeries TVDBItemType = "series"
)

type TVDBItem struct {
	Id         int
	Type       TVDBItemType
	Name       string
	Overview   string
	Year       int
	Runtime    int
	Poster     string
	Background string
	Trailer    string
	UpdatedAt  db.Timestamp

	Order  int            `json:"-"`
	Genres db.JSONIntList `json:"-"`
	IdMap  *meta.IdMap    `json:"-"`
}

func (item *TVDBItem) GenreNames() []string {
	genres := make([]string, len(item.Genres))
	for i, genreId := range item.Genres {
		genres[i] = string(genreNameById[genreId])
	}
	return genres
}

func (li *TVDBItem) HasBasicMeta() bool {
	return li.Name != ""
}

func (li *TVDBItem) IsStale() bool {
	return time.Now().After(li.UpdatedAt.Add(7 * 24 * time.Hour))
}

var ItemColumn = struct {
	Id         string
	Type       string
	Name       string
	Overview   string
	Year       string
	Runtime    string
	Poster     string
	Background string
	Trailer    string
	UpdatedAt  string
}{
	Id:         "id",
	Type:       "type",
	Name:       "name",
	Overview:   "overview",
	Year:       "year",
	Runtime:    "runtime",
	Poster:     "poster",
	Background: "background",
	Trailer:    "trailer",
	UpdatedAt:  "uat",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Type,
	ItemColumn.Name,
	ItemColumn.Overview,
	ItemColumn.Year,
	ItemColumn.Runtime,
	ItemColumn.Poster,
	ItemColumn.Background,
	ItemColumn.Trailer,
	ItemColumn.UpdatedAt,
}

const ItemGenreTableName = "tvdb_item_genre"

type TVDBItemGenre struct {
	ItemId   int
	ItemType TVDBItemType
	Genre    int
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

const ListItemTableName = "tvdb_list_item"

type TVDBListItem struct {
	ListId   string
	ItemId   int
	ItemType TVDBItemType
	Order    int
}

var ListItemColumn = struct {
	ListId   string
	ItemId   string
	ItemType string
	Order    string
}{
	ListId:   "list_id",
	ItemId:   "item_id",
	ItemType: "item_type",
	Order:    "order",
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*TVDBList, error) {
	row := db.QueryRow(query_get_list_by_id, id)
	list := &TVDBList{}
	if err := row.Scan(
		&list.Id,
		&list.Name,
		&list.Slug,
		&list.Overview,
		&list.IsOfficial,
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
	`SELECT %s, min(li."%s"), %s(ig.%s) AS genres FROM %s li JOIN %s i ON i.%s = li.%s AND i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s AND i.%s = ig.%s WHERE li.%s = ? GROUP BY i.%s, i.%s ORDER BY min(li."%s") ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	ListItemColumn.Order,
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
	ListItemColumn.Order,
)

func GetListItems(listId string) ([]TVDBItem, error) {
	var items []TVDBItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item TVDBItem
		if err := rows.Scan(
			&item.Id,
			&item.Type,
			&item.Name,
			&item.Overview,
			&item.Year,
			&item.Runtime,
			&item.Poster,
			&item.Background,
			&item.Trailer,
			&item.UpdatedAt,
			&item.Order,
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
	`SELECT %s FROM %s WHERE %s = ?`,
	ListColumn.Id,
	ListTableName,
	ListColumn.Slug,
)

func GetListIdBySlug(slug string) (string, error) {
	var id string
	row := db.QueryRow(query_get_list_id_by_slug, slug)
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
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Name, ListColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Slug, ListColumn.Slug),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Overview, ListColumn.Overview),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.IsOfficial, ListColumn.IsOfficial),
		fmt.Sprintf(`%s = %s`, ListColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertList(list *TVDBList) (err error) {
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
		list.Name,
		list.Slug,
		list.Overview,
		list.IsOfficial,
	)
	if err != nil {
		return err
	}

	list.UpdatedAt = db.Timestamp{Time: time.Now()}

	err = UpsertItems(tx, list.Items)
	if err != nil {
		return err
	}

	err = setListItems(tx, list.Id, list.Items)
	if err != nil {
		return err
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
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Name, ItemColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Overview, ItemColumn.Overview),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Year, ItemColumn.Year),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Runtime, ItemColumn.Runtime),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Poster, ItemColumn.Poster),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Background, ItemColumn.Background),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Trailer, ItemColumn.Trailer),
		fmt.Sprintf(`%s = %s`, ItemColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertItems(tx db.Executor, items []TVDBItem) error {
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
			args[i*columnCount+2] = item.Name
			args[i*columnCount+3] = item.Overview
			args[i*columnCount+4] = item.Year
			args[i*columnCount+5] = item.Runtime
			args[i*columnCount+6] = item.Poster
			args[i*columnCount+7] = item.Background
			args[i*columnCount+8] = item.Trailer
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

			if item.IdMap != nil && item.IdMap.IMDB != "" {
				idMaps = append(idMaps, *item.IdMap)
			}
		}
		util.LogError(log, meta.SetIdMapsInTrx(tx, idMaps, meta.IdProviderIMDB), "failed to set id maps")
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

func setItemGenre(tx db.Executor, itemId int, itemType TVDBItemType, genres []int) error {
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
	`INSERT INTO %s (%s) VALUES `,
	ListItemTableName,
	db.JoinColumnNames(
		ListItemColumn.ListId,
		ListItemColumn.ItemId,
		ListItemColumn.ItemType,
		ListItemColumn.Order,
	),
)
var query_set_list_item_values_placeholder = `(?,?,?,?)`
var query_set_list_item_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s,%s) DO UPDATE SET "%s" = EXCLUDED."%s"`,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.ItemType,
	ListItemColumn.Order,
	ListItemColumn.Order,
)
var query_cleanup_list_item = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	ListItemTableName,
	ListItemColumn.ListId,
)

func setListItems(tx db.Executor, listId string, items []TVDBItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}

	if _, err := tx.Exec(query_cleanup_list_item, listId); err != nil {
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
		args[i*4+3] = item.Order
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	return nil
}

var query_get_items_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s IN `,
	db.JoinColumnNames(ItemColumns...),
	ItemTableName,
	ItemColumn.Type,
	ItemColumn.Id,
)

func GetItemsById(itemType TVDBItemType, ids ...int) (map[int]*TVDBItem, error) {
	count := len(ids)
	if count == 0 {
		return nil, nil
	}

	query := query_get_items_by_id + "(" + util.RepeatJoin("?", count, ",") + ")"
	args := make([]any, 1+count)
	args[0] = itemType
	for i := range ids {
		args[1+i] = ids[i]
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itemById := make(map[int]*TVDBItem, count)
	for rows.Next() {
		li := &TVDBItem{}
		if err := rows.Scan(
			&li.Id,
			&li.Type,
			&li.Name,
			&li.Overview,
			&li.Year,
			&li.Runtime,
			&li.Poster,
			&li.Background,
			&li.Trailer,
			&li.UpdatedAt,
		); err != nil {
			return nil, err
		}
		itemById[li.Id] = li
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return itemById, nil
}

func GetItemById(itemType TVDBItemType, id int) (*TVDBItem, error) {
	itemById, err := GetItemsById(itemType, id)
	if err != nil {
		return nil, err
	}
	item, ok := itemById[id]
	if !ok {
		return nil, nil
	}
	idType := meta.IdTypeUnknown
	switch itemType {
	case TVDBItemTypeMovie:
		idType = meta.IdTypeMovie
	case TVDBItemTypeSeries:
		idType = meta.IdTypeShow
	}
	idMap, err := meta.GetIdMap(idType, "tvdb:"+strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	item.IdMap = idMap
	return item, nil
}
