package imdb_title

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const MetaTableName = "imdb_title_meta"

type genreList []string

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

type IMDBTitleMeta struct {
	TId         string       `json:"tid"`
	Description string       `json:"description"`
	Runtime     int          `json:"runtime"`
	Poster      string       `json:"poster"`
	Backdrop    string       `json:"backdrop"`
	Trailer     string       `json:"trailer"`
	Rating      int          `json:"rating"`
	MPARating   string       `json:"mpa_rating"`
	UpdatedAt   db.Timestamp `json:"uat"`

	Genres genreList `json:"-"`
}

func (m IMDBTitleMeta) IsStale() bool {
	return time.Now().After(m.UpdatedAt.Add(7 * 24 * time.Hour))
}

type MetaColumnStruct struct {
	TId         string
	Description string
	Runtime     string
	Poster      string
	Backdrop    string
	Trailer     string
	Rating      string
	MPARating   string
	UpdatedAt   string
}

var MetaColumn = MetaColumnStruct{
	TId:         "tid",
	Description: "description",
	Runtime:     "runtime",
	Poster:      "poster",
	Backdrop:    "backdrop",
	Trailer:     "trailer",
	Rating:      "rating",
	MPARating:   "mpa_rating",
	UpdatedAt:   "uat",
}

var MetaColumns = []string{
	MetaColumn.TId,
	MetaColumn.Description,
	MetaColumn.Runtime,
	MetaColumn.Poster,
	MetaColumn.Backdrop,
	MetaColumn.Trailer,
	MetaColumn.Rating,
	MetaColumn.MPARating,
	MetaColumn.UpdatedAt,
}

var query_upsert_metas_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	MetaTableName,
	db.JoinColumnNames(
		MetaColumn.TId,
		MetaColumn.Description,
		MetaColumn.Runtime,
		MetaColumn.Poster,
		MetaColumn.Backdrop,
		MetaColumn.Trailer,
		MetaColumn.Rating,
		MetaColumn.MPARating,
	),
)
var query_upsert_metas_values_placeholder = "(" + util.RepeatJoin("?", 8, ",") + ")"
var query_upsert_metas_after_values = fmt.Sprintf(
	` ON CONFLICT (%s) DO UPDATE SET %s`,
	MetaColumn.TId,
	strings.Join(
		[]string{
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Description, MetaColumn.Description),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Runtime, MetaColumn.Runtime),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Poster, MetaColumn.Poster),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Backdrop, MetaColumn.Backdrop),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Trailer, MetaColumn.Trailer),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.Rating, MetaColumn.Rating),
			fmt.Sprintf("%s = EXCLUDED.%s", MetaColumn.MPARating, MetaColumn.MPARating),
			fmt.Sprintf("%s = %s", MetaColumn.UpdatedAt, db.CurrentTimestamp),
		},
		", ",
	),
)

func UpsertMetas(metas []IMDBTitleMeta) (err error) {
	count := len(metas)
	if count == 0 {
		return nil
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

	query := query_upsert_metas_before_values +
		util.RepeatJoin(query_upsert_metas_values_placeholder, count, ",") +
		query_upsert_metas_after_values
	args := make([]any, 0, len(metas)*8)
	for _, meta := range metas {
		args = append(
			args,
			meta.TId,
			meta.Description,
			meta.Runtime,
			meta.Poster,
			meta.Backdrop,
			meta.Trailer,
			meta.Rating,
			meta.MPARating,
		)
	}

	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}

	if err := recordGenre(tx, metas); err != nil {
		return err
	}

	return nil
}

var query_get_metas_by_ids = fmt.Sprintf(
	`SELECT %s, %s(itg.%s) AS genres FROM %s itm JOIN %s itg ON itg.%s = itm.%s WHERE itm.%s IN `,
	db.JoinPrefixedColumnNames("itm.", MetaColumns...),
	db.FnJSONGroupArray,
	GenreColumn.Genre,
	MetaTableName,
	GenreTableName,
	GenreColumn.TId,
	MetaColumn.TId,
	MetaColumn.TId,
)
var query_get_metas_by_ids_group_by = fmt.Sprintf(
	` GROUP BY itm.%s`,
	MetaColumn.TId,
)

func GetMetasByIds(imdbIds []string) ([]IMDBTitleMeta, error) {
	count := len(imdbIds)
	if count == 0 {
		return nil, nil
	}

	query := query_get_metas_by_ids +
		"(" + util.RepeatJoin("?", count, ",") + ") " +
		query_get_metas_by_ids_group_by

	args := make([]any, 0, count)
	for _, id := range imdbIds {
		args = append(args, id)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metas []IMDBTitleMeta
	for rows.Next() {
		var meta IMDBTitleMeta
		if err := rows.Scan(
			&meta.TId,
			&meta.Description,
			&meta.Runtime,
			&meta.Poster,
			&meta.Backdrop,
			&meta.Trailer,
			&meta.Rating,
			&meta.MPARating,
			&meta.UpdatedAt,
			&meta.Genres,
		); err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metas, nil
}
