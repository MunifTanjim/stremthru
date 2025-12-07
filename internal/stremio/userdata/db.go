package stremio_userdata

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_userdata_account "github.com/MunifTanjim/stremthru/internal/stremio/userdata/account"
)

const TableName = "stremio_userdata"

type StremioUserData[T any] struct {
	Addon    string
	Key      string
	Value    T
	Name     string
	Disabled bool
	CAt      db.Timestamp
	UAt      db.Timestamp
}

var Column = struct {
	Addon    string
	Key      string
	Value    string
	Name     string
	Disabled string
	CAt      string
	UAt      string
}{
	Addon:    "addon",
	Key:      "key",
	Value:    "value",
	Name:     "name",
	Disabled: "disabled",
	CAt:      "cat",
	UAt:      "uat",
}

func List[T any](addon string) ([]StremioUserData[T], error) {
	query := "SELECT addon, key, value, name, cat, uat FROM " + TableName + " WHERE addon = ? AND disabled = " + db.BooleanFalse
	rows, err := db.Query(query, addon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suds := []StremioUserData[T]{}
	for rows.Next() {
		sud := StremioUserData[T]{}
		var value string
		if err := rows.Scan(&sud.Addon, &sud.Key, &value, &sud.Name, &sud.CAt, &sud.UAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(value), &sud.Value); err != nil {
			return nil, err
		}
		suds = append(suds, sud)
	}
	return suds, nil
}

func GetOptions(addon string) ([]configure.ConfigOption, error) {
	query := "SELECT key, name FROM " + TableName + " WHERE addon = ? AND disabled = " + db.BooleanFalse
	rows, err := db.Query(query, addon)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	options := []configure.ConfigOption{
		{
			Value: "",
			Label: "",
		},
	}
	for rows.Next() {
		option := configure.ConfigOption{}
		if err := rows.Scan(&option.Value, &option.Label); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, nil
}

func Get[T any](addon string, key string) (*StremioUserData[T], error) {
	query := "SELECT addon, key, value, name, cat, uat FROM " + TableName + " WHERE addon = ? AND key = ? AND disabled = " + db.BooleanFalse
	row := db.QueryRow(query, addon, key)

	sud := StremioUserData[T]{}
	var value string
	if err := row.Scan(&sud.Addon, &sud.Key, &value, &sud.Name, &sud.CAt, &sud.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(value), &sud.Value); err != nil {
		return nil, err
	}
	return &sud, nil
}

func Update[T any](addon, key string, value T) error {
	blob, err := json.Marshal(value)
	if err != nil {
		return err
	}
	query := "UPDATE " + TableName + " SET value = ?, uat = " + db.CurrentTimestamp + " WHERE addon = ? AND key = ? AND disabled = " + db.BooleanFalse
	_, err = db.Exec(query, string(blob), addon, key)
	return err
}

func Delete(addon, key string) error {
	query := "DELETE FROM " + TableName + " WHERE addon = ? AND key = ?"
	_, err := db.Exec(query, addon, key)
	if err != nil {
		return err
	}
	return stremio_userdata_account.UnlinkAllByUserdata(addon, key)
}

func Create[T any](addon, key, name string, value T) error {
	blob, err := json.Marshal(value)
	if err != nil {
		return err
	}
	query := "INSERT INTO " + TableName + " (addon, key, value, name) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, addon, key, string(blob), name)
	return err
}

type LinkedUserdata struct {
	Addon string
	Key   string
	Name  string
	CAt   db.Timestamp
}

var query_get_linked_userdata_by_account_id = fmt.Sprintf(
	`SELECT %s, uda.%s FROM %s uda JOIN %s ud ON uda.%s = ud.%s AND uda.%s = ud.%s WHERE uda.%s = ?`,
	db.JoinPrefixedColumnNames("ud.", Column.Addon, Column.Key, Column.Name),
	stremio_userdata_account.Column.CAt,
	stremio_userdata_account.TableName,
	TableName,
	stremio_userdata_account.Column.Addon,
	Column.Addon,
	stremio_userdata_account.Column.Key,
	Column.Key,
	stremio_userdata_account.Column.AccountId,
)

func GetLinkedUserdataByAccountId(accountId string) ([]LinkedUserdata, error) {
	rows, err := db.Query(query_get_linked_userdata_by_account_id, accountId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LinkedUserdata{}
	for rows.Next() {
		item := LinkedUserdata{}
		if err := rows.Scan(&item.Addon, &item.Key, &item.Name, &item.CAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
