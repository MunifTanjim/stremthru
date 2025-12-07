package stremio_account

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	stremio_api "github.com/MunifTanjim/stremthru/internal/stremio/api"
	stremio_userdata_account "github.com/MunifTanjim/stremthru/internal/stremio/userdata/account"
)

var stremioClient = stremio_api.NewClient(&stremio_api.ClientConfig{})

func encrypt(value string) (string, error) {
	return core.Encrypt(config.VaultSecret, value)
}

func decrypt(value string) (string, error) {
	return core.Decrypt(config.VaultSecret, value)
}

const TableName = "stremio_account"

type StremioAccount struct {
	Id       string
	Email    string
	Password string
	Token    string
	TokenEAt db.Timestamp
	CAt      db.Timestamp
	UAt      db.Timestamp
}

func NewStremioAccount(email, password string) (*StremioAccount, error) {
	account := &StremioAccount{
		Email: email,
	}
	err := account.SetPassword(password)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (a *StremioAccount) SetPassword(password string) error {
	encPassword, err := encrypt(password)
	if err != nil {
		return err
	}
	a.Password = encPassword
	return nil
}

func (s *StremioAccount) IsTokenValid() bool {
	if s.Token == "" {
		return false
	}
	if s.TokenEAt.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).Before(s.TokenEAt.Time)
}

func (s *StremioAccount) DecryptPassword() (string, error) {
	return decrypt(s.Password)
}

func (s *StremioAccount) DecryptToken() (string, error) {
	if s.Token == "" {
		return "", nil
	}
	return decrypt(s.Token)
}

func (a *StremioAccount) Refresh(force bool) error {
	if !force && a.Id != "" && a.IsTokenValid() {
		return nil
	}

	shouldUpsert := false

	if a.Token != "" {
		token, err := a.DecryptToken()
		if err != nil {
			return err
		}
		params := &stremio_api.GetUserParams{}
		params.APIKey = token
		_, err = stremioClient.GetUser(params)
		if err != nil {
			rerr, ok := err.(*stremio_api.ResponseError)
			if !ok {
				return err
			}
			switch rerr.Code {
			case stremio_api.ErrorCodeSessionNotFound:
				a.Token = ""
				a.TokenEAt = db.Timestamp{}
			default:
				return err
			}
		}
		a.TokenEAt = db.Timestamp{Time: time.Now().Add(7 * 24 * time.Hour)}

		shouldUpsert = true
	}

	if force || a.Token == "" {
		password, err := a.DecryptPassword()
		if err != nil {
			return err
		}
		res, err := stremioClient.Login(&stremio_api.LoginParams{
			Email:    a.Email,
			Password: password,
		})
		if err != nil {
			if rerr, ok := err.(*stremio_api.ResponseError); ok {
				switch rerr.Code {
				case stremio_api.ErrorCodeUserNotFound, stremio_api.ErrorCodeWrongPassphrase:
					return ErrorInvalidCredentials
				}
			}
			return err
		}

		a.Id = res.Data.User.Id
		a.Email = res.Data.User.Email

		token, err := encrypt(res.Data.AuthKey)
		if err != nil {
			return err
		}
		a.Token = token
		a.TokenEAt = db.Timestamp{Time: time.Now().Add(7 * 24 * time.Hour)}

		shouldUpsert = true
	}

	if shouldUpsert {
		return a.Upsert()
	}

	return nil
}

var query_upsert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?,?,?) ON CONFLICT (%s) DO UPDATE SET %s`,
	TableName,
	db.JoinColumnNames(
		Column.Id,
		Column.Email,
		Column.Password,
		Column.Token,
		Column.TokenEAt,
	),
	Column.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Email, Column.Email),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Password, Column.Password),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Token, Column.Token),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.TokenEAt, Column.TokenEAt),
		fmt.Sprintf(`%s = %s`, Column.UAt, db.CurrentTimestamp),
	}, ", "),
)

func (s *StremioAccount) GetValidToken() (string, error) {
	err := s.Refresh(false)
	if err != nil {
		return "", err
	}
	return s.DecryptToken()
}

func (a *StremioAccount) Upsert() error {
	_, err := db.Exec(query_upsert,
		a.Id,
		a.Email,
		a.Password,
		a.Token,
		a.TokenEAt,
	)
	if a.CAt.IsZero() {
		a.CAt = db.Timestamp{Time: time.Now()}
		a.UAt = db.Timestamp{Time: a.CAt.Time}
	} else {
		a.UAt = db.Timestamp{Time: time.Now()}
	}
	return err
}

var Column = struct {
	Id       string
	Email    string
	Password string
	Token    string
	TokenEAt string
	CAt      string
	UAt      string
}{
	Id:       "id",
	Email:    "email",
	Password: "password",
	Token:    "token",
	TokenEAt: "token_eat",
	CAt:      "cat",
	UAt:      "uat",
}

var columns = []string{
	Column.Id,
	Column.Email,
	Column.Password,
	Column.Token,
	Column.TokenEAt,
	Column.CAt,
	Column.UAt,
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s`,
	strings.Join(columns, ", "),
	TableName,
)

func GetAll() ([]StremioAccount, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []StremioAccount{}
	for rows.Next() {
		item := StremioAccount{}
		if err := rows.Scan(&item.Id, &item.Email, &item.Password, &item.Token, &item.TokenEAt, &item.CAt, &item.UAt); err != nil {
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

func GetById(id string) (*StremioAccount, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := StremioAccount{}
	if err := row.Scan(&item.Id, &item.Email, &item.Password, &item.Token, &item.TokenEAt, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_get_by_email = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Email,
)

func GetByEmail(email string) (*StremioAccount, error) {
	row := db.QueryRow(query_get_by_email, email)

	item := StremioAccount{}
	if err := row.Scan(&item.Id, &item.Email, &item.Password, &item.Token, &item.TokenEAt, &item.CAt, &item.UAt); err != nil {
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
	if err != nil {
		return err
	}
	return stremio_userdata_account.UnlinkAllByAccount(id)
}
