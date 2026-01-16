package newznab_indexer

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/ratelimit"
	rrl "github.com/nccapo/rate-limiter"
)

func encrypt(value string) (string, error) {
	return core.Encrypt(config.VaultSecret, value)
}

func decrypt(value string) (string, error) {
	return core.Decrypt(config.VaultSecret, value)
}

const TableName = "newznab_indexer"

type IndexerType string

const (
	IndexerTypeGeneric IndexerType = "generic"
)

func (it IndexerType) IsValid() bool {
	switch it {
	case IndexerTypeGeneric:
		return true
	default:
		return false
	}
}

type NewznabIndexer struct {
	Id                int64
	Type              IndexerType
	Name              string
	URL               string
	APIKey            string
	RateLimitConfigId sql.NullString
	CAt               db.Timestamp
	UAt               db.Timestamp
}

type newznabIndexerRateLimiter struct {
	*ratelimit.Limiter
	prefix string
}

func (rl *newznabIndexerRateLimiter) Try() (*rrl.RateLimitResult, error) {
	return rl.Limiter.Try(rl.prefix)
}

func (rl *newznabIndexerRateLimiter) Wait() error {
	return rl.Limiter.Wait(rl.prefix)
}

func (idxr NewznabIndexer) GetRateLimiter() (*newznabIndexerRateLimiter, error) {
	if !idxr.RateLimitConfigId.Valid {
		return nil, nil
	}
	rl, err := ratelimit.NewLimiterById(idxr.RateLimitConfigId.String)
	if err != nil {
		return nil, err
	}
	return &newznabIndexerRateLimiter{
		Limiter: rl,
		prefix:  fmt.Sprintf("newznab:%d", idxr.Id),
	}, nil
}

func NewNewznabIndexer(url, apiKey string) (*NewznabIndexer, error) {
	indexer := &NewznabIndexer{
		Type: IndexerTypeGeneric,
		URL:  url,
	}
	err := indexer.SetAPIKey(apiKey)
	if err != nil {
		return nil, err
	}
	return indexer, nil
}

func (i *NewznabIndexer) SetAPIKey(apiKey string) error {
	encAPIKey, err := encrypt(apiKey)
	if err != nil {
		return err
	}
	i.APIKey = encAPIKey
	return nil
}

func (i *NewznabIndexer) GetAPIKey() (string, error) {
	if i.APIKey == "" {
		return "", nil
	}
	return decrypt(i.APIKey)
}

func (i *NewznabIndexer) Validate() error {
	apiKey, err := i.GetAPIKey()
	if err != nil {
		return fmt.Errorf("failed to decrypt api key: %w", err)
	}

	client := newznab_client.NewClient(&newznab_client.ClientConfig{
		BaseURL: i.URL,
		APIKey:  apiKey,
	})

	caps, err := client.GetCaps()
	if err != nil {
		return fmt.Errorf("failed to fetch capabilities: %w", err)
	}

	if i.Name == "" && caps.Server.Title != "" {
		i.Name = caps.Server.Title
	}

	return nil
}

var Column = struct {
	Id                string
	Type              string
	Name              string
	URL               string
	APIKey            string
	RateLimitConfigId string
	CAt               string
	UAt               string
}{
	Id:                "id",
	Type:              "type",
	Name:              "name",
	URL:               "url",
	APIKey:            "api_key",
	RateLimitConfigId: "rate_limit_config_id",
	CAt:               "cat",
	UAt:               "uat",
}

var columns = []string{
	Column.Id,
	Column.Type,
	Column.Name,
	Column.URL,
	Column.APIKey,
	Column.RateLimitConfigId,
	Column.CAt,
	Column.UAt,
}

var columnsInsert = []string{
	Column.Type,
	Column.Name,
	Column.URL,
	Column.APIKey,
	Column.RateLimitConfigId,
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s`,
	strings.Join(columns, ", "),
	TableName,
)

func GetAll() ([]NewznabIndexer, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []NewznabIndexer{}
	for rows.Next() {
		item := NewznabIndexer{}
		if err := rows.Scan(&item.Id, &item.Type, &item.Name, &item.URL, &item.APIKey, &item.RateLimitConfigId, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Id,
)

func GetById(id int64) (*NewznabIndexer, error) {
	row := db.QueryRow(query_get_by_id, id)

	item := NewznabIndexer{}
	if err := row.Scan(&item.Id, &item.Type, &item.Name, &item.URL, &item.APIKey, &item.RateLimitConfigId, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

var query_insert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?,?,?)`,
	TableName,
	db.JoinColumnNames(columnsInsert...),
)

func (i *NewznabIndexer) Insert() error {
	result, err := db.Exec(query_insert,
		i.Type,
		i.Name,
		i.URL,
		i.APIKey,
		i.RateLimitConfigId,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	i.Id = id
	return nil
}

var query_update = fmt.Sprintf(
	`UPDATE %s SET %s WHERE %s = ?`,
	TableName,
	strings.Join([]string{
		fmt.Sprintf(`%s = ?`, Column.Name),
		fmt.Sprintf(`%s = ?`, Column.URL),
		fmt.Sprintf(`%s = ?`, Column.APIKey),
		fmt.Sprintf(`%s = ?`, Column.RateLimitConfigId),
		fmt.Sprintf(`%s = %s`, Column.UAt, db.CurrentTimestamp),
	}, ", "),
	Column.Id,
)

func (i *NewznabIndexer) Update() error {
	_, err := db.Exec(query_update,
		i.Name,
		i.URL,
		i.APIKey,
		i.RateLimitConfigId,
		i.Id,
	)
	return err
}

var query_delete = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ?`,
	TableName,
	Column.Id,
)

func Delete(id int64) error {
	_, err := db.Exec(query_delete, id)
	return err
}
