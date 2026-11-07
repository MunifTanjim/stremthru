package torrent_info

import (
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/db"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/stretchr/testify/require"
)

func TestFastTorzLookupOnlyFetchesMappedFileAndFeedColumns(t *testing.T) {
	setupTestDB(t)

	hash := strings.Repeat("a", 40)
	movie := TorrentInfoInsertData{
		Hash:         hash,
		TorrentTitle: "Example Show S01E02",
		Size:         1024,
		Category:     TorrentInfoCategorySeries,
		Seeders:      10,
		Leechers:     5,
		Source:       TorrentInfoSourceTorrentio,
		Files: ts.Files{
			{Idx: 0, Path: "/Example.Show.S01E01.mkv", Size: 512},
			{Idx: 1, Path: "/Example.Show.S01E02.mkv", Size: 512},
		},
	}
	require.NoError(t, Upsert([]TorrentInfoInsertData{movie}, TorrentInfoCategorySeries, true))
	ts.TagStremId(hash, "/Example.Show.S01E02.mkv", "tt1234567:1:2")
	// Indexes are intentionally unknown after source ingestion. Simulate a
	// resolved file index as supplied later by the metadata/availability worker.
	_, err := db.Exec("UPDATE torrent_stream SET i = ? WHERE h = ? AND p = ?", 1, hash, "/Example.Show.S01E02.mkv")
	require.NoError(t, err)

	fastHashes, err := ListHashesByStremIdFast("tt1234567:1:2")
	require.NoError(t, err)
	require.Equal(t, []string{hash}, fastHashes)

	matched, err := ts.GetMatchingFilesByHashes([]string{hash}, "tt1234567:1:2")
	require.NoError(t, err)
	require.Contains(t, matched, hash)
	require.Equal(t, 1, matched[hash].Idx)
	require.Equal(t, "Example.Show.S01E02.mkv", matched[hash].Name)

	absent, err := ts.GetMatchingFilesByHashes([]string{hash}, "tt1234567:1:99")
	require.NoError(t, err)
	require.Empty(t, absent)

	present, err := ts.GetHashesWithFiles([]string{hash})
	require.NoError(t, err)
	require.True(t, present[hash])

	feed, err := GetForTorznab([]string{hash})
	require.NoError(t, err)
	require.Contains(t, feed, hash)
	require.Equal(t, movie.TorrentTitle, feed[hash].TorrentTitle)
	require.Equal(t, movie.Seeders, feed[hash].Seeders)
	require.Equal(t, movie.Leechers, feed[hash].Leechers)
}
