package job_log

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TableName = "job_log"

var log = logger.Scoped(TableName)

type JobLog struct {
	Name      string
	Id        string
	Status    string
	Data      db.NullString
	Error     string
	CreatedAt db.Timestamp
	UpdatedAt db.Timestamp
	ExpiresAt db.Timestamp
}

type ParsedJobLog[T any] struct {
	Name      string    `json:"name"`
	Id        string    `json:"id"`
	Status    string    `json:"status"`
	Data      *T        `json:"data,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var Column = struct {
	Name      string
	Id        string
	Status    string
	Data      string
	Error     string
	CreatedAt string
	UpdatedAt string
	ExpiresAt string
}{
	Name:      "name",
	Id:        "id",
	Status:    "status",
	Data:      "data",
	Error:     "error",
	CreatedAt: "created_at",
	UpdatedAt: "updated_at",
	ExpiresAt: "expires_at",
}

var columns = []string{
	Column.Name,
	Column.Id,
	Column.Status,
	Column.Data,
	Column.Error,
	Column.CreatedAt,
	Column.UpdatedAt,
	Column.ExpiresAt,
}

var query_get_job_log = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.Id,
)

func GetJobLog[T any](name string, id string) (*ParsedJobLog[T], error) {
	if id == "" {
		return nil, fmt.Errorf("job id cannot be empty")
	}

	row := db.QueryRow(query_get_job_log, name, id)
	jl := JobLog{}
	if err := row.Scan(
		&jl.Name,
		&jl.Id,
		&jl.Status,
		&jl.Data,
		&jl.Error,
		&jl.CreatedAt,
		&jl.UpdatedAt,
		&jl.ExpiresAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if !jl.ExpiresAt.IsZero() && jl.ExpiresAt.Before(time.Now()) {
		_ = DeleteJobLog(name, id)
		return nil, nil
	}

	pjl := &ParsedJobLog[T]{
		Name:      jl.Name,
		Id:        jl.Id,
		Status:    jl.Status,
		Error:     jl.Error,
		CreatedAt: jl.CreatedAt.Time,
		UpdatedAt: jl.UpdatedAt.Time,
	}

	if !jl.Data.IsZero() && !jl.Data.Is("null") {
		var data T
		if err := json.Unmarshal([]byte(jl.Data.String), &data); err != nil {
			return nil, err
		}
		pjl.Data = &data
	}

	return pjl, nil
}

var query_get_last_job_log = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? ORDER BY %s DESC LIMIT 1`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
	Column.CreatedAt,
)

func GetLastJobLog[T any](name string) (*ParsedJobLog[T], error) {
	row := db.QueryRow(query_get_last_job_log, name)
	jl := JobLog{}
	if err := row.Scan(
		&jl.Name,
		&jl.Id,
		&jl.Status,
		&jl.Data,
		&jl.Error,
		&jl.CreatedAt,
		&jl.UpdatedAt,
		&jl.ExpiresAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if !jl.ExpiresAt.IsZero() && jl.ExpiresAt.Before(time.Now()) {
		_ = DeleteJobLog(name, jl.Id)
		return nil, nil
	}

	if jl.Id == "" {
		_ = DeleteJobLog(name, jl.Id)
		return nil, nil
	}

	pjl := &ParsedJobLog[T]{
		Name:      jl.Name,
		Id:        jl.Id,
		Status:    jl.Status,
		Error:     jl.Error,
		CreatedAt: jl.CreatedAt.Time,
		UpdatedAt: jl.UpdatedAt.Time,
	}

	if !jl.Data.IsZero() && !jl.Data.Is("null") {
		var data T
		if err := json.Unmarshal([]byte(jl.Data.String), &data); err != nil {
			return nil, err
		}
		pjl.Data = &data
	}

	return pjl, nil
}

var query_list_job_logs = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Name,
)

var query_delete_job_logs_by_id = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s IN `,
	TableName,
	Column.Name,
	Column.Id,
)

func GetAllJobLogs[T any](name string) ([]ParsedJobLog[T], error) {
	rows, err := db.Query(query_list_job_logs, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	idsToDelete := []string{}
	results := []ParsedJobLog[T]{}

	for rows.Next() {
		jl := JobLog{}
		if err := rows.Scan(
			&jl.Name,
			&jl.Id,
			&jl.Status,
			&jl.Data,
			&jl.Error,
			&jl.CreatedAt,
			&jl.UpdatedAt,
			&jl.ExpiresAt,
		); err != nil {
			return nil, err
		}

		if jl.Id == "" || (!jl.ExpiresAt.IsZero() && jl.ExpiresAt.Before(now)) {
			idsToDelete = append(idsToDelete, jl.Id)
			continue
		}

		pjl := ParsedJobLog[T]{
			Name:      jl.Name,
			Id:        jl.Id,
			Status:    jl.Status,
			Error:     jl.Error,
			CreatedAt: jl.CreatedAt.Time,
			UpdatedAt: jl.UpdatedAt.Time,
		}

		if !jl.Data.IsZero() && !jl.Data.Is("null") {
			var data T
			if err := json.Unmarshal([]byte(jl.Data.String), &data); err != nil {
				return nil, err
			}
			pjl.Data = &data
		}

		results = append(results, pjl)
	}

	if len(idsToDelete) > 0 {
		query := query_delete_job_logs_by_id + "(" + util.RepeatJoin("?", len(idsToDelete), ",") + ")"
		args := make([]any, 1+len(idsToDelete))
		args[0] = name
		for i, id := range idsToDelete {
			args[i+1] = id
		}
		if _, err := db.Exec(query, args...); err != nil {
			log.Error("failed to delete expired job logs", "error", err)
		}
	}

	return results, nil
}

var query_save_job_log = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s, %s) DO UPDATE SET %s`,
	TableName,
	strings.Join([]string{
		Column.Name,
		Column.Id,
		Column.Status,
		Column.Data,
		Column.Error,
		Column.ExpiresAt,
	}, ","),
	util.RepeatJoin("?", 6, ","),
	Column.Name,
	Column.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Status, Column.Status),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Data, Column.Data),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Error, Column.Error),
		fmt.Sprintf(`%s = %s`, Column.UpdatedAt, db.CurrentTimestamp),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.ExpiresAt, Column.ExpiresAt),
	}, ", "),
)

func SaveJobLog[T any](name string, id string, status string, data *T, errorMsg string, expiresIn time.Duration) error {
	if id == "" {
		return fmt.Errorf("job id cannot be empty")
	}

	var dataBlob []byte
	var err error
	if data != nil {
		dataBlob, err = json.Marshal(data)
		if err != nil {
			return err
		}
	}

	expiresAt := db.Timestamp{}
	if expiresIn != 0 {
		expiresAt.Time = time.Now().Add(expiresIn)
	}

	_, err = db.Exec(
		query_save_job_log,
		name,
		id,
		status,
		string(dataBlob),
		errorMsg,
		expiresAt,
	)
	return err
}

var query_delete_job_log = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.Name,
	Column.Id,
)

func DeleteJobLog(name string, id string) error {
	_, err := db.Exec(query_delete_job_log, name, id)
	return err
}

var query_count_job_logs = fmt.Sprintf(
	`SELECT COUNT(1) FROM %s WHERE %s = ?`,
	TableName,
	Column.Name,
)

func CountJobLogs(name string) (int, error) {
	var count int
	row := db.QueryRow(query_count_job_logs, name)
	if err := row.Scan(&count); err != nil {
		return -1, err
	}
	return count, nil
}

var query_get_unique_job_names = fmt.Sprintf(
	`SELECT DISTINCT %s FROM %s ORDER BY %s`,
	Column.Name,
	TableName,
	Column.Name,
)

func GetUniqueJobNames() ([]string, error) {
	rows, err := db.Query(query_get_unique_job_names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return names, nil
}
