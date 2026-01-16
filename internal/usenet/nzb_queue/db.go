package nzb_queue

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/rs/xid"
)

const TableName = "nzb_queue"

type Status string

const (
	StatusQueued Status = "queued"
)

type NzbQueueItem struct {
	Id       string
	Name     string
	URL      string
	Category string
	Priority int
	Password string
	Status   Status
	Error    string
	User     string
	CAt      db.Timestamp
	UAt      db.Timestamp
}

func NewNzbQueueItem(user, name, url, category string, priority int, password string) *NzbQueueItem {
	return &NzbQueueItem{
		Id:       xid.New().String(),
		Name:     name,
		URL:      url,
		Category: category,
		Priority: priority,
		Password: password,
		Status:   StatusQueued,
		User:     user,
	}
}

var Column = struct {
	Id       string
	Name     string
	URL      string
	Category string
	Priority string
	Password string
	Status   string
	Error    string
	User     string
	CAt      string
	UAt      string
}{
	Id:       "id",
	Name:     "name",
	URL:      "url",
	Category: "category",
	Priority: "priority",
	Password: "password",
	Status:   "status",
	Error:    "error",
	User:     "user",
	CAt:      "cat",
	UAt:      "uat",
}

var columns = []string{
	Column.Id,
	Column.Name,
	Column.URL,
	Column.Category,
	Column.Priority,
	Column.Password,
	Column.Status,
	Column.Error,
	Column.User,
	Column.CAt,
	Column.UAt,
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (%s)`,
	TableName,
	db.JoinColumnNames(columns[0:len(columns)-2]...),
	util.RepeatJoin("?", len(columns)-2, ", "),
)

func (i *NzbQueueItem) Insert() error {
	_, err := db.Exec(query_insert,
		i.Id,
		i.Name,
		i.URL,
		i.Category,
		i.Priority,
		i.Password,
		i.Status,
		i.Error,
		i.User,
	)
	return err
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s DESC`,
	strings.Join(columns, ", "),
	TableName,
	Column.CAt,
)

func GetAll() ([]NzbQueueItem, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []NzbQueueItem{}
	for rows.Next() {
		item := NzbQueueItem{}
		if err := rows.Scan(&item.Id, &item.Name, &item.URL, &item.Category, &item.Priority, &item.Password, &item.Status, &item.Error, &item.User, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Id,
)

func GetById(id string) (*NzbQueueItem, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := NzbQueueItem{}
	if err := row.Scan(&item.Id, &item.Name, &item.URL, &item.Category, &item.Priority, &item.Password, &item.Status, &item.Error, &item.User, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_update_status = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = ?, %s = %s WHERE %s = ?`,
	TableName,
	Column.Status,
	Column.Error,
	Column.UAt,
	db.CurrentTimestamp,
	Column.Id,
)

func UpdateStatus(id string, status Status, errMsg string) error {
	_, err := db.Exec(query_update_status, status, errMsg, id)
	return err
}

var query_delete = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.Id,
)

func Delete(id string) error {
	_, err := db.Exec(query_delete, id)
	return err
}
