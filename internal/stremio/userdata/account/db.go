package stremio_userdata_account

import (
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "stremio_userdata_account"

type UserdataAccount struct {
	Addon     string
	Key       string
	AccountId string
	CAt       db.Timestamp
}

var Column = struct {
	Addon     string
	Key       string
	AccountId string
	CAt       string
}{
	Addon:     "addon",
	Key:       "key",
	AccountId: "account_id",
	CAt:       "cat",
}

var query_link = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
	TableName,
	strings.Join([]string{
		Column.Addon,
		Column.Key,
		Column.AccountId,
	}, ", "),
)

func Link(addon, key, accountId string) error {
	_, err := db.Exec(query_link, addon, key, accountId)
	return err
}

var query_unlink = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ? AND %s = ?`,
	TableName,
	Column.Addon,
	Column.Key,
	Column.AccountId,
)

func Unlink(addon, key, accountId string) error {
	_, err := db.Exec(query_unlink, addon, key, accountId)
	return err
}

var query_get_account_ids = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	Column.AccountId,
	TableName,
	Column.Addon,
	Column.Key,
)

func GetAccountIds(addon, key string) ([]string, error) {
	rows, err := db.Query(query_get_account_ids, addon, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

var query_unlink_all_by_userdata = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.Addon,
	Column.Key,
)

func UnlinkAllByUserdata(addon, key string) error {
	_, err := db.Exec(query_unlink_all_by_userdata, addon, key)
	return err
}

var query_unlink_all_by_account = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.AccountId,
)

func UnlinkAllByAccount(accountId string) error {
	_, err := db.Exec(query_unlink_all_by_account, accountId)
	return err
}
