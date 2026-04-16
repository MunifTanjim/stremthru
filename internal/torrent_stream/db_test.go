package torrent_stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordQuery(t *testing.T) {
	expected_query := `INSERT INTO torrent_stream AS ts ("h","p","i","s","sid","asid","src","vhash","mi") VALUES ` +
		"(?,?,?,?,?,?,?,?,?) " +
		"ON CONFLICT (h,p) DO UPDATE SET " +
		"i = CASE WHEN (ts.src NOT IN ('dht','tor') AND EXCLUDED.i != -1 AND ts.i != EXCLUDED.i) THEN EXCLUDED.i ELSE ts.i END, " +
		"s = CASE WHEN (ts.src NOT IN ('dht','tor') AND EXCLUDED.s != -1 AND ts.s != EXCLUDED.s) THEN EXCLUDED.s ELSE ts.s END, " +
		"sid = CASE WHEN (EXCLUDED.sid NOT IN ('', '*') AND ts.sid != EXCLUDED.sid) THEN EXCLUDED.sid ELSE ts.sid END, " +
		"asid = CASE WHEN (EXCLUDED.asid != '' AND ts.asid != EXCLUDED.asid) THEN EXCLUDED.asid ELSE ts.asid END, " +
		"vhash = CASE WHEN (EXCLUDED.vhash != '' AND ts.vhash = '') THEN EXCLUDED.vhash ELSE ts.vhash END, " +
		"mi = CASE WHEN (EXCLUDED.mi IS NOT NULL AND (ts.mi IS NULL OR (ts.mi->>'src' IS NOT NULL AND EXCLUDED.mi->>'src' IS NULL))) THEN EXCLUDED.mi ELSE ts.mi END, " +
		"src = CASE WHEN ((EXCLUDED.src IN ('dht','tor') OR ts.src NOT IN ('dht','tor')) AND (EXCLUDED.src != 'mfn' OR ts.src = 'mfn') AND EXCLUDED.src != '') THEN EXCLUDED.src ELSE ts.src END, " +
		"uat = unixepoch() " +
		"WHERE (ts.src NOT IN ('dht','tor') AND EXCLUDED.i != -1 AND ts.i != EXCLUDED.i) OR (ts.src NOT IN ('dht','tor') AND EXCLUDED.s != -1 AND ts.s != EXCLUDED.s) OR (EXCLUDED.sid NOT IN ('', '*') AND ts.sid != EXCLUDED.sid) OR (EXCLUDED.asid != '' AND ts.asid != EXCLUDED.asid) OR (EXCLUDED.vhash != '' AND ts.vhash = '') OR (EXCLUDED.mi IS NOT NULL AND (ts.mi IS NULL OR (ts.mi->>'src' IS NOT NULL AND EXCLUDED.mi->>'src' IS NULL))) OR ((EXCLUDED.src IN ('dht','tor') OR ts.src NOT IN ('dht','tor')) AND (EXCLUDED.src != 'mfn' OR ts.src = 'mfn') AND EXCLUDED.src != '')"

	assert.Equal(t, expected_query, get_record_query(1))
}
