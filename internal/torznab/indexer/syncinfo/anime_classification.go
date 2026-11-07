package torznab_indexer_syncinfo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anime"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
)

var classifiedAnimeIMDb = cache.NewLRUCache[bool](&cache.CacheConfig{
	Name:     "torznab:anime_imdb_classification",
	Lifetime: 10 * time.Minute,
	MaxSize:  4096,
})

func verifiedMatchLabAnimeForIMDb(imdbID string, result matchlab.Response) bool {
	if !result.HasAnime() {
		return false
	}
	// A matching name with an AniDB ID is not enough: some titles are shared
	// by unrelated anime and live-action works. The IMDb identity must agree.
	for _, raw := range []*string{result.SIDStart, result.SIDEnd} {
		if raw == nil {
			continue
		}
		id, _, _, _, valid := matchlab.ParseSID(*raw)
		if valid && id == imdbID {
			return true
		}
	}
	return false
}

// classifyAnimeIMDb works with either setting of STREMTHRU_FEATURE=+anime.
// The opt-in anime feature only starts local dataset synchronization;
// MatchLab's verified identifiers and already stored file mappings remain
// valid independently. Never classify arbitrary IMDb "Animation" as anime.
func classifyAnimeIMDb(imdbID string, titles []string, season, episode int) (bool, error) {
	var cached bool
	if classifiedAnimeIMDb.Get(imdbID, &cached) {
		return cached, nil
	}

	isAnime, err := anime.IsAnimeIMDBId(imdbID)
	if err != nil {
		return false, err
	}
	if isAnime {
		_ = classifiedAnimeIMDb.Add(imdbID, true)
		return true, nil
	}

	// These file-level SID/ASID pairs were verified by MatchLab and survive
	// even when the anime dataset workers never run.
	var present int
	err = db.QueryRow(
		"SELECT 1 FROM torrent_stream WHERE (sid = ? OR sid LIKE ?) AND asid != '' LIMIT 1",
		imdbID, imdbID+":%",
	).Scan(&present)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if err == nil {
		_ = classifiedAnimeIMDb.Add(imdbID, true)
		return true, nil
	}

	if !matchlab.Enabled() {
		_ = classifiedAnimeIMDb.Add(imdbID, false)
		return false, nil
	}
	timeout := max(config.Integration.MatchLab.Timeout, 15*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for _, title := range titles {
		if title == "" {
			continue
		}
		candidate := title
		if season > 0 && episode > 0 {
			candidate = fmt.Sprintf("%s S%02dE%02d", title, season, episode)
		}
		result, err := matchlab.Default.Match(ctx, candidate)
		if err != nil {
			// Transport errors are not cached as negative classifications.
			return false, fmt.Errorf("classify IMDb anime through MatchLab: %w", err)
		}
		if verifiedMatchLabAnimeForIMDb(imdbID, result) {
			_ = classifiedAnimeIMDb.Add(imdbID, true)
			return true, nil
		}
	}
	_ = classifiedAnimeIMDb.Add(imdbID, false)
	return false, nil
}
