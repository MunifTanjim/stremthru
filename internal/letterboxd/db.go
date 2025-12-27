package letterboxd

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const ListTableName = "letterboxd_list"

type LetterboxdList struct {
	Id          string
	UserId      string
	UserName    string
	Name        string
	Slug        string
	Description string
	Private     bool
	ItemCount   int
	Version     int64
	UpdatedAt   db.Timestamp

	Items []LetterboxdItem `json:"-"`
}

const ID_PREFIX_DYNAMIC = "~:"
const ID_PREFIX_USER_WATCHLIST = ID_PREFIX_DYNAMIC + "watchlist:"

func (l *LetterboxdList) IsStale() bool {
	return time.Now().After(l.UpdatedAt.Add(config.Integration.Letterboxd.ListStaleTime + util.GetRandomDuration(5*time.Second, 5*time.Minute)))
}

func (l *LetterboxdList) StaleIn() time.Duration {
	if l.HasUnfetchedItems() {
		return 15 * time.Minute
	}
	staleIn := time.Until(l.UpdatedAt.Time.Add(config.Integration.Letterboxd.ListStaleTime))
	if staleIn <= 0 {
		staleIn = 15 * time.Minute
	}
	return staleIn
}

func (l *LetterboxdList) GetURL() string {
	if l.IsUserWatchlist() {
		return SITE_BASE_URL + "/" + strings.ToLower(l.UserName) + "/watchlist"
	}
	return SITE_BASE_URL + "/" + strings.ToLower(l.UserName) + "/list/" + l.Slug
}

func (l *LetterboxdList) IsUserWatchlist() bool {
	return strings.HasPrefix(l.Id, ID_PREFIX_USER_WATCHLIST)
}

func (l *LetterboxdList) HasUnfetchedItems() bool {
	fetchedItemCount := len(l.Items)
	return l.ItemCount > fetchedItemCount && fetchedItemCount < MAX_LIST_ITEM_COUNT
}

var ListColumn = struct {
	Id          string
	UserId      string
	UserName    string
	Name        string
	Slug        string
	Description string
	Private     string
	ItemCount   string
	Version     string
	UpdatedAt   string
}{
	Id:          "id",
	UserId:      "user_id",
	UserName:    "user_name",
	Name:        "name",
	Slug:        "slug",
	Description: "description",
	Private:     "private",
	ItemCount:   "item_count",
	Version:     "v",
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
	ListColumn.ItemCount,
	ListColumn.Version,
	ListColumn.UpdatedAt,
}

const ItemTableName = "letterboxd_item"

type LetterboxdItem struct {
	Id          string
	Name        string
	ReleaseYear int
	Runtime     int
	Rating      int
	Adult       bool
	Poster      string
	UpdatedAt   db.Timestamp

	GenreIds db.JSONStringList `json:"-"`
	IdMap    *meta.IdMap       `json:"-"`
	Rank     int               `json:"-"`
}

func (li LetterboxdItem) GenreNames() []string {
	genres := make([]string, len(li.GenreIds))
	for i, genreId := range li.GenreIds {
		if genre, ok := genreNameById[genreId]; ok {
			genres[i] = genre
		}
	}
	return genres
}

var ItemColumn = struct {
	Id          string
	Name        string
	ReleaseYear string
	Runtime     string
	Rating      string
	Adult       string
	Poster      string
	UpdatedAt   string
}{
	Id:          "id",
	Name:        "name",
	ReleaseYear: "release_year",
	Runtime:     "runtime",
	Rating:      "rating",
	Adult:       "adult",
	Poster:      "poster",
	UpdatedAt:   "uat",
}

var ItemColumns = []string{
	ItemColumn.Id,
	ItemColumn.Name,
	ItemColumn.ReleaseYear,
	ItemColumn.Runtime,
	ItemColumn.Rating,
	ItemColumn.Adult,
	ItemColumn.Poster,
	ItemColumn.UpdatedAt,
}

const ItemGenreTableName = "letterboxd_item_genre"

type LetterboxdItemGenre struct {
	ItemId  string
	GenreId string
}

var ItemGenreColumn = struct {
	ItemId  string
	GenreId string
}{
	ItemId:  "item_id",
	GenreId: "genre_id",
}

const ListItemTableName = "letterboxd_list_item"

type LetterboxdListItem struct {
	ListId string
	ItemId string
	Rank   int
}

var ListItemColumn = struct {
	ListId string
	ItemId string
	Rank   string
}{
	ListId: "list_id",
	ItemId: "item_id",
	Rank:   "rank",
}

var query_get_list_id_by_slug = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	ListColumn.Id,
	ListTableName,
	ListColumn.UserName,
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

var query_get_list_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(ListColumns...),
	ListTableName,
	ListColumn.Id,
)

func GetListById(id string) (*LetterboxdList, error) {
	row := db.QueryRow(query_get_list_by_id, id)
	list := &LetterboxdList{}
	if err := row.Scan(
		&list.Id,
		&list.UserId,
		&list.UserName,
		&list.Name,
		&list.Slug,
		&list.Description,
		&list.Private,
		&list.ItemCount,
		&list.Version,
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
	`SELECT %s, %s(ig.%s) AS genre FROM %s li JOIN %s i ON i.%s = li.%s LEFT JOIN %s ig ON i.%s = ig.%s WHERE li.%s = ? GROUP BY i.%s ORDER BY min(li.%s) ASC`,
	db.JoinPrefixedColumnNames("i.", ItemColumns...),
	db.FnJSONGroupArray,
	ItemGenreColumn.GenreId,
	ListItemTableName,
	ItemTableName,
	ItemColumn.Id,
	ListItemColumn.ItemId,
	ItemGenreTableName,
	ItemColumn.Id,
	ItemGenreColumn.ItemId,
	ListItemColumn.ListId,
	ItemColumn.Id,
	ListItemColumn.Rank,
)

func GetListItems(listId string) ([]LetterboxdItem, error) {
	var items []LetterboxdItem
	rows, err := db.Query(query_get_list_items, listId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item LetterboxdItem
		if err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.ReleaseYear,
			&item.Runtime,
			&item.Rating,
			&item.Adult,
			&item.Poster,
			&item.UpdatedAt,
			&item.GenreIds,
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
	strings.Join(ListColumns, ", "),
	util.RepeatJoin("?", len(ListColumns), ", "),
	ListColumn.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.UserId, ListColumn.UserId),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.UserName, ListColumn.UserName),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Name, ListColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Slug, ListColumn.Slug),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Description, ListColumn.Description),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Private, ListColumn.Private),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.ItemCount, ListColumn.ItemCount),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.Version, ListColumn.Version),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ListColumn.UpdatedAt, ListColumn.UpdatedAt),
	}, ", "),
)

func UpsertList(list *LetterboxdList) (err error) {
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

	if list.UpdatedAt.IsZero() {
		list.UpdatedAt = db.Timestamp{Time: time.Now()}
	}
	_, err = tx.Exec(
		query_upsert_list,
		list.Id,
		list.UserId,
		list.UserName,
		list.Name,
		list.Slug,
		list.Description,
		list.Private,
		list.ItemCount,
		list.Version,
		list.UpdatedAt,
	)
	if err != nil {
		return err
	}

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

var query_upsert_items_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	ItemTableName,
	strings.Join(ItemColumns, ", "),
)
var query_upsert_items_values_placholder = fmt.Sprintf(
	`(%s)`,
	util.RepeatJoin("?", len(ItemColumns), ","),
)
var query_upsert_items_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s`,
	ItemColumn.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Name, ItemColumn.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.ReleaseYear, ItemColumn.ReleaseYear),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Runtime, ItemColumn.Runtime),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Rating, ItemColumn.Rating),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Adult, ItemColumn.Adult),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.Poster, ItemColumn.Poster),
		fmt.Sprintf(`%s = EXCLUDED.%s`, ItemColumn.UpdatedAt, ItemColumn.UpdatedAt),
	}, ", "),
)

func upsertItems(tx db.Executor, items []LetterboxdItem) error {
	if len(items) == 0 {
		return nil
	}

	now := time.Now()

	for cItems := range slices.Chunk(items, 250) {
		count := len(cItems)

		query := query_upsert_items_before_values +
			util.RepeatJoin(query_upsert_items_values_placholder, count, ",") +
			query_upsert_items_after_values

		columnCount := len(ItemColumns)
		args := make([]any, count*columnCount)
		for i, item := range cItems {
			if item.UpdatedAt.IsZero() {
				item.UpdatedAt = db.Timestamp{Time: now}
			}
			args[i*columnCount+0] = item.Id
			args[i*columnCount+1] = item.Name
			args[i*columnCount+2] = item.ReleaseYear
			args[i*columnCount+3] = item.Runtime
			args[i*columnCount+4] = item.Rating
			args[i*columnCount+5] = item.Adult
			args[i*columnCount+6] = item.Poster
			args[i*columnCount+7] = item.UpdatedAt
		}

		_, err := tx.Exec(query, args...)
		if err != nil {
			return err
		}

		idMaps := make([]meta.IdMap, 0, count)
		for _, item := range cItems {
			if err := setItemGenre(tx, item.Id, item.GenreIds); err != nil {
				return err
			}

			if item.IdMap.IMDB != "" {
				idMaps = append(idMaps, *item.IdMap)
			}
		}
		util.LogError(log, meta.SetIdMapsInTrx(tx, idMaps, meta.IdProviderIMDB), "failed to set id maps")
	}

	return nil
}

var query_set_item_genre_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s) VALUES `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.GenreId,
)
var query_set_item_genre_values_placeholder = `(?,?)`
var query_set_item_genre_after_values = ` ON CONFLICT DO NOTHING`
var query_cleanup_item_genre = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s NOT IN `,
	ItemGenreTableName,
	ItemGenreColumn.ItemId,
	ItemGenreColumn.GenreId,
)

func setItemGenre(tx db.Executor, itemId string, genreIds []string) error {
	count := len(genreIds)

	if count == 0 {
		return nil
	}

	cleanupQuery := query_cleanup_item_genre + "(" + util.RepeatJoin("?", count, ",") + ")"
	cleanupArgs := make([]any, 1+count)
	cleanupArgs[0] = itemId
	for i, genre := range genreIds {
		cleanupArgs[1+i] = genre
	}
	if _, err := tx.Exec(cleanupQuery, cleanupArgs...); err != nil {
		return err
	}

	query := query_set_item_genre_before_values +
		util.RepeatJoin(query_set_item_genre_values_placeholder, count, ",") +
		query_set_item_genre_after_values
	args := make([]any, len(genreIds)*2)
	for i, genreId := range genreIds {
		args[i*2+0] = itemId
		args[i*2+1] = genreId
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	return nil
}

var query_set_list_item_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s,%s,%s) VALUES `,
	ListItemTableName,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Rank,
)
var query_set_list_item_values_placeholder = `(?,?,?)`
var query_set_list_item_after_values = fmt.Sprintf(
	` ON CONFLICT (%s,%s) DO UPDATE SET %s = EXCLUDED.%s`,
	ListItemColumn.ListId,
	ListItemColumn.ItemId,
	ListItemColumn.Rank,
	ListItemColumn.Rank,
)
var query_cleanup_list_item = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	ListItemTableName,
	ListItemColumn.ListId,
)

func setListItems(tx db.Executor, listId string, items []LetterboxdItem) error {
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
		args[i*3+1] = item.Id
		args[i*3+2] = item.Rank
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	return nil
}
