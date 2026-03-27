package torrent_info

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpsertQuery(t *testing.T) {
	expected_query := "INSERT INTO torrent_info AS ti (hash,t_title,size,indexer,src,category,seeders,leechers,private) VALUES " +
		"(?,?,?,?,?,?,?,?,?) " +
		"ON CONFLICT (hash) DO UPDATE SET " +
		"t_title = CASE WHEN (EXCLUDED.src = 'dht' OR (EXCLUDED.src != 'ato' AND ti.src NOT IN ('dht','tio','ad','dl','rd'))) THEN EXCLUDED.t_title ELSE ti.t_title END, " +
		"size = CASE WHEN (EXCLUDED.src = 'dht' OR (EXCLUDED.size > 0 AND ti.size < 1)) THEN EXCLUDED.size ELSE ti.size END, " +
		"indexer = CASE WHEN (EXCLUDED.indexer NOT IN ('', 'bitmagnet') AND ti.indexer != EXCLUDED.indexer) THEN EXCLUDED.indexer ELSE ti.indexer END, " +
		"src = CASE WHEN (EXCLUDED.src = 'dht' OR (EXCLUDED.src != 'ato' AND ti.src NOT IN ('dht','tio','ad','dl','rd'))) THEN EXCLUDED.src ELSE ti.src END, " +
		"category = CASE WHEN (EXCLUDED.category != '' AND ti.category = '') THEN EXCLUDED.category ELSE ti.category END, " +
		"seeders = CASE WHEN (EXCLUDED.src = 'dht' OR (EXCLUDED.seeders > 0 AND ti.seeders != EXCLUDED.seeders)) THEN EXCLUDED.seeders ELSE ti.seeders END, " +
		"leechers = CASE WHEN EXCLUDED.src = 'dht' THEN EXCLUDED.leechers ELSE ti.leechers END, " +
		"private = CASE WHEN (EXCLUDED.private = 1 AND ti.private = 0) THEN EXCLUDED.private ELSE ti.private END, " +
		"updated_at = unixepoch() " +
		"WHERE EXCLUDED.src = 'dht' OR (EXCLUDED.src != 'ato' AND ti.src NOT IN ('dht','tio','ad','dl','rd')) OR (EXCLUDED.size > 0 AND ti.size < 1) OR (EXCLUDED.indexer NOT IN ('', 'bitmagnet') AND ti.indexer != EXCLUDED.indexer) OR (EXCLUDED.category != '' AND ti.category = '') OR (EXCLUDED.seeders > 0 AND ti.seeders != EXCLUDED.seeders) OR (EXCLUDED.private = 1 AND ti.private = 0)"

	assert.Equal(t, expected_query, get_upsert_query(1))
}
