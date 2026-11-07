package torrent_info

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/stretchr/testify/require"
)

func TestResetMatchLabIdentitiesForNativeMatching(t *testing.T) {
	setupTestDB(t)
	matchlabHash := strings.Repeat("b", 40)
	nativeHash := strings.Repeat("c", 40)
	items := []TorrentInfoInsertData{
		{
			Hash: matchlabHash, TorrentTitle: "Example Show S01E02",
			Source: TorrentInfoSourceTorrentio, Size: 1024,
			Files: ts.Files{{Path: "/Example.Show.S01E02.mkv", Size: 1024}},
		},
		{
			Hash: nativeHash, TorrentTitle: "Native Show S01E01",
			Source: TorrentInfoSourceTorrentio, Size: 1024,
			Files: ts.Files{{Path: "/Native.Show.S01E01.mkv", Size: 1024}},
		},
	}
	require.NoError(t, Upsert(items, TorrentInfoCategorySeries, true))
	_, err := db.Exec(
		"UPDATE torrent_info SET parser_input = t_title, parser_version = ?, matchlab_mapped_at = "+db.CurrentTimestamp+" WHERE hash = ?",
		matchlab.ParserVersion, matchlabHash,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		"UPDATE torrent_info SET parser_input = t_title, parser_version = ? WHERE hash = ?",
		9000, nativeHash,
	)
	require.NoError(t, err)
	for _, hash := range []string{matchlabHash, nativeHash} {
		_, err = db.Exec("INSERT INTO imdb_torrent (hash, tid) VALUES (?, ?)", hash, "tt1234567")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO anidb_torrent (tid, hash, s_type, s, ep_start, ep_end, eps) VALUES (?, ?, ?, ?, ?, ?, ?)", "42", hash, "ani", -1, 2, 2, "2")
		require.NoError(t, err)
		_, err = db.Exec("UPDATE torrent_stream SET sid = ?, asid = ? WHERE h = ?", "tt1234567:1:2", "42:2", hash)
		require.NoError(t, err)
	}
	unparsed, err := GetUnparsed(100)
	require.NoError(t, err)
	found := false
	for _, item := range unparsed {
		if item.Hash == matchlabHash {
			found = true
		}
	}
	require.True(t, found, "a MatchLab-parsed hash must be eligible for native reparse")

	require.NoError(t, ResetMatchLabIdentitiesForNativeMatching([]string{matchlabHash}))

	for _, table := range []string{"imdb_torrent", "anidb_torrent"} {
		var cleared int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE hash = ?", matchlabHash).Scan(&cleared))
		require.Zero(t, cleared)
		var retained int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE hash = ?", nativeHash).Scan(&retained))
		require.Equal(t, 1, retained, "native mapping must be untouched")
	}
	for _, item := range []struct {
		hash string
		want string
	}{
		{matchlabHash, ""},
		{nativeHash, "tt1234567:1:2"},
	} {
		var sid, asid string
		err = db.QueryRow("SELECT sid, asid FROM torrent_stream WHERE h = ?", item.hash).Scan(&sid, &asid)
		require.NoError(t, err)
		require.Equal(t, item.want, sid)
		if item.want == "" {
			require.Empty(t, asid)
		} else {
			require.Equal(t, "42:2", asid)
		}
	}
	var mappedAt sql.NullInt64
	require.NoError(t, db.QueryRow("SELECT matchlab_mapped_at FROM torrent_info WHERE hash = ?", matchlabHash).Scan(&mappedAt))
	require.False(t, mappedAt.Valid)

	// The local mapper must now see the hash as unmapped and may repopulate
	// the IMDb/AniDB tables independently of the external MatchLab service.
	unmapped, err := GetIMDBUnmappedHashes(20)
	require.NoError(t, err)
	require.Contains(t, unmapped, matchlabHash)
	require.NotContains(t, unmapped, nativeHash)
}
