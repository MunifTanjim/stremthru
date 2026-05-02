package nzb_info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashNZBFileLink(t *testing.T) {
	t.Run("api path with different apikeys produce same hash", func(t *testing.T) {
		hash1 := HashNZBFileLink("http://indexer.com/api?t=get&id=abc123&apikey=secret1")
		hash2 := HashNZBFileLink("http://indexer.com/api?t=get&id=abc123&apikey=secret2")
		assert.Equal(t, hash1, hash2)
	})

	t.Run("api path different ids produce different hashes", func(t *testing.T) {
		hash1 := HashNZBFileLink("http://indexer.com/api?t=get&id=abc123&apikey=xxx")
		hash2 := HashNZBFileLink("http://indexer.com/api?t=get&id=def456&apikey=xxx")
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("non-api url query stripped with ?", func(t *testing.T) {
		hash1 := HashNZBFileLink("http://indexer.com/getnzb/abc123")
		hash2 := HashNZBFileLink("http://indexer.com/getnzb/abc123?apikey=secret")
		assert.Equal(t, hash1, hash2)
	})

	t.Run("non-api url query stripped with &", func(t *testing.T) {
		hash1 := HashNZBFileLink("http://indexer.com/getnzb/abc123")
		hash2 := HashNZBFileLink("http://indexer.com/getnzb/abc123&apikey=secret")
		assert.Equal(t, hash1, hash2)
	})

	t.Run("different paths produce different hashes", func(t *testing.T) {
		hash1 := HashNZBFileLink("http://indexer.com/getnzb/abc123")
		hash2 := HashNZBFileLink("http://indexer.com/getnzb/def456")
		assert.NotEqual(t, hash1, hash2)
	})
}
