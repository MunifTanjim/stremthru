package torrent_review

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type ReviewRequest struct {
	Id         int64        `json:"id"`
	Hash       string       `json:"hash"`
	Reason     ReviewReason `json:"reason"`
	PrevIMDBId string       `json:"prev_imdb_id"`
	IMDBId     string       `json:"imdb_id"`
	Files      []File       `json:"files"`
	Comment    string       `json:"comment"`
	IP         string       `json:"ip"`
	CreatedAt  db.Timestamp `json:"created_at"`
}

type InsertItem struct {
	Hash       string
	Reason     ReviewReason
	PrevIMDBId string
	IMDBId     string
	Files      []File
	Comment    string
	IP         string
}

const TableName = "torrent_review_request"

var Column = struct {
	Id         string
	Hash       string
	Reason     string
	PrevIMDBId string
	IMDBId     string
	Files      string
	Comment    string
	IP         string
	CreatedAt  string
}{
	Id:         "id",
	Hash:       "hash",
	Reason:     "reason",
	PrevIMDBId: "prev_imdb_id",
	IMDBId:     "imdb_id",
	Files:      "files",
	Comment:    "comment",
	IP:         "ip",
	CreatedAt:  "created_at",
}

var Columns = []string{
	Column.Id,
	Column.Hash,
	Column.Reason,
	Column.PrevIMDBId,
	Column.IMDBId,
	Column.Files,
	Column.Comment,
	Column.IP,
	Column.CreatedAt,
}

var get_by_id_query = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(Columns...),
	TableName,
	Column.Id,
)

func GetByID(id int64) (*ReviewRequest, error) {
	row := db.QueryRow(get_by_id_query, id)
	var r ReviewRequest
	var filesJSON sql.NullString

	if err := row.Scan(
		&r.Id,
		&r.Hash,
		&r.Reason,
		&r.PrevIMDBId,
		&r.IMDBId,
		&filesJSON,
		&r.Comment,
		&r.IP,
		&r.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse files JSON
	if filesJSON.Valid && filesJSON.String != "" {
		if err := json.Unmarshal([]byte(filesJSON.String), &r.Files); err != nil {
			return nil, fmt.Errorf("failed to parse files: %w", err)
		}
	}

	return &r, nil
}

var insert_query_before_values = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES `,
	TableName,
	strings.Join([]string{
		Column.Hash,
		Column.Reason,
		Column.PrevIMDBId,
		Column.IMDBId,
		Column.Files,
		Column.Comment,
		Column.IP,
	}, ","),
)

var insert_query_values_placeholder = "(" + util.RepeatJoin("?", 7, ",") + ")"

func Insert(items []InsertItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}

	query := insert_query_before_values + util.RepeatJoin(insert_query_values_placeholder, count, ",")

	args := make([]any, 0, count*7)
	for _, item := range items {
		args = append(args,
			item.Hash,
			item.Reason,
			item.PrevIMDBId,
			item.IMDBId,
			item.Files,
			item.Comment,
			item.IP,
		)
	}

	_, err := db.Exec(query, args...)
	return err
}

var list_query_count = fmt.Sprintf(`SELECT COUNT(*) FROM %s`, TableName)

var list_query_select = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s DESC LIMIT ? OFFSET ?`,
	db.JoinColumnNames(Columns...),
	TableName,
	Column.CreatedAt,
)

func List(limit, offset int) ([]ReviewRequest, int, error) {
	// Get total count
	var total int
	row := db.QueryRow(list_query_count)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get items
	rows, err := db.Query(list_query_select, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ReviewRequest
	for rows.Next() {
		var r ReviewRequest
		var filesJSON sql.NullString

		if err := rows.Scan(
			&r.Id,
			&r.Hash,
			&r.Reason,
			&r.PrevIMDBId,
			&r.IMDBId,
			&filesJSON,
			&r.Comment,
			&r.IP,
			&r.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		// Parse files JSON
		if filesJSON.Valid && filesJSON.String != "" {
			if err := json.Unmarshal([]byte(filesJSON.String), &r.Files); err != nil {
				return nil, 0, fmt.Errorf("failed to parse files: %w", err)
			}
		}

		items = append(items, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
