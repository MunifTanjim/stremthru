package job_queue

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TableName = "job_queue"

type EntryStatus string

var (
	EntryStatusQueued     EntryStatus = "queued"
	EntryStatusProcessing EntryStatus = "processing"
	EntryStatusFailed     EntryStatus = "failed"
	EntryStatusDone       EntryStatus = "done"
	EntryStatusDead       EntryStatus = "dead"
)

var Column = struct {
	Name         string
	Key          string
	Payload      string
	Status       string
	Error        string
	Priority     string
	ProcessAfter string
	CreatedAt    string
	UpdatedAt    string
}{
	Name:         "name",
	Key:          "key",
	Payload:      "payload",
	Status:       "status",
	Error:        "error",
	Priority:     "priority",
	ProcessAfter: "process_after",
	CreatedAt:    "cat",
	UpdatedAt:    "uat",
}

var columns = []string{
	Column.Name,
	Column.Key,
	Column.Payload,
	Column.Status,
	Column.Error,
	Column.Priority,
	Column.ProcessAfter,
	Column.CreatedAt,
	Column.UpdatedAt,
}

type JobQueueEntry[T any] struct {
	Name         string
	Key          string
	Payload      db.JSONB[T]
	Status       string
	Error        db.JSONStringList
	Priority     int
	ProcessAfter db.Timestamp
	CreatedAt    db.Timestamp
	UpdatedAt    db.Timestamp
}

var query_queue_entry = fmt.Sprintf(
	`INSERT INTO %s AS tq (%s, %s, %s, %s, %s) VALUES (?, ?, ?, ?, ?) ON CONFLICT (%s, %s) DO UPDATE SET %s = EXCLUDED.%s, %s = '%s', %s = '[]', %s = MAX(tq.%s, EXCLUDED.%s), %s = EXCLUDED.%s, %s = %s`,
	TableName,
	Column.Name,
	Column.Key,
	Column.Payload,
	Column.Priority,
	Column.ProcessAfter,
	Column.Name, Column.Key,
	Column.Payload, Column.Payload,
	Column.Status, EntryStatusQueued,
	Column.Error,
	Column.Priority, Column.Priority, Column.Priority,
	Column.ProcessAfter, Column.ProcessAfter,
	Column.UpdatedAt, db.CurrentTimestamp,
)

func QueueEntry[T any](name string, payload T, key string, processAfter time.Time, priority int) error {
	_, err := db.Exec(query_queue_entry,
		name,
		key,
		db.JSONB[T]{Data: payload},
		priority,
		db.Timestamp{Time: processAfter},
	)
	return err
}

var query_get_all_pending_entries = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s NOT IN ('%s', '%s', '%s') AND %s <= %s ORDER BY %s DESC, %s ASC`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.Status,
	EntryStatusProcessing, EntryStatusDone, EntryStatusDead,
	Column.ProcessAfter, db.CurrentTimestamp,
	Column.Priority,
	Column.CreatedAt,
)

func GetAllPendingEntries[T any](name string) ([]JobQueueEntry[T], error) {
	rows, err := db.Query(query_get_all_pending_entries, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []JobQueueEntry[T]{}
	for rows.Next() {
		e := JobQueueEntry[T]{}
		if err := rows.Scan(&e.Name, &e.Key, &e.Payload, &e.Status, &e.Error, &e.Priority, &e.ProcessAfter, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

var query_get_first_entry = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s NOT IN ('%s', '%s', '%s') AND %s <= %s ORDER BY %s DESC, %s ASC LIMIT 1`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.Status,
	EntryStatusProcessing, EntryStatusDone, EntryStatusDead,
	Column.ProcessAfter, db.CurrentTimestamp,
	Column.Priority,
	Column.CreatedAt,
)

func GetFirstEntry[T any](name string) (*JobQueueEntry[T], error) {
	row := db.QueryRow(query_get_first_entry, name)
	e := JobQueueEntry[T]{}
	if err := row.Scan(&e.Name, &e.Key, &e.Payload, &e.Status, &e.Error, &e.Priority, &e.ProcessAfter, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

var query_delay_entries = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ? AND %s IN `,
	TableName,
	Column.ProcessAfter,
	Column.UpdatedAt, db.CurrentTimestamp,
	Column.Name,
	Column.Key,
)

func DelayEntries(name string, keys []string, processAfter time.Time) error {
	if len(keys) == 0 {
		return nil
	}
	args := make([]any, 2+len(keys))
	args[0] = db.Timestamp{Time: processAfter}
	args[1] = name
	for i, key := range keys {
		args[2+i] = key
	}
	query := query_delay_entries + "(" + util.RepeatJoin("?", len(keys), ",") + ")"
	_, err := db.Exec(query, args...)
	return err
}

var query_delete_entries = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s IN `,
	TableName,
	Column.Name,
	Column.Key,
)

func DeleteEntries(name string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	args := make([]any, 1+len(keys))
	args[0] = name
	for i, key := range keys {
		args[1+i] = key
	}
	query := query_delete_entries + "(" + util.RepeatJoin("?", len(keys), ",") + ")"
	_, err := db.Exec(query, args...)
	return err
}

var query_entry_exists = fmt.Sprintf(
	`SELECT 1 FROM %s WHERE %s = ? AND %s NOT IN ('%s', '%s', '%s') AND %s <= %s LIMIT 1`,
	TableName,
	Column.Name,
	Column.Status,
	EntryStatusProcessing, EntryStatusDone, EntryStatusDead,
	Column.ProcessAfter, db.CurrentTimestamp,
)

func EntryExists(name string) (bool, error) {
	row := db.QueryRow(query_entry_exists, name)
	var one int
	if err := row.Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

var query_set_entry_failed = fmt.Sprintf(
	`UPDATE %s SET %s = '%s', %s = ?, %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.Status, EntryStatusFailed,
	Column.Error,
	Column.ProcessAfter,
	Column.UpdatedAt, db.CurrentTimestamp,
	Column.Name,
	Column.Key,
)

func SetEntryFailed(name, key string, errs db.JSONStringList, processAfter time.Time) error {
	_, err := db.Exec(query_set_entry_failed,
		errs,
		db.Timestamp{Time: processAfter},
		name,
		key,
	)
	return err
}

var query_set_entry_dead = fmt.Sprintf(
	`UPDATE %s SET %s = '%s', %s = ?, %s = %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.Status, EntryStatusDead,
	Column.Error,
	Column.UpdatedAt, db.CurrentTimestamp,
	Column.Name,
	Column.Key,
)

func SetEntryDead(name, key string, errs db.JSONStringList) error {
	_, err := db.Exec(query_set_entry_dead,
		errs,
		name,
		key,
	)
	return err
}

var query_set_entries_done = fmt.Sprintf(
	`UPDATE %s SET %s = '%s', %s = %s WHERE %s = ? AND %s IN `,
	TableName,
	Column.Status, EntryStatusDone,
	Column.UpdatedAt, db.CurrentTimestamp,
	Column.Name,
	Column.Key,
)

func SetEntriesDone(name string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	args := make([]any, 1+len(keys))
	args[0] = name
	for i, key := range keys {
		args[1+i] = key
	}
	query := query_set_entries_done + "(" + util.RepeatJoin("?", len(keys), ",") + ")"
	_, err := db.Exec(query, args...)
	return err
}

var query_set_entries_processing = fmt.Sprintf(
	`UPDATE %s SET %s = '%s', %s = %s WHERE %s = ? AND %s IN `,
	TableName,
	Column.Status, EntryStatusProcessing,
	Column.UpdatedAt, db.CurrentTimestamp,
	Column.Name,
	Column.Key,
)

func SetEntriesProcessing(name string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	args := make([]any, 1+len(keys))
	args[0] = name
	for i, key := range keys {
		args[1+i] = key
	}
	query := query_set_entries_processing + "(" + util.RepeatJoin("?", len(keys), ",") + ")"
	_, err := db.Exec(query, args...)
	return err
}

var query_get_entry_by_key = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.Key,
)

func GetEntryByKey[T any](name, key string) (*JobQueueEntry[T], error) {
	row := db.QueryRow(query_get_entry_by_key, name, key)
	e := JobQueueEntry[T]{}
	if err := row.Scan(&e.Name, &e.Key, &e.Payload, &e.Status, &e.Error, &e.Priority, &e.ProcessAfter, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

var query_get_entries_by_name = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? ORDER BY %s DESC`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.CreatedAt,
)

func GetEntriesByName[T any](name string) ([]JobQueueEntry[T], error) {
	rows, err := db.Query(query_get_entries_by_name, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []JobQueueEntry[T]{}
	for rows.Next() {
		e := JobQueueEntry[T]{}
		if err := rows.Scan(&e.Name, &e.Key, &e.Payload, &e.Status, &e.Error, &e.Priority, &e.ProcessAfter, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
