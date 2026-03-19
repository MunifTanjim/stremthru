package serializd

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

const ListTableName = "serializd_list"

type SerializdList struct {
	Id        string
	Name      string
	UpdatedAt db.Timestamp

	Items []SerializdItem
}

const ID_PREFIX_DYNAMIC = "~:"

func (l *SerializdList) GetURL() string {
	return SITE_BASE_URL_PARSED.JoinPath(strings.TrimPrefix(l.Id, ID_PREFIX_DYNAMIC)).String()
}

func (l *SerializdList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.Serializd.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

var ListColumn = struct {
	Id        string
	Name      string
	UpdatedAt string
}{
	Id:        "id",
	Name:      "name",
	UpdatedAt: "uat",
}

var ListColumns = []string{
	ListColumn.Id,
	ListColumn.Name,
	ListColumn.UpdatedAt,
}

const ItemTableName = "serializd_item"

type SerializdItem struct {
	ID          int // TMDB ID
	Name        string
	BannerImage string
	UpdatedAt   db.Timestamp

	Idx int `json:"-"`
}

func (item *SerializdItem) PosterURL() string {
	if item.BannerImage == "" {
		return ""
	}
	return "https://serializd-tmdb-images.b-cdn.net/t/p/w500" + item.BannerImage
}

var ItemColumn = struct {
	Id          string
	Name        string
	BannerImage string
	UpdatedAt   string
}{
	Id:          "id",
	Name:        "name",
	BannerImage: "banner_image",
	UpdatedAt:   "uat",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Name,
	ItemColumn.BannerImage,
	ItemColumn.UpdatedAt,
}

const ListItemTableName = "serializd_list_item"

var ListItemColumn = struct {
	ListId string
	ItemId string
	Idx    string
}{
	ListId: "list_id",
	ItemId: "item_id",
	Idx:    "idx",
}

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*SerializdList, error) {
	row := db.QueryRow(query_get_list_by_id, id)
	list := &SerializdList{}
	if err := row.Scan(
		&list.Id,
		&list.Name,
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
	`SELECT %s, min(li.%s) FROM %s li JOIN %s i ON i.%s = li.%s WHERE li.%s = ? GROUP BY i.%s ORDER BY min(li.%s) ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	ListItemColumn.Idx,
	ListItemTableName,
	ItemTableName,
	ItemColumn.Id,
	ListItemColumn.ItemId,
	ListItemColumn.ListId,
	ItemColumn.Id,
	ListItemColumn.Idx,
)

func GetListItems(listId string) ([]SerializdItem, error) {
	var items []SerializdItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item SerializdItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.BannerImage,
			&item.UpdatedAt,
			&item.Idx,
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
		fmt.Sprintf(`%s = %s`, ListColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertList(list *SerializdList) (err error) {
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
var query_upsert_items_values_placeholder = fmt.Sprintf(
	`(%s)`,
	util.RepeatJoin("?", len(ItemColumns)-1, ","),
)
var query_upsert_items_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s`,
	ItemColumn.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Name, ItemColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.BannerImage, ItemColumn.BannerImage),
		fmt.Sprintf(`%s = %s`, ItemColumn.UpdatedAt, db.CurrentTimestamp),
	}, ", "),
)

func UpsertItems(tx db.Executor, items []SerializdItem) error {
	if len(items) == 0 {
		return nil
	}

	columnCount := len(ItemColumns) - 1
	for cItems := range slices.Chunk(items, 500) {
		count := len(cItems)

		query := query_upsert_items_before_values +
			util.RepeatJoin(query_upsert_items_values_placeholder, count, ",") +
			query_upsert_items_after_values

		args := make([]any, count*columnCount)
		for i, item := range cItems {
			args[i*columnCount+0] = item.ID
			args[i*columnCount+1] = item.Name
			args[i*columnCount+2] = item.BannerImage
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}
	}

	return nil
}

var query_set_list_item_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	ListItemTableName,
	db.JoinColumnNames(
		ListItemColumn.ListId,
		ListItemColumn.ItemId,
		ListItemColumn.Idx,
	),
)
var query_set_list_item_values_placeholder = `(?,?,?)`
var query_set_list_item_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s) DO UPDATE SET %s = EXCLUDED.%s`,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Idx,
	ListItemColumn.Idx,
)
var query_cleanup_list_item = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	ListItemTableName,
	ListItemColumn.ListId,
)

func setListItems(tx db.Executor, listId string, items []SerializdItem) error {
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
	args := make([]any, len(items)*3)
	for i, item := range items {
		args[i*3+0] = listId
		args[i*3+1] = item.ID
		args[i*3+2] = item.Idx
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	return nil
}
