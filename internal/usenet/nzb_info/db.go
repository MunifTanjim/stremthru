package nzb_info

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/rs/xid"
)

const TableName = "nzb_info"

var Column = struct {
	Id         string
	Hash       string
	Name       string
	Size       string
	FileCount  string
	Password   string
	URL        string
	Files      string
	Streamable string
	User       string
	CAt        string
	UAt        string
}{
	Id:         "id",
	Hash:       "hash",
	Name:       "name",
	Size:       "size",
	FileCount:  "file_count",
	Password:   "password",
	URL:        "url",
	Files:      "files",
	Streamable: "streamable",
	User:       "user",
	CAt:        "cat",
	UAt:        "uat",
}

var columns = []string{
	Column.Id,
	Column.Hash,
	Column.Name,
	Column.Size,
	Column.FileCount,
	Column.Password,
	Column.URL,
	Column.Files,
	Column.Streamable,
	Column.User,
	Column.CAt,
	Column.UAt,
}

type NZBInfo struct {
	Id           string
	Hash         string
	Name         string
	Size         int64
	FileCount    int
	Password     string
	URL          string
	ContentFiles db.JSONB[[]usenet_pool.NZBContentFile]
	Streamable   bool
	User         string
	CAt          db.Timestamp
	UAt          db.Timestamp
}

var query_upsert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = %s`,
	TableName,
	strings.Join([]string{Column.Id, Column.Hash, Column.Name, Column.Size, Column.FileCount, Column.Password, Column.URL, Column.Files, Column.Streamable, Column.User}, ", "),
	Column.Hash,
	Column.Name, Column.Name,
	Column.Size, Column.Size,
	Column.FileCount, Column.FileCount,
	Column.Password, Column.Password,
	Column.URL, Column.URL,
	Column.Files, Column.Files,
	Column.Streamable, Column.Streamable,
	Column.UAt, db.CurrentTimestamp,
)

func Upsert(info *NZBInfo) error {
	if info.Id == "" {
		info.Id = xid.New().String()
	}
	_, err := db.Exec(query_upsert,
		info.Id,
		info.Hash,
		info.Name,
		info.Size,
		info.FileCount,
		info.Password,
		info.URL,
		info.ContentFiles,
		info.Streamable,
		info.User,
	)
	return err
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Id,
)

func GetById(id string) (*NZBInfo, error) {
	row := db.QueryRow(query_get_by_id, id)
	info := NZBInfo{}
	if err := row.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.CAt, &info.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

var query_get_by_hash = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Hash,
)

func GetByHash(hash string) (*NZBInfo, error) {
	row := db.QueryRow(query_get_by_hash, hash)
	info := NZBInfo{}
	if err := row.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.CAt, &info.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s DESC`,
	strings.Join(columns, ", "),
	TableName,
	Column.CAt,
)

func GetAll() ([]NZBInfo, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	infos := []NZBInfo{}
	for rows.Next() {
		info := NZBInfo{}
		if err := rows.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.CAt, &info.UAt); err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return infos, nil
}

var query_delete_by_id = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.Id,
)

func DeleteById(id string) error {
	_, err := db.Exec(query_delete_by_id, id)
	return err
}
