package db

import (
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/plugin/imdb"
)

func TestRefreshStateAfterFullUpsert(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}

	series, err := db.GetRefreshState("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if !series.IsSeries || !series.Running || series.Partial || series.UpdatedAt.IsZero() || series.FastUpdatedAt.IsZero() || series.SeriesID != "" {
		t.Fatalf("series state = %+v", series)
	}

	ep, err := db.GetRefreshState("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if !ep.Partial || ep.Type != "tvEpisode" || ep.SeriesID != "tt0903747" || !ep.SeriesRunning || ep.FastUpdatedAt.IsZero() || !ep.SeriesFastUpdatedAt.Equal(series.FastUpdatedAt) {
		t.Fatalf("episode state = %+v", ep)
	}
}

func TestRefreshStateOrphanEpisode(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM title_episode`); err != nil {
		t.Fatal(err)
	}
	ep, err := db.GetRefreshState("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if ep.SeriesID != "" || ep.SeriesRunning {
		t.Fatalf("orphan episode state = %+v", ep)
	}
}

func TestRefreshStateNotFound(t *testing.T) {
	db := openTestDB(t)
	if _, err := db.GetRefreshState("tt0000000"); !IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func titleCount(t *testing.T, db *DB, id string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM title WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func episodeIDs(t *testing.T, db *DB, seriesID string) []string {
	t.Helper()
	ft, err := db.GetFullTitle(seriesID)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range episodesOf(ft) {
		ids = append(ids, e.ID)
	}
	return ids
}

func TestApplyFastRefreshAddsAndDropsEpisodes(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Seasons[0].Episodes = append(series.Seasons[0].Episodes,
		Episode{TitleSummary: TitleSummary{ID: "tt1054724", Name: "Cat's in the Bag..."}, Episode: intPtrOf(2)})
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	// B has had its own full fetch.
	if err := db.UpsertFullTitle(&FullTitle{Title: Title{
		TitleSummary: TitleSummary{ID: "tt1054724", Name: "Cat's in the Bag..."}, Type: "tvEpisode", Genres: []string{"Crime"},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := db.ApplyFastRefresh(&FastRefresh{
		ID:     "tt0903747",
		Series: true,
		Episodes: []EpisodeRef{
			{ID: "tt1054724", Season: 1, Episode: intPtrOf(2)},
			{ID: "tt1054725", Season: 1, Episode: intPtrOf(3)},
		},
		Details: []TitleSummary{{ID: "tt1054725", Name: "...And the Bag's in the River"}},
	}); err != nil {
		t.Fatal(err)
	}

	if got := episodeIDs(t, db, "tt0903747"); len(got) != 2 || got[0] != "tt1054724" || got[1] != "tt1054725" {
		t.Fatalf("episodes = %v", got)
	}
	if n := titleCount(t, db, "tt0959621"); n != 0 {
		t.Fatal("dropped partial episode row was kept")
	}
	b, err := db.GetFullTitle("tt1054724")
	if err != nil || len(b.Title.Genres) != 1 {
		t.Fatalf("full episode = %+v, %v", b, err)
	}
	c, err := db.GetFullTitle("tt1054725")
	if err != nil || !c.Title.Partial || c.Title.Name != "...And the Bag's in the River" {
		t.Fatalf("new episode = %+v, %v", c, err)
	}
}

func TestApplyFastRefreshKeepsUndetailedFields(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Seasons[0].Episodes[0].Runtime = 2880
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyFastRefresh(&FastRefresh{
		ID: "tt0903747", Series: true,
		Episodes: []EpisodeRef{{ID: "tt0959621", Season: 1, Episode: intPtrOf(1), Plot: "new plot"}},
	}); err != nil {
		t.Fatal(err)
	}
	ep, err := db.GetFullTitle("tt0959621")
	if err != nil {
		t.Fatal(err)
	}
	if ep.Title.Name != "Pilot" || ep.Title.Runtime != 2880 || ep.Title.Description != "new plot" {
		t.Fatalf("episode = %+v", ep.Title)
	}
}

func TestApplyFastRefreshRatingsAndTimes(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE title SET fast_uat = 1, uat = 1000`); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyFastRefresh(&FastRefresh{
		ID: "tt0903747", Series: true,
		Episodes: []EpisodeRef{{ID: "tt0959621", Season: 1, Episode: intPtrOf(1)}},
		Ratings:  []imdbplugin.Rating{{ID: "tt0903747", Rating: 95, Votes: 2300000}, {ID: "tt0959621", Rating: 90, Votes: 41000}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		id            string
		rating, votes int
	}{{"tt0903747", 95, 2300000}, {"tt0959621", 90, 41000}} {
		var rating, votes int
		var uat, fastUAT int64
		if err := db.QueryRow(`SELECT rating, votes, uat, fast_uat FROM title WHERE id = ?`, c.id).Scan(&rating, &votes, &uat, &fastUAT); err != nil {
			t.Fatal(err)
		}
		if rating != c.rating || votes != c.votes || uat != 1000 || fastUAT <= 1 {
			t.Fatalf("%s: rating %d votes %d uat %d fast_uat %d", c.id, rating, votes, uat, fastUAT)
		}
	}
}

func TestApplyFastRefreshRatingsOnlyLeavesEpisodes(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyFastRefresh(&FastRefresh{
		ID:      "tt0903747",
		Ratings: []imdbplugin.Rating{{ID: "tt0903747", Rating: 95, Votes: 10}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := episodeIDs(t, db, "tt0903747"); len(got) != 1 {
		t.Fatalf("episodes = %v", got)
	}
	ft, _ := db.GetFullTitle("tt0903747")
	if ft.Title.Rating != 95 || ft.Title.Votes != 10 {
		t.Fatalf("title = %+v", ft.Title)
	}
}

func TestListEpisodeDates(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Seasons[0].Episodes[0].ReleaseDate = "2008-01-20"
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	got, err := db.ListEpisodeDates("tt0903747")
	if err != nil || len(got) != 1 || got["tt0959621"] != "2008-01-20" {
		t.Fatalf("dates = %v, %v", got, err)
	}
}

func TestDeleteTitleRemovesEverything(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Title.Image = Image{ID: "rm-poster", URL: "http://poster"}
	series.Title.Genres = []string{"Crime"}
	series.Seasons[0].Episodes[0].Image = Image{ID: "rm-still", URL: "http://still"}
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt0903747", galleryImages(1, 3)); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteGallery("tt0903747"); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt2", galleryImages(2, 3)); err != nil {
		t.Fatal(err)
	}

	if err := db.DeleteTitle("tt0903747"); err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetFullTitle("tt0903747"); !IsNotFound(err) {
		t.Fatalf("title: %v", err)
	}
	if _, err := db.GetGallery("tt0903747"); !IsNotFound(err) {
		t.Fatalf("gallery: %v", err)
	}
	if n := titleCount(t, db, "tt0959621"); n != 0 {
		t.Fatal("partial episode was kept")
	}
	for id, want := range map[string]int{"rm-poster": 0, "rm-still": 0, "rm1": 0, "rm2": 1} {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM image WHERE id = ?`, id).Scan(&n)
		if n != want {
			t.Fatalf("image %s rows = %d, want %d", id, n, want)
		}
	}
	for _, table := range []string{"title_genre", "title_episode", "title_image"} {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE ` + map[string]string{"title_genre": "id", "title_episode": "title_id", "title_image": "title_id"}[table] + ` = 'tt0903747'`).Scan(&n)
		if n != 0 {
			t.Fatalf("%s rows = %d", table, n)
		}
	}
}

func TestFullUpsertDeletesDroppedPartialEpisodes(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Seasons[0].Episodes = append(series.Seasons[0].Episodes,
		Episode{TitleSummary: TitleSummary{ID: "tt1054724", Name: "Cat's in the Bag..."}, Episode: intPtrOf(2)},
		Episode{TitleSummary: TitleSummary{ID: "tt1054725", Name: "...And the Bag's in the River"}, Episode: intPtrOf(3)})
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	storeFullEpisode(t, db, "tt1054725")

	series.Seasons[0].Episodes = series.Seasons[0].Episodes[:1]
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	if n := titleCount(t, db, "tt1054724"); n != 0 {
		t.Fatal("dropped partial episode row was kept")
	}
	if n := titleCount(t, db, "tt1054725"); n != 1 {
		t.Fatal("dropped full episode row was deleted")
	}
}

func storeFullEpisode(t *testing.T, db *DB, id string) {
	t.Helper()
	if err := db.UpsertFullTitle(&FullTitle{Title: Title{
		TitleSummary: TitleSummary{ID: id, Name: "Episode " + id}, Type: "tvEpisode", Genres: []string{"Crime"},
	}}); err != nil {
		t.Fatal(err)
	}
}

func countWhere(t *testing.T, db *DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeleteTitleRemovesEpisodeGalleryRows(t *testing.T) {
	db := openTestDB(t)
	series := partialFixture()
	series.Seasons[0].Episodes[0].Image = Image{ID: "rm-still", URL: "http://still"}
	if err := db.UpsertFullTitle(series); err != nil {
		t.Fatal(err)
	}
	// The partial episode's own gallery, from a request for its images.
	if err := db.AddGalleryImages("tt0959621", galleryImages(5, 6)); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteGallery("tt0959621"); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteTitle("tt0903747"); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, db, `SELECT COUNT(*) FROM title_image WHERE title_id = 'tt0959621'`); n != 0 {
		t.Fatalf("deleted episode's gallery rows = %d", n)
	}
	if n := countWhere(t, db, `SELECT COUNT(*) FROM title_gallery WHERE title_id = 'tt0959621'`); n != 0 {
		t.Fatal("deleted episode's gallery state kept")
	}
	if n := countWhere(t, db, `SELECT COUNT(*) FROM image WHERE id IN ('rm5', 'rm-still')`); n != 0 {
		t.Fatalf("unused image rows = %d", n)
	}
}

func TestDeleteEpisodeRemovesItsStillsFromOtherGalleries(t *testing.T) {
	db := openTestDB(t)
	storeFullEpisode(t, db, "tt0959621")
	if err := db.AddGalleryImages("tt0903747", []Image{{ID: "rm-still", URL: "http://still", EpID: "tt0959621"}, {ID: "rm1", URL: "http://img/1"}}); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteTitle("tt0959621"); err != nil {
		t.Fatal(err)
	}
	got, total, _ := db.ListGalleryImages("tt0903747", 10, 0)
	if total != 1 || got[0].ID != "rm1" {
		t.Fatalf("series gallery = %+v", got)
	}
}

func TestRemoveTitleDeletesAndRecordsMissing(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	if err := db.RemoveTitle("tt0903747"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetFullTitle("tt0903747"); !IsNotFound(err) {
		t.Fatalf("title: %v", err)
	}
	if missing, err := db.IsMissingFresh("tt0903747", time.Hour); err != nil || !missing {
		t.Fatalf("missing = %v, %v", missing, err)
	}
}

func TestCompleteGalleryRefusesMissingTitle(t *testing.T) {
	db := openTestDB(t)
	if err := db.AddGalleryImages("tt0903747", galleryImages(0, 2)); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkMissing("tt0903747"); err != nil {
		t.Fatal(err)
	}
	if err := db.CompleteGallery("tt0903747"); !IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
	if _, err := db.GetGallery("tt0903747"); !IsNotFound(err) {
		t.Fatalf("gallery completed: %v", err)
	}
	if _, total, _ := db.ListGalleryImages("tt0903747", 10, 0); total != 0 {
		t.Fatalf("gallery rows kept: %d", total)
	}
}

func TestGetTitleAndState(t *testing.T) {
	db := openTestDB(t)
	if err := db.UpsertFullTitle(partialFixture()); err != nil {
		t.Fatal(err)
	}
	ft, st, err := db.GetTitleAndState("tt0903747")
	if err != nil || ft.Title.Name != "Breaking Bad" || !st.Running || len(episodesOf(ft)) != 1 {
		t.Fatalf("ft = %+v, st = %+v, %v", ft, st, err)
	}
	if _, _, err := db.GetTitleAndState("tt0000000"); !IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
}
