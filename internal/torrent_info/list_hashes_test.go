package torrent_info

import (
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHashesByStremIdForSeries(t *testing.T) {
	setupTestDB(t)

	imdbId := "tt19244304"
	seasonPack := strings.Repeat("a", 40)
	s01e01 := strings.Repeat("b", 40)
	s02Pack := strings.Repeat("c", 40)

	for _, ti := range []struct{ hash, title, seasons, episodes string }{
		{seasonPack, "IT.Welcome.to.Derry.S01.1080p.AMZN.WEB-DL", "1", ""},
		{s01e01, "IT.Welcome.to.Derry.S01E01.1080p.AMZN.WEB-DL", "1", "1"},
		{s02Pack, "IT.Welcome.to.Derry.S02.1080p.AMZN.WEB-DL", "2", ""},
	} {
		_, err := db.Exec("INSERT INTO torrent_info (hash, t_title, src, seasons, episodes) VALUES (?, ?, ?, ?, ?)", ti.hash, ti.title, TorrentInfoSourceTorrentio, ti.seasons, ti.episodes)
		require.NoError(t, err)
	}
	require.NoError(t, imdb_torrent.Insert([]imdb_torrent.IMDBTorrent{
		{TId: imdbId, Hash: seasonPack},
		{TId: imdbId, Hash: s01e01},
		{TId: imdbId, Hash: s02Pack},
	}))

	hashes, err := ListHashesByStremId(imdbId + ":1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{seasonPack, s01e01}, hashes)

	hashes, err = ListHashesByStremId(imdbId + ":1:1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{seasonPack, s01e01}, hashes)

	hashes, err = ListHashesByStremId(imdbId + ":1:2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{seasonPack}, hashes)
}
