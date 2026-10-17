package db

import (
	"fmt"
	"testing"
)

func galleryImages(from, to int) []Image {
	var out []Image
	for i := from; i < to; i++ {
		out = append(out, Image{
			ID: fmt.Sprintf("rm%d", i), URL: fmt.Sprintf("http://img/%d", i), Width: 10, Height: 20,
			Kind: "still_frame", Caption: fmt.Sprintf("caption %d", i),
		})
	}
	return out
}

func TestGalleryAddListComplete(t *testing.T) {
	db := openTestDB(t)

	if _, err := db.GetGallery("tt1"); !IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
	if err := db.AddGalleryImages("tt1", galleryImages(0, 3)); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt1", galleryImages(3, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetGallery("tt1"); !IsNotFound(err) {
		t.Fatalf("gallery must not be complete before CompleteGallery: %v", err)
	}
	if err := db.CompleteGallery("tt1"); err != nil {
		t.Fatal(err)
	}
	if st, err := db.GetGallery("tt1"); err != nil || st.UpdatedAt.IsZero() {
		t.Fatalf("state = %+v, %v", st, err)
	}

	got, total, err := db.ListGalleryImages("tt1", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || len(got) != 2 || got[0].ID != "rm2" || got[1].ID != "rm3" {
		t.Fatalf("total = %d, got %+v", total, got)
	}
	if got[0] != (Image{ID: "rm2", URL: "http://img/2", Width: 10, Height: 20, Kind: "still_frame", Caption: "caption 2"}) {
		t.Fatalf("image fields = %+v", got[0])
	}
	if got, _, _ := db.ListGalleryImages("tt1", 10, 99); got == nil || len(got) != 0 {
		t.Fatalf("past the end = %+v", got)
	}
}

func TestGallerySkipsBlankAndIsRepeatable(t *testing.T) {
	db := openTestDB(t)
	imgs := append(galleryImages(0, 2), Image{ID: "", URL: "http://x"}, Image{ID: "rm9", URL: "  "})
	for i := 0; i < 2; i++ {
		if err := db.AddGalleryImages("tt1", imgs); err != nil {
			t.Fatal(err)
		}
	}
	got, total, _ := db.ListGalleryImages("tt1", 10, 0)
	if total != 2 || len(got) != 2 || got[0].ID != "rm0" || got[1].ID != "rm1" {
		t.Fatalf("total = %d, images = %+v", total, got)
	}
}

func TestGalleryRefreshesExistingImageInPlace(t *testing.T) {
	db := openTestDB(t)
	if err := db.AddGalleryImages("tt1", galleryImages(0, 3)); err != nil {
		t.Fatal(err)
	}
	updated := galleryImages(1, 2)
	updated[0].URL = "http://img/new"
	if err := db.AddGalleryImages("tt1", updated); err != nil {
		t.Fatal(err)
	}
	got, _, _ := db.ListGalleryImages("tt1", 10, 0)
	if len(got) != 3 || got[1].ID != "rm1" || got[1].URL != "http://img/new" {
		t.Fatalf("got %+v", got)
	}
}

func TestGalleryImagesAreSharedAcrossTitles(t *testing.T) {
	db := openTestDB(t)
	if err := db.AddGalleryImages("tt1", galleryImages(0, 2)); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt2", galleryImages(1, 3)); err != nil {
		t.Fatal(err)
	}
	g1, t1, _ := db.ListGalleryImages("tt1", 10, 0)
	g2, t2, _ := db.ListGalleryImages("tt2", 10, 0)
	if t1 != 2 || g1[0].ID != "rm0" || g1[1].ID != "rm1" {
		t.Fatalf("tt1 = %+v", g1)
	}
	if t2 != 2 || g2[0].ID != "rm1" || g2[1].ID != "rm2" {
		t.Fatalf("tt2 must include the image tt1 also has: %+v", g2)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image WHERE id = 'rm1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("shared image rows = %d, %v", n, err)
	}
}

func TestGalleryEpisodeTagIsPerTitle(t *testing.T) {
	db := openTestDB(t)
	still := Image{ID: "rm-still", URL: "http://still", EpID: "tt2"}
	if err := db.AddGalleryImages("tt1", []Image{still}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt2", []Image{{ID: "rm-still", URL: "http://still"}}); err != nil {
		t.Fatal(err)
	}
	g1, _, _ := db.ListGalleryImages("tt1", 10, 0)
	g2, _, _ := db.ListGalleryImages("tt2", 10, 0)
	if g1[0].EpID != "tt2" || g2[0].EpID != "" {
		t.Fatalf("tt1 = %+v, tt2 = %+v", g1, g2)
	}
}

func seriesFixture(poster Image) *FullTitle {
	return &FullTitle{
		Title: Title{
			TitleSummary: TitleSummary{ID: "tt0903747", Name: "Breaking Bad", Image: poster},
			Type:         "tvSeries", IsSeries: true, YearStart: 2008,
		},
	}
}

func TestGallerySurvivesTitleUpsert(t *testing.T) {
	db := openTestDB(t)
	poster := Image{ID: "rm1", URL: "http://img/1", Width: 99, Height: 99}
	for i := 0; i < 2; i++ {
		if err := db.UpsertFullTitle(seriesFixture(poster)); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if err := db.AddGalleryImages("tt0903747", galleryImages(0, 3)); err != nil {
				t.Fatal(err)
			}
		}
	}

	got, total, _ := db.ListGalleryImages("tt0903747", 10, 0)
	if total != 3 || len(got) != 3 {
		t.Fatalf("gallery changed by title upsert: total = %d, %+v", total, got)
	}
	// The primary image is the first row; the gallery's copy of it is the
	// same row, not a duplicate.
	if got[0].ID != "rm1" || got[1].ID != "rm0" || got[2].ID != "rm2" {
		t.Fatalf("order = %+v", got)
	}
	title, err := db.GetFullTitle("tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if title.Title.Image.ID != "rm1" || title.Title.Image.URL != "http://img/1" {
		t.Fatalf("primary image = %+v", title.Title.Image)
	}
}

func TestPrimaryImagesAreGalleryRows(t *testing.T) {
	db := openTestDB(t)
	ft := seriesFixture(Image{ID: "rm-poster", URL: "http://poster", Kind: "poster"})
	ft.Seasons = []Season{{Number: 1, Episodes: []Episode{
		{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot", Image: Image{ID: "rm-still", URL: "http://still"}}, Episode: intPtrOf(1)},
	}}}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	got, total, _ := db.ListGalleryImages("tt0903747", 10, 0)
	if total != 2 || got[0].ID != "rm-poster" || got[1].ID != "rm-still" || got[1].EpID != "tt0959621" {
		t.Fatalf("total = %d, %+v", total, got)
	}
}

func TestTitleUpsertKeepsGalleryImageKindAndCaption(t *testing.T) {
	db := openTestDB(t)
	if err := db.AddGalleryImages("tt0903747", galleryImages(1, 2)); err != nil {
		t.Fatal(err)
	}
	// The primary image carries no kind, caption or dims.
	if err := db.UpsertFullTitle(seriesFixture(Image{ID: "rm1", URL: "http://img/new"})); err != nil {
		t.Fatal(err)
	}
	got, _, _ := db.ListGalleryImages("tt0903747", 10, 0)
	want := Image{ID: "rm1", URL: "http://img/new", Width: 10, Height: 20, Kind: "still_frame", Caption: "caption 1"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPruneGalleryKeepsSyncedAndPrimaryImages(t *testing.T) {
	db := openTestDB(t)
	ft := seriesFixture(Image{ID: "rm-poster", URL: "http://poster"})
	ft.Seasons = []Season{{Number: 1, Episodes: []Episode{
		{TitleSummary: TitleSummary{ID: "tt0959621", Name: "Pilot", Image: Image{ID: "rm-still", URL: "http://still"}}, Episode: intPtrOf(1)},
	}}}
	if err := db.UpsertFullTitle(ft); err != nil {
		t.Fatal(err)
	}
	if err := db.AddGalleryImages("tt0903747", galleryImages(0, 4)); err != nil {
		t.Fatal(err)
	}
	// rm3 is also in another title's gallery, so its row must survive.
	if err := db.AddGalleryImages("tt2", galleryImages(3, 4)); err != nil {
		t.Fatal(err)
	}

	if err := db.PruneGallery("tt0903747", []string{"rm0", "rm2"}); err != nil {
		t.Fatal(err)
	}

	got, total, _ := db.ListGalleryImages("tt0903747", 10, 0)
	var ids []string
	for _, img := range got {
		ids = append(ids, img.ID)
	}
	if total != 4 || fmt.Sprint(ids) != "[rm-poster rm-still rm0 rm2]" {
		t.Fatalf("total = %d, ids = %v", total, ids)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM image WHERE id = 'rm1'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("unused image rows = %d, %v", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM image WHERE id = 'rm3'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("shared image rows = %d, %v", n, err)
	}
}
