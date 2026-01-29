package peer_token

import (
	"database/sql"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/db"
)

const TableName = "peer_token"

type PeerToken struct {
	Id        string
	Name      string
	CreatedAt db.Timestamp
}

var peerTokenCache = cache.NewLRUCache[bool](&cache.CacheConfig{
	Lifetime: 15 * time.Minute,
	Name:     "peer_token_is_valid",
	MaxSize:  512,
})

func IsValid(token string) (isValid bool, err error) {
	if token == "" {
		return false, nil
	}

	if peerTokenCache.Get(token, &isValid) {
		return isValid, nil
	}

	id := ""
	row := db.QueryRow("SELECT id FROM "+TableName+" WHERE id = ?", token)
	if err := row.Scan(&id); err != nil && err != sql.ErrNoRows {
		return false, err
	}

	isValid = id == token

	if err := peerTokenCache.Add(token, isValid); err != nil {
		return false, err
	}

	return isValid, nil
}
