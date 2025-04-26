package dmm_hashlist

import (
	"database/sql"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
)

type DMMHashlist struct {
	Id         string       `json:"id"`
	EntryCount int          `json:"entry_count"`
	CAt        db.Timestamp `json:"cat"`
}

type ColumnStruct struct {
	Id         string
	EntryCount string
	CAt        string
}

const TableName = "dmm_hashlist"

var Column = ColumnStruct{
	Id:         "id",
	EntryCount: "entry_count",
	CAt:        "cat",
}

var exist_query = fmt.Sprintf(
	"SELECT %s FROM %s WHERE %s = ?",
	Column.Id,
	TableName,
	Column.Id,
)

func Exists(id string) (bool, error) {
	row := db.QueryRow(exist_query, id)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

var insert_query = fmt.Sprintf(
	"INSERT INTO %s (%s, %s) VALUES (?, ?)",
	TableName,
	Column.Id,
	Column.EntryCount,
)

func Insert(id string, entryCount int) error {
	_, err := db.Exec(insert_query, id, entryCount)
	return err
}
