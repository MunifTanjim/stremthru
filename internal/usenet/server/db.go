package usenet_server

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/rs/xid"
)

func encrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return core.Encrypt(config.VaultSecret, value)
}

func decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return core.Decrypt(config.VaultSecret, value)
}

const TableName = "usenet_server"

type UsenetServer struct {
	Id             string
	Name           string
	Host           string
	Port           int
	Username       string
	Password       string
	TLS            bool
	TLSSkipVerify  bool
	Priority       int
	IsBackup       bool
	MaxConnections int
	Disabled       bool
	CAt            db.Timestamp
	UAt            db.Timestamp
}

func (s *UsenetServer) ProviderId() string {
	return s.Host + ":" + util.IntToString(s.Port) + ":" + s.Username
}

func NewUsenetServer(name, host string, port int, username, password string, tls, tlsSkipVerify bool, priority int, isBackup bool, maxConnections int) (*UsenetServer, error) {
	server := &UsenetServer{
		Id:             xid.New().String(),
		Name:           name,
		Host:           host,
		Port:           port,
		Username:       username,
		TLS:            tls,
		TLSSkipVerify:  tlsSkipVerify,
		Priority:       priority,
		IsBackup:       isBackup,
		MaxConnections: maxConnections,
	}
	err := server.SetPassword(password)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *UsenetServer) SetPassword(password string) error {
	encPassword, err := encrypt(password)
	if err != nil {
		return err
	}
	s.Password = encPassword
	return nil
}

func (s *UsenetServer) GetPassword() (string, error) {
	return decrypt(s.Password)
}

var Column = struct {
	Id             string
	Name           string
	Host           string
	Port           string
	Username       string
	Password       string
	TLS            string
	TLSSkipVerify  string
	Priority       string
	IsBackup       string
	MaxConnections string
	Disabled       string
	CAt            string
	UAt            string
}{
	Id:             "id",
	Name:           "name",
	Host:           "host",
	Port:           "port",
	Username:       "username",
	Password:       "password",
	TLS:            "tls",
	TLSSkipVerify:  "tls_skip_verify",
	Priority:       "priority",
	IsBackup:       "is_backup",
	MaxConnections: "max_conn",
	Disabled:       "disabled",
	CAt:            "cat",
	UAt:            "uat",
}

var columns = []string{
	Column.Id,
	Column.Name,
	Column.Host,
	Column.Port,
	Column.Username,
	Column.Password,
	Column.TLS,
	Column.TLSSkipVerify,
	Column.Priority,
	Column.IsBackup,
	Column.MaxConnections,
	Column.Disabled,
	Column.CAt,
	Column.UAt,
}

var query_upsert = fmt.Sprintf(
	`INSERT INTO %s AS us (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s`,
	TableName,
	db.JoinColumnNames(columns[0:len(columns)-2]...),
	util.RepeatJoin("?", len(columns)-2, ", "),
	Column.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Name, Column.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Host, Column.Host),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Port, Column.Port),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Username, Column.Username),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Password, Column.Password),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.TLS, Column.TLS),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.TLSSkipVerify, Column.TLSSkipVerify),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Priority, Column.Priority),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.IsBackup, Column.IsBackup),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.MaxConnections, Column.MaxConnections),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Disabled, Column.Disabled),
		fmt.Sprintf(`%s = %s`, Column.UAt, db.CurrentTimestamp),
	}, ", "),
)

func (s *UsenetServer) Upsert() error {
	_, err := db.Exec(query_upsert,
		s.Id,
		s.Name,
		s.Host,
		s.Port,
		s.Username,
		s.Password,
		s.TLS,
		s.TLSSkipVerify,
		s.Priority,
		s.IsBackup,
		s.MaxConnections,
		s.Disabled,
	)
	return err
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s ORDER BY %s ASC, %s DESC`,
	strings.Join(columns, ", "),
	TableName,
	Column.Priority,
	Column.UAt,
)

func GetAll() ([]UsenetServer, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []UsenetServer{}
	for rows.Next() {
		item := UsenetServer{}
		if err := rows.Scan(&item.Id, &item.Name, &item.Host, &item.Port, &item.Username, &item.Password, &item.TLS, &item.TLSSkipVerify, &item.Priority, &item.IsBackup, &item.MaxConnections, &item.Disabled, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

var query_get_all_enabled = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = %s ORDER BY %s ASC, %s DESC`,
	strings.Join(columns, ", "),
	TableName,
	Column.Disabled, db.BooleanFalse,
	Column.Priority,
	Column.UAt,
)

func GetAllEnabled() ([]UsenetServer, error) {
	rows, err := db.Query(query_get_all_enabled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []UsenetServer{}
	for rows.Next() {
		item := UsenetServer{}
		if err := rows.Scan(&item.Id, &item.Name, &item.Host, &item.Port, &item.Username, &item.Password, &item.TLS, &item.TLSSkipVerify, &item.Priority, &item.IsBackup, &item.MaxConnections, &item.Disabled, &item.CAt, &item.UAt); err != nil {
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

func GetById(id string) (*UsenetServer, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := UsenetServer{}
	if err := row.Scan(&item.Id, &item.Name, &item.Host, &item.Port, &item.Username, &item.Password, &item.TLS, &item.TLSSkipVerify, &item.Priority, &item.IsBackup, &item.MaxConnections, &item.Disabled, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
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

var query_set_disabled = fmt.Sprintf(
	`UPDATE %s SET %s = ?, %s = %s WHERE %s = ?`,
	TableName,
	Column.Disabled,
	Column.UAt, db.CurrentTimestamp,
	Column.Id,
)

func SetDisabled(id string, disabled bool) error {
	_, err := db.Exec(query_set_disabled, disabled, id)
	return err
}
