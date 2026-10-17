package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("imdb_meta/db: title not found")

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// dedupe drops duplicate items, keeping the first occurrence. Plugin
// payloads occasionally repeat values (e.g. genres), which would violate
// the child tables' uniqueness constraints on insert.
func dedupe[T any, K comparable](items []T, key func(T) K) []T {
	seen := make(map[K]struct{}, len(items))
	out := make([]T, 0, len(items))
	for _, item := range items {
		k := key(item)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, item)
	}
	return out
}

type akaKey struct {
	text, country, language string
}

// nonBlank drops empty and whitespace-only strings.
func nonBlank(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func nullInt(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// queryer is what reads need; *sql.DB and *sql.Tx both have it, so a read
// can run inside a transaction with others.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func (db *DB) GetFullTitle(id string) (*FullTitle, error) {
	return getFullTitle(db.DB, id)
}

func getFullTitle(db queryer, id string) (*FullTitle, error) {
	ft := FullTitle{
		Title: Title{
			Genres: []string{}, TrailerVIs: []string{}, Akas: []Aka{},
			Directors: []Person{}, Cast: []Person{},
		},
		Seasons: []Season{},
	}
	var isSeries, partial int
	var yearEnd, rating sql.NullInt64
	var uat int64
	var imageID string
	err := db.QueryRow(
		`SELECT id, type, is_series, name, orig_name, year_start, year_end, release_date, release_country, description, runtime, image_id, rating, votes, certificate, partial, uat FROM title WHERE id = ?`, id,
	).Scan(
		&ft.Title.ID, &ft.Title.Type, &isSeries, &ft.Title.Name, &ft.Title.OrigName,
		&ft.Title.YearStart, &yearEnd, &ft.Title.ReleaseDate, &ft.Title.ReleaseCountry,
		&ft.Title.Description, &ft.Title.Runtime, &imageID,
		&rating, &ft.Title.Votes, &ft.Title.Certificate, &partial, &uat,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ft.Title.IsSeries = isSeries != 0
	ft.Title.Partial = partial != 0
	ft.Title.YearEnd = intPtr(yearEnd)
	ft.Title.Rating = 0
	if rating.Valid {
		ft.Title.Rating = int(rating.Int64)
	}
	ft.Title.UpdatedAt = time.Unix(uat, 0).UTC()

	// Episodes are titles too, so a tvEpisode id loads the same child tables;
	// the season and episode queries below are simply empty for it.
	rows, err := db.Query(`SELECT genre FROM title_genre WHERE id = ? ORDER BY genre`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			rows.Close()
			return nil, err
		}
		ft.Title.Genres = append(ft.Title.Genres, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`SELECT vi_id FROM title_trailer WHERE id = ? ORDER BY pos`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var vi string
		if err := rows.Scan(&vi); err != nil {
			rows.Close()
			return nil, err
		}
		ft.Title.TrailerVIs = append(ft.Title.TrailerVIs, vi)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(`SELECT text, country, language FROM title_aka WHERE title_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var aka Aka
		if err := rows.Scan(&aka.Text, &aka.Country, &aka.Language); err != nil {
			rows.Close()
			return nil, err
		}
		ft.Title.Akas = append(ft.Title.Akas, aka)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, role := range []string{"director", "cast"} {
		rows, err = db.Query(`SELECT nmid, name, character_name, pos FROM title_person WHERE id = ? AND role = ? ORDER BY pos`, id, role)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var p Person
			if err := rows.Scan(&p.NMID, &p.Name, &p.Character, &p.Pos); err != nil {
				rows.Close()
				return nil, err
			}
			if role == "director" {
				ft.Title.Directors = append(ft.Title.Directors, p)
			} else {
				ft.Title.Cast = append(ft.Title.Cast, p)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Seasons are derived from the episodes' season numbers; ft.Seasons input
	// beyond the episodes is ignored on write.
	var epImageIDs []string
	rows, err = db.Query(
		`SELECT t.id, te.season, te.episode, t.name, t.description, t.image_id, t.release_date, t.runtime, t.rating, t.votes
		 FROM title_episode te JOIN title t ON t.id = te.id
		 WHERE te.title_id = ?
		 ORDER BY te.season, te.episode IS NULL, te.episode, t.release_date = '', t.release_date, t.id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var e Episode
		var season int
		var epImageID string
		var number sql.NullInt64
		if err := rows.Scan(&e.ID, &season, &number, &e.Name, &e.Description, &epImageID, &e.ReleaseDate, &e.Runtime, &e.Rating, &e.Votes); err != nil {
			rows.Close()
			return nil, err
		}
		e.Episode = intPtr(number)
		if n := len(ft.Seasons); n == 0 || ft.Seasons[n-1].Number != season {
			ft.Seasons = append(ft.Seasons, Season{Number: season, Episodes: []Episode{}})
		}
		last := &ft.Seasons[len(ft.Seasons)-1]
		last.Episodes = append(last.Episodes, e)
		epImageIDs = append(epImageIDs, epImageID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	images, err := loadPrimaryImages(db, id)
	if err != nil {
		return nil, err
	}
	ft.Title.Image = resolveImage(images, imageID)
	i := 0
	for si := range ft.Seasons {
		for ei := range ft.Seasons[si].Episodes {
			ft.Seasons[si].Episodes[ei].Image = resolveImage(images, epImageIDs[i])
			i++
		}
	}

	return &ft, nil
}

// newEpisodeFilter returns a func reporting whether an episode id should be
// stored for the series: it skips ids that are blank, the series' own id
// (which would overwrite the series row), or repeated across seasons.
func newEpisodeFilter(seriesID string) func(id string) bool {
	seen := map[string]bool{seriesID: true, "": true}
	return func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		return true
	}
}

func (db *DB) UpsertFullTitle(ft *FullTitle) error {
	t := ft.Title
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`INSERT INTO title (id, type, is_series, name, orig_name, year_start, year_end, release_date, release_country, description, runtime, image_id, rating, votes, certificate, partial, uat, fast_uat)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, unixepoch(), unixepoch())
		 ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, is_series = EXCLUDED.is_series, name = EXCLUDED.name, orig_name = EXCLUDED.orig_name, year_start = EXCLUDED.year_start, year_end = EXCLUDED.year_end, release_date = EXCLUDED.release_date, release_country = EXCLUDED.release_country, description = EXCLUDED.description, runtime = EXCLUDED.runtime, image_id = EXCLUDED.image_id, rating = EXCLUDED.rating, votes = EXCLUDED.votes, certificate = EXCLUDED.certificate, partial = 0, uat = unixepoch(), fast_uat = unixepoch()`,
		t.ID, t.Type, boolToInt(t.IsSeries), t.Name, t.OrigName, t.YearStart, nullInt(t.YearEnd),
		t.ReleaseDate, t.ReleaseCountry, t.Description, t.Runtime,
		t.Image.ID, t.Rating, t.Votes, t.Certificate,
	); err != nil {
		return err
	}

	for _, table := range []string{"title_genre", "title_trailer", "title_person"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE id = ?`, t.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM title_aka WHERE title_id = ?`, t.ID); err != nil {
		return err
	}
	priorEpisodes, err := queryStrings(tx, `SELECT id FROM title_episode WHERE title_id = ?`, t.ID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM title_episode WHERE title_id = ?`, t.ID); err != nil {
		return err
	}

	for _, g := range dedupe(nonBlank(t.Genres), func(g string) string { return g }) {
		if _, err := tx.Exec(`INSERT INTO title_genre (id, genre) VALUES (?, ?)`, t.ID, g); err != nil {
			return err
		}
	}
	for i, vi := range dedupe(nonBlank(t.TrailerVIs), func(vi string) string { return vi }) {
		if _, err := tx.Exec(`INSERT INTO title_trailer (id, vi_id, pos) VALUES (?, ?, ?)`, t.ID, vi, i); err != nil {
			return err
		}
	}
	akas := make([]Aka, 0, len(t.Akas))
	for _, aka := range t.Akas {
		if strings.TrimSpace(aka.Text) == "" {
			continue
		}
		akas = append(akas, aka)
	}
	for _, aka := range dedupe(akas, func(a Aka) akaKey { return akaKey{a.Text, a.Country, a.Language} }) {
		if _, err := tx.Exec(`INSERT INTO title_aka (title_id, text, country, language) VALUES (?, ?, ?, ?)`, t.ID, aka.Text, aka.Country, aka.Language); err != nil {
			return err
		}
	}
	for i, d := range t.Directors {
		if _, err := tx.Exec(`INSERT INTO title_person (id, role, pos, nmid, name, character_name) VALUES (?, 'director', ?, ?, ?, '')`, t.ID, i, d.NMID, d.Name); err != nil {
			return err
		}
	}
	for i, c := range t.Cast {
		if _, err := tx.Exec(`INSERT INTO title_person (id, role, pos, nmid, name, character_name) VALUES (?, 'cast', ?, ?, ?, ?)`, t.ID, i, c.NMID, c.Name, c.Character); err != nil {
			return err
		}
	}
	var episodes []Episode
	keepEpisode := newEpisodeFilter(t.ID)
	seasons := make([]Season, 0, len(ft.Seasons))
	for _, s := range ft.Seasons {
		kept := Season{Number: s.Number}
		for _, e := range s.Episodes {
			if !keepEpisode(e.ID) {
				continue
			}
			kept.Episodes = append(kept.Episodes, e)
			episodes = append(episodes, e)
		}
		seasons = append(seasons, kept)
	}
	for _, s := range seasons {
		for _, e := range s.Episodes {
			// Episodes are titles: stored as tvEpisode rows so episode ids
			// are directly servable, linked via title_episode. A new row is
			// partial and an existing one keeps its partial flag and uat, so
			// only a full upsert of the episode itself marks it fresh. The
			// series payload refreshes the episode's fast part.
			if _, err := tx.Exec(
				`INSERT INTO title (id, type, name, description, image_id, release_date, runtime, rating, votes, partial, fast_uat)
			 VALUES (?, 'tvEpisode', ?, ?, ?, ?, ?, ?, ?, 1, unixepoch())
			 ON CONFLICT (id) DO UPDATE SET type = 'tvEpisode', name = EXCLUDED.name, description = EXCLUDED.description, image_id = EXCLUDED.image_id, release_date = EXCLUDED.release_date, runtime = EXCLUDED.runtime, rating = EXCLUDED.rating, votes = EXCLUDED.votes, fast_uat = unixepoch()`,
				e.ID, e.Name, e.Description, e.Image.ID, e.ReleaseDate, e.Runtime, e.Rating, e.Votes,
			); err != nil {
				return err
			}
			if _, err := tx.Exec(
				`INSERT INTO title_episode (id, title_id, season, episode) VALUES (?, ?, ?, ?)
			 ON CONFLICT (id) DO UPDATE SET title_id = EXCLUDED.title_id, season = EXCLUDED.season, episode = EXCLUDED.episode`,
				e.ID, t.ID, s.Number, nullInt(e.Episode),
			); err != nil {
				return err
			}
		}
	}

	// Primary images are stored so their ids can be resolved on read, and are
	// members of the title's gallery too (ep_id marks an episode's image).
	// They are kept when the title is refreshed.
	if err := addTitleImage(tx, t.ID, "", t.Image); err != nil {
		return err
	}
	for _, e := range episodes {
		if err := addTitleImage(tx, t.ID, e.ID, e.Image); err != nil {
			return err
		}
	}

	if err := deleteUnlinkedPartialEpisodes(tx, priorEpisodes); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM missing WHERE id = ?`, t.ID); err != nil {
		return err
	}

	return tx.Commit()
}

// addTitleImage stores the image and makes it a member of the title's
// gallery. A new member goes after the title's existing ones (pos is insertion
// order). An image already in the gallery keeps its position and episode tag.
// Image rows are shared: the same image can be in many titles' galleries.
// Empty fields do not overwrite stored ones: a primary image has no kind or
// caption, and must not erase what the gallery sync stored for it.
func addTitleImage(tx *sql.Tx, titleID, epID string, img Image) error {
	if img.ID == "" || strings.TrimSpace(img.URL) == "" {
		return nil
	}
	if _, err := tx.Exec(
		`INSERT INTO image (id, kind, url, width, height, caption) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET kind = COALESCE(NULLIF(EXCLUDED.kind, ''), image.kind), url = EXCLUDED.url, width = COALESCE(NULLIF(EXCLUDED.width, 0), image.width), height = COALESCE(NULLIF(EXCLUDED.height, 0), image.height), caption = COALESCE(NULLIF(EXCLUDED.caption, ''), image.caption)`,
		img.ID, img.Kind, img.URL, img.Width, img.Height, img.Caption,
	); err != nil {
		return err
	}
	_, err := tx.Exec(
		`INSERT INTO title_image (title_id, image_id, ep_id, pos)
		 VALUES (?, ?, ?, (SELECT COALESCE(MAX(pos) + 1, 0) FROM title_image WHERE title_id = ?))
		 ON CONFLICT (title_id, image_id) DO NOTHING`,
		titleID, img.ID, epID, titleID,
	)
	return err
}

// loadPrimaryImages returns the stored image rows for the title's own image
// and its episodes' images, by image id. Looking up by id keeps the result
// independent of which title or episode row currently owns an image row.
func loadPrimaryImages(db queryer, id string) (map[string]Image, error) {
	rows, err := db.Query(
		`SELECT id, kind, url, width, height, caption FROM image WHERE id IN (
		   SELECT image_id FROM title WHERE id = ?
		   UNION
		   SELECT t.image_id FROM title_episode te JOIN title t ON t.id = te.id WHERE te.title_id = ?
		 )`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images := map[string]Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.Kind, &img.URL, &img.Width, &img.Height, &img.Caption); err != nil {
			return nil, err
		}
		images[img.ID] = img
	}
	return images, rows.Err()
}

// resolveImage maps an image id to its stored image. Empty ids serve an empty
// object; ids without a stored image serve bare ({id}).
func resolveImage(images map[string]Image, id string) Image {
	if id == "" {
		return Image{}
	}
	img, ok := images[id]
	if !ok {
		return Image{ID: id}
	}
	return img
}

// PurgeBlankValues deletes blank child rows (empty/whitespace genre,
// trailer vi_id, aka text) left by older writes. Runs at startup; new
// upserts already filter these.
func (db *DB) PurgeBlankValues() error {
	for _, q := range []string{
		`DELETE FROM title_genre WHERE trim(genre) = ''`,
		`DELETE FROM title_trailer WHERE trim(vi_id) = ''`,
		`DELETE FROM title_aka WHERE trim(text) = ''`,
	} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// missingKept is how long a missing row is kept before it is pruned.
const missingKept = 7 * 24 * time.Hour

// MarkMissing records a plugin 404 for id, and prunes rows older than
// missingKept so the table stays bounded.
func (db *DB) MarkMissing(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := markMissing(tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func markMissing(tx *sql.Tx, id string) error {
	if _, err := tx.Exec(`INSERT INTO missing (id, uat) VALUES (?, unixepoch()) ON CONFLICT (id) DO UPDATE SET uat = unixepoch()`, id); err != nil {
		return err
	}
	_, err := tx.Exec(`DELETE FROM missing WHERE uat < unixepoch() - ?`, int64(missingKept.Seconds()))
	return err
}

// IsMissingFresh reports whether id has a fresh 404 tombstone (younger than maxAge).
func (db *DB) IsMissingFresh(id string, maxAge time.Duration) (bool, error) {
	var uat int64
	err := db.QueryRow(`SELECT uat FROM missing WHERE id = ?`, id).Scan(&uat)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return time.Since(time.Unix(uat, 0)) < maxAge, nil
}
