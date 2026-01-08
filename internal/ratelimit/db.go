package ratelimit

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/rs/xid"
)

const TableName = "rate_limit_config"

type RateLimitConfig struct {
	Id     string
	Name   string
	Limit  int
	Window string // Duration string like "1m", "1h", "30s"
	CAt    db.Timestamp
	UAt    db.Timestamp
}

var Column = struct {
	Id     string
	Name   string
	Limit  string
	Window string
	CAt    string
	UAt    string
}{
	Id:     "id",
	Name:   "name",
	Limit:  "limit",
	Window: "window",
	CAt:    "cat",
	UAt:    "uat",
}

var columns = []string{
	Column.Id,
	Column.Name,
	Column.Limit,
	Column.Window,
	Column.CAt,
	Column.UAt,
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s DESC`,
	db.JoinColumnNames(columns...),
	TableName,
	Column.CAt,
)

func GetAll() ([]RateLimitConfig, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []RateLimitConfig{}
	for rows.Next() {
		item := RateLimitConfig{}
		if err := rows.Scan(&item.Id, &item.Name, &item.Limit, &item.Window, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(columns...),
	TableName,
	Column.Id,
)

func GetById(id string) (*RateLimitConfig, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := RateLimitConfig{}
	if err := row.Scan(&item.Id, &item.Name, &item.Limit, &item.Window, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_get_by_name = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	db.JoinColumnNames(columns...),
	TableName,
	Column.Name,
)

func GetByName(name string) (*RateLimitConfig, error) {
	row := db.QueryRow(query_get_by_name, name)

	item := RateLimitConfig{}
	if err := row.Scan(&item.Id, &item.Name, &item.Limit, &item.Window, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?,?)`,
	TableName,
	db.JoinColumnNames(
		Column.Id,
		Column.Name,
		Column.Limit,
		Column.Window,
	),
)

func Create(name string, limit int, window string) (*RateLimitConfig, error) {
	id := xid.New().String()

	_, err := db.Exec(query_insert, id, name, limit, window)
	if err != nil {
		return nil, err
	}

	return GetById(id)
}

var query_update = fmt.Sprintf(
	`UPDATE %s SET %s = ?, "%s" = ?, %s = ?, %s = %s WHERE %s = ?`,
	TableName,
	Column.Name,
	Column.Limit,
	Column.Window,
	Column.UAt,
	db.CurrentTimestamp,
	Column.Id,
)

func Update(id string, name string, limit int, window string) (*RateLimitConfig, error) {
	_, err := db.Exec(query_update, name, limit, window, id)
	if err != nil {
		return nil, err
	}

	return GetById(id)
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

func (c *RateLimitConfig) ParseWindow() (time.Duration, error) {
	return time.ParseDuration(c.Window)
}
