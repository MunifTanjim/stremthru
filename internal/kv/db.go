package kv

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "kv"

type KV struct {
	Key       string
	Value     string // JSON Encoded Value
	CreatedAt db.Timestamp
	UpdatedAt db.Timestamp
}

type KVStoreConfig struct {
	GetKey func(key string) string
}

type KVStore[V any] interface {
	Get(key string, value *V) error
	Set(key string, value V) error
	Del(key string) error
}

type SQLKVStore[V any] struct {
	getKey func(key string) string
}

func (kv SQLKVStore[V]) Get(key string, value *V) error {
	var val string
	row := db.QueryRow("SELECT v FROM "+TableName+" WHERE k = ?", kv.getKey(key))
	if err := row.Scan(&val); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return json.Unmarshal([]byte(val), &value)
}

func (kv SQLKVStore[V]) Set(key string, value V) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO "+TableName+" (k, v) VALUES (?, ?) ON CONFLICT (k) DO UPDATE SET v = EXCLUDED.v, uat = "+db.CurrentTimestamp, kv.getKey(key), val)
	return err
}

func (kv SQLKVStore[V]) Del(key string) error {
	_, err := db.Exec("DELETE FROM "+TableName+" WHERE k = ?", kv.getKey(key))
	return err
}

func NewKVStore[V any](config *KVStoreConfig) *SQLKVStore[V] {
	inputKey := "key"
	outputKey := config.GetKey(inputKey)
	if outputKey == inputKey {
		panic("GetKey ouput is same as input")
	}
	if !strings.Contains(outputKey, inputKey) {
		panic("GetKey output does not contain input")
	}
	return &SQLKVStore[V]{
		getKey: config.GetKey,
	}
}
