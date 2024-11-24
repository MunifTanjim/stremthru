package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const TableName = "store_magnet_cache"

type StoreMagnetCacheStore string

const (
	StoreMagnetCacheStoreRealDebrid StoreMagnetCacheStore = "rd"
)

type StoreMagnetCacheFile struct {
	Idx  int    `json:"i"`
	Name string `json:"n"`
	Size int    `json:"s"`
}

type StoreMagnetCacheFiles []StoreMagnetCacheFile

func (arr StoreMagnetCacheFiles) Value() (driver.Value, error) {
	return json.Marshal(arr)
}

func (arr *StoreMagnetCacheFiles) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to convert value to []byte")
	}
	return json.Unmarshal(bytes, arr)
}

type StoreMagnetCache struct {
	Store     string
	Hash      string
	Files     StoreMagnetCacheFiles
	TouchedAt time.Time
}

func (sms *StoreMagnetCache) IsCached() bool {
	return time.Since(sms.TouchedAt) < 5*24*time.Hour
}

func GetStoreMagnetCache(store StoreMagnetCacheStore, hash string) (StoreMagnetCache, error) {
	row := db.QueryRow("SELECT store, hash, files, touched_at FROM "+TableName+" WHERE store = ? AND hash = ?", store, hash)
	smc := StoreMagnetCache{}
	err := row.Scan(&smc.Store, &smc.Hash, &smc.Files, &smc.TouchedAt)
	return smc, err
}

func GetStoreMagnetCaches(store StoreMagnetCacheStore, hashes []string) ([]StoreMagnetCache, error) {
	args := make([]interface{}, len(hashes)+1)
	args[0] = store

	hashPlaceholders := make([]string, len(hashes))
	for i, hash := range hashes {
		hashPlaceholders[i] = "?"
		args[i+1] = hash
	}

	rows, err := db.Query("SELECT store, hash, files, touched_at FROM "+TableName+" WHERE store = ? AND hash IN ("+strings.Join(hashPlaceholders, ",")+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	smcs := []StoreMagnetCache{}
	for rows.Next() {
		smc := StoreMagnetCache{}
		if err := rows.Scan(&smc.Store, &smc.Hash, &smc.Files, &smc.TouchedAt); err != nil {
			return nil, err
		}
		smcs = append(smcs, smc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return smcs, nil
}

func TouchStoreMagnetCache(store StoreMagnetCacheStore, hash string, files StoreMagnetCacheFiles) error {
	result, err := db.Exec("INSERT INTO "+TableName+" (store, hash, files) VALUES (?, ?, ?) ON CONFLICT (store, hash) DO UPDATE SET files = excluded.files, touched_at = current_timestamp", store, hash, files)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
