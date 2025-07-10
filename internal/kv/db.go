package kv

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/util"
)

const TableName = "kv"

type KV struct {
	Type      string
	Key       string
	Value     string // JSON Encoded Value
	CreatedAt db.Timestamp
	UpdatedAt db.Timestamp
	ExpiresAt db.Timestamp
}

type ParsedKV[V any] struct {
	Key       string
	Value     V
	CreatedAt time.Time
	UpdatedAt time.Time
}

type KVStoreConfig struct {
	Type      string
	GetKey    func(key string) string
	ExpiresIn time.Duration
}

type KVStore[V any] interface {
	GetValue(key string, value *V) error
	GetLast() (*ParsedKV[V], error)
	List() ([]ParsedKV[V], error)
	Count() (int, error)
	Set(key string, value V) error
	Del(key string) error

	WithScope(scope string) KVStore[V]
}

type SQLKVStore[V any] struct {
	t         string
	getKey    func(key string) string
	expiresIn time.Duration
}

func (kv *SQLKVStore[V]) GetValue(key string, value *V) error {
	var val string
	var expAt db.Timestamp
	query := "SELECT v, eat FROM " + TableName + " WHERE t = ? AND k = ?"
	row := db.QueryRow(query, kv.t, kv.getKey(key))
	if err := row.Scan(&val, &expAt); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if !expAt.IsZero() && expAt.Before(time.Now()) {
		return kv.Del(key)
	}
	return json.Unmarshal([]byte(val), &value)
}

func (kv *SQLKVStore[V]) GetLast() (*ParsedKV[V], error) {
	pkv := ParsedKV[V]{}
	var val string
	var expAt db.Timestamp
	query := "SELECT k, v, cat, uat, eat FROM " + TableName + " WHERE t = ? ORDER BY cat DESC LIMIT 1"
	row := db.QueryRow(query, kv.t)
	if err := row.Scan(&pkv.Key, &val, &pkv.CreatedAt, &pkv.UpdatedAt, &expAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if !expAt.IsZero() && expAt.Before(time.Now()) {
		return nil, kv.Del(pkv.Key)
	}
	return &pkv, json.Unmarshal([]byte(val), &pkv.Value)
}

func (kv *SQLKVStore[V]) List() ([]ParsedKV[V], error) {
	if kv.t == "" {
		return nil, errors.New("missing kv type value")
	}
	query := "SELECT k, v, cat, uat, eat FROM " + TableName + " WHERE t = ?"
	rows, err := db.Query(query, kv.t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	keysToDelete := []string{}
	vs := []ParsedKV[V]{}
	for rows.Next() {
		kv := KV{}
		if err := rows.Scan(&kv.Key, &kv.Value, &kv.CreatedAt, &kv.UpdatedAt, &kv.ExpiresAt); err != nil {
			return nil, err
		}
		if !kv.ExpiresAt.IsZero() && kv.ExpiresAt.Before(now) {
			keysToDelete = append(keysToDelete, kv.Key)
			continue
		}
		var val V
		if err := json.Unmarshal([]byte(kv.Value), &val); err != nil {
			return nil, err
		}
		vs = append(vs, ParsedKV[V]{
			Key:       kv.Key,
			Value:     val,
			CreatedAt: kv.CreatedAt.Time,
			UpdatedAt: kv.UpdatedAt.Time,
		})
	}

	if expiredKeysCount := len(keysToDelete); expiredKeysCount > 0 {
		query := "DELETE FROM " + TableName + " WHERE t = ? AND k IN (" + util.RepeatJoin("?", expiredKeysCount, ",") + ")"
		args := make([]any, 1+expiredKeysCount)
		args[0] = kv.t
		for i, key := range keysToDelete {
			args[i+1] = kv.getKey(key)
		}
		if _, err := db.Exec(query, args...); err != nil {
			return vs, err
		}
	}

	return vs, nil
}

func (kv *SQLKVStore[V]) Count() (int, error) {
	if kv.t == "" {
		return -1, errors.New("missing kv type value")
	}
	query := "SELECT COUNT(t) FROM " + TableName + " WHERE t = ?"
	var val int
	row := db.QueryRow(query, kv.t)
	if err := row.Scan(&val); err != nil {
		return -1, err
	}
	return val, nil
}

func (kv *SQLKVStore[V]) Set(key string, value V) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}
	query := "INSERT INTO " + TableName + " (t,k,v,eat) VALUES (?,?,?,?) ON CONFLICT (t,k) DO UPDATE SET v = EXCLUDED.v, eat = EXCLUDED.eat, uat = " + db.CurrentTimestamp
	expiresAt := db.Timestamp{}
	if kv.expiresIn != 0 {
		expiresAt.Time = time.Now().Add(kv.expiresIn)
	}
	_, err = db.Exec(query, kv.t, kv.getKey(key), val, expiresAt)
	return err
}

func (kv *SQLKVStore[V]) Del(key string) error {
	query := "DELETE FROM " + TableName + " WHERE t = ? AND k = ?"
	_, err := db.Exec(query, kv.t, kv.getKey(key))
	return err
}

func (kv *SQLKVStore[V]) WithScope(scope string) KVStore[V] {
	if scope == "" {
		return kv
	}
	skv := *kv
	skv.t = skv.t + ":" + strings.ToLower(scope)
	return &skv
}

func NewKVStore[V any](config *KVStoreConfig) KVStore[V] {
	if config.Type != "" && config.GetKey == nil {
		config.GetKey = func(key string) string {
			return key
		}
	}

	inputKey := "key"
	outputKey := config.GetKey(inputKey)
	if config.Type == "" && outputKey == inputKey {
		panic("GetKey ouput is same as input, when type is missing")
	}
	if !strings.Contains(outputKey, inputKey) {
		panic("GetKey output does not contain input")
	}
	return &SQLKVStore[V]{
		t:         strings.ToLower(config.Type),
		getKey:    config.GetKey,
		expiresIn: config.ExpiresIn,
	}
}
