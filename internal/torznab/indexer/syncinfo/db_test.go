package torznab_indexer_syncinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSyncPendingQuery(t *testing.T) {
	t.Run("without excluded indexer ids", func(t *testing.T) {
		expected := `SELECT "indexer_id","sid","queued_at","synced_at","error","result_count","status","queries" FROM torznab_indexer_syncinfo WHERE ` +
			`queued_at IS NOT NULL AND (synced_at IS NULL OR (queued_at > synced_at AND synced_at <= ?))` +
			` ORDER BY queued_at ASC LIMIT 1000`

		assert.Equal(t, expected, get_sync_pending_query(0))
	})

	t.Run("with excluded indexer ids", func(t *testing.T) {
		expected := `SELECT "indexer_id","sid","queued_at","synced_at","error","result_count","status","queries" FROM torznab_indexer_syncinfo WHERE ` +
			`queued_at IS NOT NULL AND (synced_at IS NULL OR (queued_at > synced_at AND synced_at <= ?))` +
			` AND indexer_id NOT IN (?,?,?)` +
			` ORDER BY queued_at ASC LIMIT 1000`

		assert.Equal(t, expected, get_sync_pending_query(3))
	})
}
