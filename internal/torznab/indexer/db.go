package torznab_indexer

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

func encrypt(value string) (string, error) {
	return core.Encrypt(config.VaultSecret, value)
}

func decrypt(value string) (string, error) {
	return core.Decrypt(config.VaultSecret, value)
}

const TableName = "torznab_indexer"

type IndexerType string

const (
	IndexerTypeJackett IndexerType = "jackett"
)

func (it IndexerType) IsValid() bool {
	switch it {
	case IndexerTypeJackett:
		return true
	default:
		return false
	}
}

func ParseCompositeId(compositeId string) (IndexerType, string, error) {
	typeStr, id, ok := strings.Cut(compositeId, ":")
	if !ok {
		return "", "", fmt.Errorf("invalid composite id format: expected {type}:{id}")
	}
	indexerType := IndexerType(typeStr)
	if !indexerType.IsValid() {
		return "", "", fmt.Errorf("invalid indexer type: %s", typeStr)
	}
	return indexerType, id, nil
}

type TorznabIndexer struct {
	Type   IndexerType
	Id     string
	Name   string
	URL    string
	APIKey string
	CAt    db.Timestamp
	UAt    db.Timestamp
}

func NewTorznabIndexer(indexerType IndexerType, url, apiKey string) (*TorznabIndexer, error) {
	switch indexerType {
	case IndexerTypeJackett:
		u := jackett.TorznabURL(url)
		if err := u.Parse(); err != nil {
			return nil, fmt.Errorf("invalid torznab url: %w", err)
		}

		indexer := &TorznabIndexer{
			Type: indexerType,
			Id:   u.Encode(),
			URL:  url,
		}
		err := indexer.SetAPIKey(apiKey)
		if err != nil {
			return nil, err
		}
		return indexer, nil
	default:
		return nil, fmt.Errorf("unsupported indexer type: %s", indexerType)
	}
}

func (i *TorznabIndexer) SetAPIKey(apiKey string) error {
	encAPIKey, err := encrypt(apiKey)
	if err != nil {
		return err
	}
	i.APIKey = encAPIKey
	return nil
}

func (i *TorznabIndexer) GetAPIKey() (string, error) {
	if i.APIKey == "" {
		return "", nil
	}
	return decrypt(i.APIKey)
}

func (i *TorznabIndexer) Validate() error {
	switch i.Type {
	case IndexerTypeJackett:
		u := jackett.TorznabURL(i.URL)
		if err := u.Parse(); err != nil {
			return fmt.Errorf("invalid torznab url: %w", err)
		}

		apiKey, err := i.GetAPIKey()
		if err != nil {
			return fmt.Errorf("failed to decrypt api key: %w", err)
		}

		client := jackett.NewClient(&jackett.ClientConfig{
			BaseURL: u.BaseURL,
			APIKey:  apiKey,
		})

		torznabClient := client.GetTorznabClient(u.IndexerId)

		_, err = torznabClient.GetCaps()
		if err != nil {
			return fmt.Errorf("failed to fetch capabilities: %w", err)
		}

		if i.Name == "" {
			i.Name = jackett.GetIndexerName(u.IndexerId)
		}

		return nil
	default:
		return fmt.Errorf("unsupported indexer type: %s", i.Type)
	}
}

var Column = struct {
	Type   string
	Id     string
	Name   string
	URL    string
	APIKey string
	CAt    string
	UAt    string
}{
	Type:   "type",
	Id:     "id",
	Name:   "name",
	URL:    "url",
	APIKey: "api_key",
	CAt:    "cat",
	UAt:    "uat",
}

var columns = []string{
	Column.Type,
	Column.Id,
	Column.Name,
	Column.URL,
	Column.APIKey,
	Column.CAt,
	Column.UAt,
}

var query_exists = fmt.Sprintf(
	`SELECT 1 FROM %s`,
	TableName,
)

func Exists() bool {
	var one int
	err := db.QueryRow(query_exists).Scan(&one)
	return err == nil
}

var query_get_all = fmt.Sprintf(
	`SELECT %s FROM %s`,
	strings.Join(columns, ", "),
	TableName,
)

func GetAll() ([]TorznabIndexer, error) {
	rows, err := db.Query(query_get_all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []TorznabIndexer{}
	for rows.Next() {
		item := TorznabIndexer{}
		if err := rows.Scan(&item.Type, &item.Id, &item.Name, &item.URL, &item.APIKey, &item.CAt, &item.UAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

var query_get_by_id = fmt.Sprintf(
	`SELECT %s FROM %s WHERE %s = ? AND %s = ?`,
	strings.Join(columns, ", "),
	TableName,
	Column.Type,
	Column.Id,
)

func GetById(indexerType IndexerType, id string) (*TorznabIndexer, error) {
	row := db.QueryRow(query_get_by_id, indexerType, id)

	item := TorznabIndexer{}
	if err := row.Scan(&item.Type, &item.Id, &item.Name, &item.URL, &item.APIKey, &item.CAt, &item.UAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func GetByCompositeId(compositeId string) (*TorznabIndexer, error) {
	indexerType, id, err := ParseCompositeId(compositeId)
	if err != nil {
		return nil, err
	}
	return GetById(indexerType, id)
}

var query_upsert = fmt.Sprintf(
	`INSERT INTO %s (%s) VALUES (?,?,?,?,?) ON CONFLICT (%s, %s) DO UPDATE SET %s`,
	TableName,
	db.JoinColumnNames(
		Column.Type,
		Column.Id,
		Column.Name,
		Column.URL,
		Column.APIKey,
	),
	Column.Type,
	Column.Id,
	strings.Join([]string{
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.Name, Column.Name),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.URL, Column.URL),
		fmt.Sprintf(`%s = EXCLUDED.%s`, Column.APIKey, Column.APIKey),
		fmt.Sprintf(`%s = %s`, Column.UAt, db.CurrentTimestamp),
	}, ", "),
)

func (i *TorznabIndexer) Upsert() error {
	_, err := db.Exec(query_upsert,
		i.Type,
		i.Id,
		i.Name,
		i.URL,
		i.APIKey,
	)
	return err
}

var query_delete = fmt.Sprintf(
	`DELETE FROM %s WHERE %s = ? AND %s = ?`,
	TableName,
	Column.Type,
	Column.Id,
)

func Delete(indexerType IndexerType, id string) error {
	_, err := db.Exec(query_delete, indexerType, id)
	return err
}

func DeleteByCompositeId(compositeId string) error {
	indexerType, id, err := ParseCompositeId(compositeId)
	if err != nil {
		return err
	}
	return Delete(indexerType, id)
}
