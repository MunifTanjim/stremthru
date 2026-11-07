package torznab_indexer_syncinfo

import (
	"database/sql"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func animePointer(s string) *string { return &s }

func TestVerifiedMatchLabAnimeChecksIMDbIdentity(t *testing.T) {
	tests := []struct {
		name     string
		result   matchlab.Response
		expected bool
	}{
		{name: "matched anime", result: matchlab.Response{SIDStart: animePointer("tt1234567:1:2"), ASIDStart: animePointer("123:2")}, expected: true},
		{name: "other IMDb title", result: matchlab.Response{SIDStart: animePointer("tt9999999:1:2"), ASIDStart: animePointer("123:2")}},
		{name: "no AniDB identity", result: matchlab.Response{SIDStart: animePointer("tt1234567")}},
		{name: "no IMDb identity", result: matchlab.Response{ASIDStart: animePointer("123:2")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, verifiedMatchLabAnimeForIMDb("tt1234567", tc.result))
		})
	}
}

func TestClassifyAnimeIMDbFromMatchedFileWithoutAnimeDataset(t *testing.T) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	goose.SetLogger(goose.NopLogger())
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.Up(sqlDB, "../../../../migrations/sqlite"))
	db.SetForTesting(sqlDB, t.Cleanup)

	// Exactly the SID/ASID pair that PR #1 writes from MatchLab. The anime
	// feature and its dataset-sync workers are not involved in this lookup.
	_, err = db.Exec(
		"INSERT INTO torrent_stream (h, p, sid, asid) VALUES (?, ?, ?, ?)",
		"abcdef0123456789abcdef0123456789abcdef01", "/Episode.mkv",
		"tt88776655:1:2", "4321:2",
	)
	require.NoError(t, err)
	isAnime, err := classifyAnimeIMDb("tt88776655", []string{"Unknown"}, 1, 2)
	require.NoError(t, err)
	require.True(t, isAnime)
}
