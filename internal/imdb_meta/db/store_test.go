package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// episodesOf flattens the episodes of all seasons.
func episodesOf(ft *FullTitle) []Episode {
	var out []Episode
	for _, s := range ft.Seasons {
		out = append(out, s.Episodes...)
	}
	return out
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "imdb.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpsertGetRoundTrip(t *testing.T) {
	db := openTestDB(t)

	rating := 77
	yearEnd := 2013
	ft := &FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747",
			Name: "Breaking Bad",

			ReleaseDate: "2008-01-20",
			Description: "A chemistry teacher...", Runtime: 2880,
			Image: Image{ID: "rm1", URL: "http://image", Width: 300, Height: 450},

			Rating: rating, Votes: 1000}, Type: "tvSeries", IsSeries: true,
			OrigName:  "Breaking Bad",
			YearStart: 2008, YearEnd: &yearEnd,
			ReleaseCountry: "US",

			TrailerVIs:  []string{"vi123"},
			Certificate: "TV-MA",
			Genres:      []string{"Crime", "Drama"},
			Directors:   []Person{{NMID: "nm0123456", Name: "Vince Gilligan", Pos: 0}},
			Cast:        []Person{{NMID: "nm0654877", Name: "Bryan Cranston", Character: "Walter White", Pos: 0}},
			Akas:        []Aka{{Text: "Breaking Bad", Country: "US", Language: "en"}},
		},
		Seasons: []Season{{
			Number: 1,
			Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot", Description: "Plot...", Image: Image{ID: "rm-thumb", URL: "http://thumb"}, ReleaseDate: "2008-01-20", Runtime: 2880, Rating: rating, Votes: 86500}, Episode: intPtrOf(1)},
				{TitleSummary: TitleSummary{ID: "tt1054725", Name: "Cat's in the Bag...", Description: "Plot2...", ReleaseDate: "2008-01-27"}, Episode: intPtrOf(2)},
			},
		}},
	}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title.Name != "Breaking Bad" || got.Title.Type != "tvSeries" || !got.Title.IsSeries {
		t.Fatalf("title mismatch: %+v", got.Title)
	}
	if got.Title.YearEnd == nil || *got.Title.YearEnd != 2013 {
		t.Fatalf("year_end mismatch: %+v", got.Title.YearEnd)
	}
	if got.Title.Image.ID != "rm1" || got.Title.Image.URL != "http://image" || got.Title.Image.Width != 300 {
		t.Fatalf("image mismatch: %+v", got.Title.Image)
	}
	if len(got.Title.Genres) != 2 || len(got.Title.TrailerVIs) != 1 || len(got.Title.Akas) != 1 {
		t.Fatalf("child counts mismatch: %+v", got.Title)
	}
	if len(got.Seasons) != 1 || len(got.Seasons[0].Episodes) != 2 {
		t.Fatalf("seasons mismatch: %+v", got.Seasons)
	}
	if len(episodesOf(got)) != 2 || episodesOf(got)[0].ID != "tt0959621" || episodesOf(got)[1].ID != "tt1054725" {
		t.Fatalf("episodes mismatch: %+v", episodesOf(got))
	}
	if episodesOf(got)[0].Votes != 86500 || episodesOf(got)[0].Rating == 0 {
		t.Fatalf("episode rating mismatch: %+v", episodesOf(got)[0])
	}
	if want := (Image{ID: "rm-thumb", URL: "http://thumb"}); episodesOf(got)[0].Image != want {
		t.Fatalf("episode image mismatch: %+v", episodesOf(got)[0].Image)
	}
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	db := openTestDB(t)
	_, err := db.GetFullTitle("tt0000000")
	if !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDirForDatabaseURI(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for uri, want := range map[string]string{
		"sqlite://./data/stremthru.db":  filepath.Join(cwd, "data"),
		"sqlite:///tmp/st.db":           "/tmp",
		"postgresql://u:p@localhost/db": "/fallback",
		"bogus://uri":                   "/fallback",
	} {
		if got := dirForDatabaseURI(uri, "/fallback"); got != want {
			t.Fatalf("dirForDatabaseURI(%q) = %q, want %q", uri, got, want)
		}
	}
}

func TestOpenCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "imdb.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
}

func TestUpsertShrinksEpisodes(t *testing.T) {
	db := openTestDB(t)

	rating := 77
	ft := &FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad", Rating: rating}, Type: "tvSeries", IsSeries: true, YearStart: 2008},
		Seasons: []Season{{
			Number: 1,
			Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot"}, Episode: intPtrOf(1)},
				{TitleSummary: TitleSummary{ID: "tt1054725", Name: "Cat's in the Bag..."}, Episode: intPtrOf(2)},
			},
		}},
	}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}

	ft.Seasons[0].Episodes = ft.Seasons[0].Episodes[:1]
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if len(episodesOf(got)) != 1 || episodesOf(got)[0].ID != "tt0959621" {
		t.Fatalf("ghost episodes after shrink: %+v", episodesOf(got))
	}
}

func TestUpsertSkipsInvalidEpisodes(t *testing.T) {
	db := openTestDB(t)

	ft := &FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}, Type: "tvSeries", IsSeries: true, YearStart: 2008},
		Seasons: []Season{
			{Number: 1, Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot"}, Episode: intPtrOf(1)},
				{TitleSummary: TitleSummary{ID: "", Name: "Blank"}, Episode: intPtrOf(2)},
				{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Self"}, Episode: intPtrOf(3)},
			}},
			{Number: 2, Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot again"}, Episode: intPtrOf(1)},
			}},
		},
	}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title.Type != "tvSeries" || got.Title.Name != "Breaking Bad" {
		t.Fatalf("series row overwritten: %+v", got.Title)
	}
	eps := episodesOf(got)
	if len(eps) != 1 || eps[0].ID != "tt0959621" || eps[0].Name != "Pilot" {
		t.Fatalf("episodes = %+v", eps)
	}
	if _, err := db.GetFullTitle(""); err == nil {
		t.Fatal("blank-id title row was stored")
	}
}

func TestUpsertDedupesChildren(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0110912", Name: "Pulp Fiction"}, Type: "movie", YearStart: 1994,
			Genres:     []string{"Crime", "Drama", "Crime"},
			TrailerVIs: []string{"vi1", "vi1"},
			Akas: []Aka{
				{Text: "Pulp Fiction", Country: "US"},
				{Text: "Pulp Fiction", Country: "US"},
			},
		},
		Seasons: []Season{{
			Number: 1,
			Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0110913", Name: "Part 1"}, Episode: intPtrOf(1)},
				{TitleSummary: TitleSummary{ID: "tt0110914", Name: "Part 2"}, Episode: intPtrOf(2)},
			},
		}},
	}); err != nil {
		t.Fatalf("upsert with duplicates: %v", err)
	}
	got, err := db.GetFullTitle("tt0110912")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Title.Genres) != 2 || len(got.Title.TrailerVIs) != 1 || len(got.Title.Akas) != 1 {
		t.Fatalf("children not deduped: %+v", got.Title)
	}
	if len(got.Seasons) != 1 || len(got.Seasons[0].Episodes) != 2 {
		t.Fatalf("seasons not derived: %+v", got.Seasons)
	}
}

func TestPurgeBlankValues(t *testing.T) {
	db := openTestDB(t)
	for _, q := range []string{
		`INSERT INTO title (id, name) VALUES ('tt0110912', 'Pulp Fiction')`,
		`INSERT INTO title_genre (id, genre) VALUES ('tt0110912', ''), ('tt0110912', '  '), ('tt0110912', 'Crime')`,
		`INSERT INTO title_trailer (id, vi_id, pos) VALUES ('tt0110912', '', 0), ('tt0110912', 'vi1', 1)`,
		`INSERT INTO title_aka (title_id, text, country) VALUES ('tt0110912', '', 'US'), ('tt0110912', 'Pulp Fiction', 'US')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.PurgeBlankValues(); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetFullTitle("tt0110912")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Title.Genres) != 1 || got.Title.Genres[0] != "Crime" {
		t.Fatalf("genres: %+v", got.Title.Genres)
	}
	if len(got.Title.TrailerVIs) != 1 || got.Title.TrailerVIs[0] != "vi1" {
		t.Fatalf("trailers: %+v", got.Title.TrailerVIs)
	}
	if len(got.Title.Akas) != 1 || got.Title.Akas[0].Text != "Pulp Fiction" {
		t.Fatalf("akas: %+v", got.Title.Akas)
	}
}

func TestUpsertDropsBlankValues(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0110912", Name: "Pulp Fiction"}, Type: "movie", YearStart: 1994,
			Genres:     []string{"", "  ", "Crime"},
			TrailerVIs: []string{"", "vi1"},
			Akas: []Aka{
				{Text: "", Country: "US"},
				{Text: "Pulp Fiction", Country: "US"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetFullTitle("tt0110912")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Title.Genres) != 1 || got.Title.Genres[0] != "Crime" {
		t.Fatalf("blank genres not dropped: %+v", got.Title.Genres)
	}
	if len(got.Title.TrailerVIs) != 1 || got.Title.TrailerVIs[0] != "vi1" {
		t.Fatalf("blank trailers not dropped: %+v", got.Title.TrailerVIs)
	}
	if len(got.Title.Akas) != 1 || got.Title.Akas[0].Text != "Pulp Fiction" {
		t.Fatalf("blank akas not dropped: %+v", got.Title.Akas)
	}
}

func TestGetEmptyChildrenAreArrays(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0110912", Name: "Pulp Fiction"}, Type: "movie", YearStart: 1994},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetFullTitle("tt0110912")
	if err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]any{
		"genres": got.Title.Genres, "trailers": got.Title.TrailerVIs,
		"akas": got.Title.Akas, "directors": got.Title.Directors, "cast": got.Title.Cast,
		"seasons": got.Seasons,
	} {
		if reflect.ValueOf(v).IsNil() {
			t.Fatalf("%s serialized as null", name)
		}
	}
}

func TestGetEpisodeDirectly(t *testing.T) {
	db := openTestDB(t)
	rating := 77
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}, Type: "tvSeries", IsSeries: true, YearStart: 2008},
		Seasons: []Season{{
			Number: 1,
			Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot", Description: "Plot...", Image: Image{ID: "rm-thumb", URL: "http://thumb"}, ReleaseDate: "2008-01-20", Runtime: 2880, Rating: rating, Votes: 86500}, Episode: intPtrOf(1)},
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatalf("episode id not servable: %v", err)
	}
	if got.Title.ID != "tt0959621" || got.Title.Name != "Pilot" {
		t.Fatalf("episode title mismatch: %+v", got.Title)
	}
	if got.Title.Description != "Plot..." || got.Title.Image != (Image{ID: "rm-thumb", URL: "http://thumb"}) {
		t.Fatalf("episode image not stored from the series payload: %+v", got.Title.Image)
	}
}

func partialFixture() *FullTitle {
	return &FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}, Type: "tvSeries", IsSeries: true, YearStart: 2008},
		Seasons: []Season{{
			Number: 1,
			Episodes: []Episode{
				{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot", Rating: 91, Votes: 100}, Episode: intPtrOf(1)},
			},
		}},
	}
}

func episodeRow(t *testing.T, db *DB, id string) (partial int, uat int64) {
	t.Helper()
	if err := db.QueryRow(`SELECT partial, uat FROM title WHERE id = ?`, id).Scan(&partial, &uat); err != nil {
		t.Fatal(err)
	}
	return partial, uat
}

func TestEpisodeFromSeriesIsPartial(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	series, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if series.Title.Partial {
		t.Fatal("series must not be partial")
	}
	ep, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if !ep.Title.Partial {
		t.Fatalf("episode stored from a series must be partial: %+v", ep.Title)
	}
}

func TestSeriesUpsertKeepsEpisodeUat(t *testing.T) {
	db := openTestDB(t)
	ft := partialFixture()
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE title SET uat = 100 WHERE id = 'tt0959621'`); err != nil {
		t.Fatal(err)
	}

	ft.Seasons[0].Episodes[0].Name = "Pilot (renamed)"
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}

	partial, uat := episodeRow(t, db, "tt0959621")
	if partial != 1 || uat != 100 {
		t.Fatalf("series upsert changed partial/uat: partial=%d uat=%d", partial, uat)
	}
	ep, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Title.Name != "Pilot (renamed)" {
		t.Fatalf("episode columns must still update: %+v", ep.Title)
	}
}

func TestFullUpsertClearsPartialAndSurvivesSeriesUpsert(t *testing.T) {
	db := openTestDB(t)
	ft := partialFixture()
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot"}, Type: "tvEpisode", YearStart: 2008, Genres: []string{"Crime"}},
	}); err != nil {
		t.Fatal(err)
	}
	partial, uat := episodeRow(t, db, "tt0959621")
	if partial != 0 || uat < 1_000_000_000 {
		t.Fatalf("full upsert must clear partial and set uat: partial=%d uat=%d", partial, uat)
	}

	if _, err := db.Exec(`UPDATE title SET uat = 100 WHERE id = 'tt0959621'`); err != nil {
		t.Fatal(err)
	}
	ft.Seasons[0].Episodes[0].Name = "Pilot (renamed)"
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	partial, uat = episodeRow(t, db, "tt0959621")
	if partial != 0 || uat != 100 {
		t.Fatalf("series upsert must not downgrade a full episode: partial=%d uat=%d", partial, uat)
	}
}

func imageFixture() *FullTitle {
	return &FullTitle{
		Title: Title{
			TitleSummary: TitleSummary{
				ID: "tt0903747", Name: "Breaking Bad",
				Image: Image{ID: "rm-poster", Kind: "poster", URL: "http://poster", Width: 2000, Height: 3000, Caption: "Breaking Bad poster"},
			},
			Type: "tvSeries", IsSeries: true, YearStart: 2008,
		}, Seasons: []Season{{Number: 1, Episodes: []Episode{{
			TitleSummary: TitleSummary{
				ID: "tt0959621", Name: "Pilot",
				Image: Image{ID: "rm-still", URL: "http://still", Width: 3224, Height: 1814},
			},
			Episode: intPtrOf(1),
		}}}},
	}
}

func TestSeriesAndEpisodeImagesAreStoredAndResolved(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(imageFixture()); err != nil {
		t.Fatal(err)
	}

	series, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Image{ID: "rm-poster", Kind: "poster", URL: "http://poster", Width: 2000, Height: 3000, Caption: "Breaking Bad poster"}); series.Title.Image != want {
		t.Fatalf("series image = %+v, want %+v", series.Title.Image, want)
	}
	if want := (Image{ID: "rm-still", URL: "http://still", Width: 3224, Height: 1814}); len(episodesOf(series)) != 1 || episodesOf(series)[0].Image != want {
		t.Fatalf("episode image must be restored on read: %+v", episodesOf(series))
	}

	ep, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if want := (Image{ID: "rm-still", URL: "http://still", Width: 3224, Height: 1814}); ep.Title.Image != want {
		t.Fatalf("episode served directly lost its image: %+v", ep.Title.Image)
	}
}

func TestStoredImageRowsMarkEpisodeImages(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(imageFixture()); err != nil {
		t.Fatal(err)
	}
	var kind, caption, epID, titleID string
	if err := db.QueryRow(`SELECT ti.title_id, ti.ep_id, i.kind, i.caption FROM title_image ti JOIN image i ON i.id = ti.image_id WHERE i.id = 'rm-still'`).Scan(&titleID, &epID, &kind, &caption); err != nil {
		t.Fatal(err)
	}
	if titleID != "tt0903747" || epID != "tt0959621" {
		t.Fatalf("episode image row: title_id=%q ep_id=%q", titleID, epID)
	}
	if err := db.QueryRow(`SELECT ti.title_id, ti.ep_id, i.kind, i.caption FROM title_image ti JOIN image i ON i.id = ti.image_id WHERE i.id = 'rm-poster'`).Scan(&titleID, &epID, &kind, &caption); err != nil {
		t.Fatal(err)
	}
	if titleID != "tt0903747" || epID != "" || kind != "poster" || caption != "Breaking Bad poster" {
		t.Fatalf("title image row: title_id=%q ep_id=%q kind=%q caption=%q", titleID, epID, kind, caption)
	}
}

func TestImageWithoutURLIsNotStored(t *testing.T) {
	db := openTestDB(t)
	ft := imageFixture()
	ft.Title.Image = Image{ID: "rm-bare"}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image WHERE id = 'rm-bare'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("an image without a url must not be stored")
	}
	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title.Image != (Image{ID: "rm-bare"}) {
		t.Fatalf("unresolvable image should serve bare id: %+v", got.Title.Image)
	}
}

func TestImageResolvesWhenRowChangesOwner(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(imageFixture()); err != nil {
		t.Fatal(err)
	}
	// The episode's own fetch stores its image again as its title-level
	// image, moving the row away from the series.
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{
			ID: "tt0959621", Name: "Pilot",
			Image: Image{ID: "rm-still", URL: "http://still", Width: 3224, Height: 1814},
		}, Type: "tvEpisode"},
	}); err != nil {
		t.Fatal(err)
	}

	series, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if episodesOf(series)[0].Image.URL != "http://still" || episodesOf(series)[0].Image.Width != 3224 {
		t.Fatalf("series lost the episode image after the episode's own fetch: %+v", episodesOf(series)[0].Image)
	}
}

func TestEpisodeServesChildTablesAfterFullFetch(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot"}, Type: "tvEpisode", YearStart: 2008, Certificate: "TV-MA",
			Genres:     []string{"Crime", "Drama"},
			TrailerVIs: []string{"vi1"},
			Akas:       []Aka{{Text: "Piloto", Country: "ES"}},
			Directors:  []Person{{NMID: "nm0004982", Name: "Vince Gilligan"}},
			Cast:       []Person{{NMID: "nm0186505", Name: "Bryan Cranston", Character: "Walter White"}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Title.Genres, []string{"Crime", "Drama"}) || !reflect.DeepEqual(got.Title.TrailerVIs, []string{"vi1"}) {
		t.Fatalf("genres/trailers not served for an episode: %+v", got.Title)
	}
	if len(got.Title.Akas) != 1 || len(got.Title.Directors) != 1 || len(got.Title.Cast) != 1 || got.Title.Cast[0].Character != "Walter White" {
		t.Fatalf("akas/people not served for an episode: %+v", got.Title)
	}
	if len(episodesOf(got)) != 0 || len(got.Seasons) != 0 {
		t.Fatalf("an episode has no episodes of its own: %+v %+v", got.Seasons, episodesOf(got))
	}
}

func TestEpisodesAreNestedInSeasonsInOrder(t *testing.T) {
	db := openTestDB(t)
	ep := func(id string, n int) Episode {
		return Episode{TitleSummary: TitleSummary{ID: id, Name: id}, Episode: intPtrOf(n)}
	}
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}, Type: "tvSeries", IsSeries: true},
		Seasons: []Season{
			{Number: 2, Episodes: []Episode{ep("tt21", 1), ep("tt22", 2)}},
			{Number: 0, Episodes: []Episode{ep("tt00", 1)}},
			{Number: 1, Episodes: []Episode{ep("tt12", 2), ep("tt11", 1)}},
		},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, s := range got.Seasons {
		for _, e := range s.Episodes {
			seen = append(seen, strconv.Itoa(s.Number)+":"+e.ID)
		}
	}
	want := []string{"0:tt00", "1:tt11", "1:tt12", "2:tt21", "2:tt22"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("seasons/episodes = %v, want %v", seen, want)
	}

	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var shape struct {
		Episodes json.RawMessage `json:"episodes"`
		Seasons  []struct {
			Episodes []map[string]json.RawMessage `json:"episodes"`
		} `json:"seasons"`
	}
	if err := json.Unmarshal(b, &shape); err != nil {
		t.Fatal(err)
	}
	if shape.Episodes != nil {
		t.Fatal("episodes must not be a top-level key")
	}
	if len(shape.Seasons) != 3 || len(shape.Seasons[1].Episodes) != 2 {
		t.Fatalf("seasons json: %s", b)
	}
	if _, ok := shape.Seasons[1].Episodes[0]["season"]; ok {
		t.Fatal("episodes must not repeat their season number")
	}
}

// Episode 0 is a real number (specials that air before episode 1), so it sorts
// first, and unnumbered episodes sort last; ties go by release date, undated
// last.
func TestEpisodesOrderedByNumberThenReleaseDate(t *testing.T) {
	db := openTestDB(t)
	ep := func(id string, n int, date string) Episode {
		return Episode{TitleSummary: TitleSummary{ID: id, Name: id, ReleaseDate: date}, Episode: intPtrOf(n)}
	}
	unnumbered := func(id string, date string) Episode {
		return Episode{TitleSummary: TitleSummary{ID: id, Name: id, ReleaseDate: date}}
	}
	if err := db.UpsertFullTitle(&FullTitle{
		Title: Title{TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}, Type: "tvSeries", IsSeries: true},
		Seasons: []Season{
			{Number: 1, Episodes: []Episode{
				unnumbered("tt-x-undated", ""),
				ep("tt-n3", 3, "2008-02-10"),
				unnumbered("tt-x-dated", "2001-01-01"),
				ep("tt-u-late", 0, "2009-01-01"),
				ep("tt-n1", 1, "2008-01-20"),
				ep("tt-u-undated", 0, ""),
				ep("tt-u-early", 0, "2007-05-05"),
				ep("tt-n2", 2, ""),
			}},
			{Number: 2, Episodes: []Episode{
				ep("tt-b", 0, "2010-01-01"),
				ep("tt-a", 0, "2010-01-01"),
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	var ids [][]string
	for _, s := range got.Seasons {
		var row []string
		for _, e := range s.Episodes {
			row = append(row, e.ID)
		}
		ids = append(ids, row)
	}
	want := [][]string{
		{"tt-u-early", "tt-u-late", "tt-u-undated", "tt-n1", "tt-n2", "tt-n3", "tt-x-dated", "tt-x-undated"},
		{"tt-a", "tt-b"},
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("episode order = %v, want %v", ids, want)
	}
}

func intPtrOf(n int) *int {
	return &n
}

func TestMarkMissingPrunesOldRows(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.Exec(`INSERT INTO missing (id, uat) VALUES ('tt1', unixepoch() - 8*86400)`); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkMissing("tt2"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	rows, err := db.Query(`SELECT id FROM missing ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	if len(ids) != 1 || ids[0] != "tt2" {
		t.Fatalf("missing ids = %v", ids)
	}
}
