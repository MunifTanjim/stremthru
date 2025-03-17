package stremio_userdata

import (
	"encoding/json"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
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
	return err
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
