package torbox

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	goose.SetLogger(goose.NopLogger())
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(sqlDB, "../../migrations/sqlite"))

	db.SetForTesting(sqlDB, t.Cleanup)
}

func queryIsCached(hash string) (bool, error) {
	var isCached bool
	err := db.QueryRow(
		"SELECT is_cached FROM magnet_cache WHERE store = ? AND hash = ?",
		store.StoreNameTorBox.Code(), hash,
	).Scan(&isCached)
	return isCached, err
}

// A store check must remember the hashes TorBox does not have, so the next
// check can answer from magnet_cache instead of asking TorBox again.
func TestCheckMagnetRecordsNotCachedHashes(t *testing.T) {
	setupTestDB(t)

	cachedHash := strings.Repeat("a", 40)
	missingHash := strings.Repeat("b", 40)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"detail":"ok","data":[{"hash":"` + cachedHash + `","name":"Some Movie","size":10,"files":[{"name":"Some Movie/movie.mkv","size":10}]}]}`))
	}))
	defer srv.Close()

	c := NewStoreClient(&StoreClientConfig{HTTPClient: srv.Client()})
	c.client.BaseURL, _ = url.Parse(srv.URL)

	data, err := c.CheckMagnet(&store.CheckMagnetParams{
		Magnets: []string{cachedHash, missingHash},
	})
	require.NoError(t, err)
	require.Len(t, data.Items, 2)
	assert.Equal(t, store.MagnetStatusCached, data.Items[0].Status)
	assert.Equal(t, store.MagnetStatusUnknown, data.Items[1].Status)

	// BulkTrackMagnet runs in a goroutine.
	require.Eventually(t, func() bool {
		_, err := queryIsCached(missingHash)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "no magnet_cache row for the hash TorBox does not have")

	isCached, err := queryIsCached(cachedHash)
	require.NoError(t, err)
	assert.True(t, isCached)

	isCached, err = queryIsCached(missingHash)
	require.NoError(t, err)
	assert.False(t, isCached)
}
