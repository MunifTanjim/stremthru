package db

import (
	"database/sql"
	"time"

	"github.com/MunifTanjim/stremthru/plugin/imdb"
)

// RefreshState is what the service needs to decide freshness for a title.
type RefreshState struct {
	Type          string
	IsSeries      bool
	Running       bool   // is_series && year_end IS NULL
	ReleaseDate   string // "YYYY", "YYYY-MM-DD" or ""
	YearStart     int
	Partial       bool
	UpdatedAt     time.Time // slow part (uat)
	FastUpdatedAt time.Time // fast part (fast_uat); zero if never
	SeriesID      string    // for an episode linked to a series, else ""
	SeriesRunning bool      // the linked series is running
	// SeriesFastUpdatedAt is the linked series' fast_uat: an episode's fast
	// part is refreshed by its series.
	SeriesFastUpdatedAt time.Time
}

func unixTime(sec int64) time.Time {
	if sec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

// GetRefreshState returns the title's refresh state, or ErrNotFound.
func (db *DB) GetRefreshState(id string) (*RefreshState, error) {
	return getRefreshState(db.DB, id)
}

// GetTitleAndState reads the title and its refresh state in one read
// transaction, so the state matches the data. ErrNotFound if no row.
func (db *DB) GetTitleAndState(id string) (*FullTitle, *RefreshState, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	st, err := getRefreshState(tx, id)
	if err != nil {
		return nil, nil, err
	}
	ft, err := getFullTitle(tx, id)
	if err != nil {
		return nil, nil, err
	}
	return ft, st, tx.Commit()
}

func getRefreshState(db queryer, id string) (*RefreshState, error) {
	var st RefreshState
	var isSeries, partial int
	var yearEnd sql.NullInt64
	var uat, fastUAT int64
	err := db.QueryRow(
		`SELECT type, is_series, year_end, release_date, year_start, partial, uat, fast_uat FROM title WHERE id = ?`, id,
	).Scan(&st.Type, &isSeries, &yearEnd, &st.ReleaseDate, &st.YearStart, &partial, &uat, &fastUAT)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	st.IsSeries = isSeries != 0
	st.Running = st.IsSeries && !yearEnd.Valid
	st.Partial = partial != 0
	st.UpdatedAt = time.Unix(uat, 0).UTC()
	st.FastUpdatedAt = unixTime(fastUAT)

	var seriesIsSeries int
	var seriesYearEnd sql.NullInt64
	var seriesFastUAT int64
	err = db.QueryRow(
		`SELECT s.id, s.is_series, s.year_end, s.fast_uat FROM title_episode te JOIN title s ON s.id = te.title_id WHERE te.id = ?`, id,
	).Scan(&st.SeriesID, &seriesIsSeries, &seriesYearEnd, &seriesFastUAT)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	st.SeriesFastUpdatedAt = unixTime(seriesFastUAT)
	st.SeriesRunning = st.SeriesID != "" && seriesIsSeries != 0 && !seriesYearEnd.Valid
	return &st, nil
}

// EpisodeRef is one entry of a series' episode list.
type EpisodeRef struct {
	ID      string
	Season  int
	Episode *int
	Plot    string
}

// FastRefresh is the result of refreshing a title's fast part.
type FastRefresh struct {
	ID       string
	Series   bool // true: Episodes is the full current list
	Episodes []EpisodeRef
	Details  []TitleSummary // fetched details for some episodes
	Ratings  []imdbplugin.Rating
}

// ApplyFastRefresh stores a fast refresh. For a series it replaces the
// episode list: new episodes are stored as partial rows, listed episodes get
// the list's plot and any fetched details, and dropped episodes that were
// only stored from the series are deleted (an episode that had its own full
// fetch keeps its row). It never touches the slow part (uat, genres, people,
// akas, trailers) or an existing row's partial flag.
func (db *DB) ApplyFastRefresh(fr *FastRefresh) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if fr.Series {
		prior, err := queryStrings(tx, `SELECT id FROM title_episode WHERE title_id = ?`, fr.ID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM title_episode WHERE title_id = ?`, fr.ID); err != nil {
			return err
		}
		keepEpisode := newEpisodeFilter(fr.ID)
		listed := map[string]bool{}
		for _, e := range fr.Episodes {
			if !keepEpisode(e.ID) {
				continue
			}
			listed[e.ID] = true
			if _, err := tx.Exec(
				`INSERT INTO title (id, type, name, partial) VALUES (?, 'tvEpisode', '', 1) ON CONFLICT (id) DO NOTHING`, e.ID,
			); err != nil {
				return err
			}
			if e.Plot != "" {
				if _, err := tx.Exec(`UPDATE title SET description = ? WHERE id = ?`, e.Plot, e.ID); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(
				`INSERT INTO title_episode (id, title_id, season, episode) VALUES (?, ?, ?, ?)
				 ON CONFLICT (id) DO UPDATE SET title_id = EXCLUDED.title_id, season = EXCLUDED.season, episode = EXCLUDED.episode`,
				e.ID, fr.ID, e.Season, nullInt(e.Episode),
			); err != nil {
				return err
			}
		}
		for _, d := range fr.Details {
			if !listed[d.ID] {
				continue
			}
			if _, err := tx.Exec(
				`UPDATE title SET name = ?, description = COALESCE(NULLIF(?, ''), description), image_id = ?, release_date = ?, runtime = ? WHERE id = ?`,
				d.Name, d.Description, d.Image.ID, d.ReleaseDate, d.Runtime, d.ID,
			); err != nil {
				return err
			}
			if err := addTitleImage(tx, fr.ID, d.ID, d.Image); err != nil {
				return err
			}
		}
		if err := deleteUnlinkedPartialEpisodes(tx, prior); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`UPDATE title SET fast_uat = unixepoch() WHERE id IN (SELECT id FROM title_episode WHERE title_id = ?)`, fr.ID,
		); err != nil {
			return err
		}
	}

	for _, r := range fr.Ratings {
		if _, err := tx.Exec(`UPDATE title SET rating = ?, votes = ? WHERE id = ?`, r.Rating, r.Votes, r.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE title SET fast_uat = unixepoch() WHERE id = ?`, fr.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListEpisodeDates returns the release date of each episode linked to the
// series, by episode id.
func (db *DB) ListEpisodeDates(seriesID string) (map[string]string, error) {
	rows, err := db.Query(
		`SELECT t.id, t.release_date FROM title_episode te JOIN title t ON t.id = te.id WHERE te.title_id = ?`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dates := map[string]string{}
	for rows.Next() {
		var id, date string
		if err := rows.Scan(&id, &date); err != nil {
			return nil, err
		}
		dates[id] = date
	}
	return dates, rows.Err()
}

// deleteUnlinkedPartialEpisodes deletes the given episode rows that were only
// stored from a series (partial) and are no longer linked to any. An episode
// that had its own full fetch keeps its row.
func deleteUnlinkedPartialEpisodes(tx *sql.Tx, ids []string) error {
	for _, id := range ids {
		if _, err := tx.Exec(
			`DELETE FROM title WHERE id = ? AND partial = 1 AND NOT EXISTS (SELECT 1 FROM title_episode WHERE id = ?)`, id, id,
		); err != nil {
			return err
		}
	}
	return nil
}

// DeleteTitle removes a title IMDb no longer has: its row and child rows, its
// episode links, the episodes that were only stored from it, its gallery, and
// the images no other title uses.
func (db *DB) DeleteTitle(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := deleteTitle(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveTitle deletes the title and records the plugin 404 in one
// transaction, so no reader sees the title gone without the tombstone.
func (db *DB) RemoveTitle(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := deleteTitle(tx, id); err != nil {
		return err
	}
	if err := markMissing(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteTitle(tx *sql.Tx, id string) error {
	partialEpisodes, err := queryStrings(tx,
		`SELECT te.id FROM title_episode te JOIN title t ON t.id = te.id WHERE te.title_id = ? AND t.partial = 1`, id)
	if err != nil {
		return err
	}
	// The title's own episodes' galleries and its stills in other titles'
	// galleries go too.
	owners := append([]string{id}, partialEpisodes...)
	var images []string
	for _, owner := range owners {
		ids, err := queryStrings(tx,
			`SELECT image_id FROM title_image WHERE title_id = ?1 OR ep_id = ?1
			 UNION SELECT image_id FROM title WHERE id = ?1`, owner)
		if err != nil {
			return err
		}
		images = append(images, ids...)
	}

	for _, owner := range owners {
		for _, table := range []string{"title_genre", "title_trailer", "title_person"} {
			if _, err := tx.Exec(`DELETE FROM `+table+` WHERE id = ?`, owner); err != nil {
				return err
			}
		}
		for _, q := range []string{
			`DELETE FROM title_aka WHERE title_id = ?`,
			`DELETE FROM title_episode WHERE title_id = ?1 OR id = ?1`,
			`DELETE FROM title_image WHERE title_id = ?1 OR ep_id = ?1`,
			`DELETE FROM title_gallery WHERE title_id = ?`,
			`DELETE FROM title WHERE id = ?`,
		} {
			if _, err := tx.Exec(q, owner); err != nil {
				return err
			}
		}
	}
	for _, imageID := range images {
		if imageID == "" {
			continue
		}
		if err := deleteUnusedImage(tx, imageID); err != nil {
			return err
		}
	}
	return nil
}
