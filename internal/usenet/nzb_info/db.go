package nzb_info

import (
	"database/sql"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
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
	Date       string
	Status     string
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
	Date:       "date",
	Status:     "status",
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
	Column.Date,
	Column.Status,
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
	Date         db.Timestamp
	Status       string
	CAt          db.Timestamp
	UAt          db.Timestamp
}

var query_upsert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = EXCLUDED.%s, %s = %s`,
	TableName,
	db.JoinColumnNames(Column.Id, Column.Hash, Column.Name, Column.Size, Column.FileCount, Column.Password, Column.URL, Column.Files, Column.Streamable, Column.User, Column.Date, Column.Status),
	Column.Hash,
	Column.Name, Column.Name,
	Column.Size, Column.Size,
	Column.FileCount, Column.FileCount,
	Column.Password, Column.Password,
	Column.URL, Column.URL,
	Column.Files, Column.Files,
	Column.Streamable, Column.Streamable,
	Column.Date, Column.Date,
	Column.Status, Column.Status,
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
		info.Date,
		info.Status,
	)
	return err
}

var query_update_status = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ?`,
	TableName,
	Column.Status,
	Column.UAt, db.CurrentTimestamp,
	Column.Hash,
)

func UpdateStatus(hash string, status string) error {
	_, err := db.Exec(query_update_status, status, hash)
	return err
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(columns...),
	TableName,
	Column.Id,
)

func GetById(id string) (*NZBInfo, error) {
	row := db.QueryRow(query_get_by_id, id)
	info := NZBInfo{}
	if err := row.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.Date, &info.Status, &info.CAt, &info.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

var query_get_by_hash = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(columns...),
	TableName,
	Column.Hash,
)

func GetByHash(hash string) (*NZBInfo, error) {
	row := db.QueryRow(query_get_by_hash, hash)
	info := NZBInfo{}
	if err := row.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.Date, &info.Status, &info.CAt, &info.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &info, nil
}

var query_get_by_hashes = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s IN `,
	db.JoinColumnNames(columns...),
	TableName,
	Column.Hash,
)

func GetByHashes(hashes []string) (map[string]*NZBInfo, error) {
	byHash := map[string]*NZBInfo{}

	count := len(hashes)
	if count == 0 {
		return byHash, nil
	}

	query := fmt.Sprintf("%s (%s)", query_get_by_hashes, util.RepeatJoin("?", count, ","))
	args := make([]any, count)
	for i, hash := range hashes {
		args[i] = hash
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		info := NZBInfo{}
		if err := rows.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.Date, &info.Status, &info.CAt, &info.UAt); err != nil {
			return nil, err
		}
		byHash[info.Hash] = &info
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return byHash, nil
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s DESC`,
	db.JoinColumnNames(columns...),
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
		if err := rows.Scan(&info.Id, &info.Hash, &info.Name, &info.Size, &info.FileCount, &info.Password, &info.URL, &info.ContentFiles, &info.Streamable, &info.User, &info.Date, &info.Status, &info.CAt, &info.UAt); err != nil {
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
