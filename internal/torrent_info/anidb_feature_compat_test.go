package torrent_info

import (
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/stretchr/testify/require"
)

func TestAniDBHashesWithNoLocalSeasonMetadata(t *testing.T) {
	setupTestDB(t)

	tid := "993701"
	hashKnown := strings.Repeat("a", 40)
	hashUnknown := strings.Repeat("b", 40)
	for _, row := range []struct {
		hash   string
		season int
	}{
		{hashKnown, 1},
		{hashUnknown, -1},
	} {
		_, err := db.Exec(
			"INSERT INTO anidb_torrent (tid, hash, s_type, s, ep_start, ep_end, eps) VALUES (?, ?, 'ani', ?, 7, 7, '7')",
			tid, row.hash, row.season,
		)
		require.NoError(t, err)
	}
	// The anime dataset sync is not running, so anidb_title is empty.
	// A MatchLab torrent mapped with known season or -1 is still retrievable.
	hashes, err := ListHashesByStremId("anidb:" + tid + ":7")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{hashKnown, hashUnknown}, hashes)
}

func TestAniDBKnownSeasonAlsoFindsUnspecifiedSeasonMappings(t *testing.T) {
	setupTestDB(t)
	tid := "993702"
	_, err := db.Exec(
		"INSERT INTO anidb_title (tid, ttype, tlang, value, season, year) VALUES (?, 'official', 'en', 'Known Anime', '1', '2026')",
		tid,
	)
	require.NoError(t, err)
	for _, row := range []struct {
		hash   string
		season int
	}{
		{strings.Repeat("a", 40), -1},
		{strings.Repeat("b", 40), 1},
		{strings.Repeat("c", 40), 2},
	} {
		_, err = db.Exec(
			"INSERT INTO anidb_torrent (tid, hash, s_type, s, ep_start, ep_end, eps) VALUES (?, ?, 'ani', ?, 7, 7, '7')",
			tid, row.hash, row.season,
		)
		require.NoError(t, err)
	}
	hashes, err := ListHashesByStremId("anidb:" + tid + ":7")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{strings.Repeat("a", 40), strings.Repeat("b", 40)}, hashes)
}
