package plugin

import (
	"context"
	"errors"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/config"
	imdbplugin "github.com/MunifTanjim/stremthru/plugin/imdb"
)

type fakeProvider struct {
	title   *imdbplugin.FullTitle
	pages   map[string]*imdbplugin.ImagePage
	seasons []imdbplugin.Season
	ratings []imdbplugin.Rating
	err     error
}

func (f *fakeProvider) GetEpisodeList(ctx context.Context, id string) ([]imdbplugin.Season, error) {
	return f.seasons, f.err
}

func (f *fakeProvider) GetRatings(ctx context.Context, ids []string) ([]imdbplugin.Rating, error) {
	return f.ratings, f.err
}

func (f *fakeProvider) GetTitle(ctx context.Context, id string) (*imdbplugin.FullTitle, error) {
	return f.title, f.err
}

func (f *fakeProvider) GetImages(ctx context.Context, id string, after string) (*imdbplugin.ImagePage, error) {
	return f.pages[after], f.err
}

func (f *fakeProvider) GetEpisodes(ctx context.Context, ids []string) ([]imdbplugin.TitleSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]imdbplugin.TitleSummary, 0, len(ids))
	for _, id := range ids {
		out = append(out, imdbplugin.TitleSummary{ID: id, Name: "ep " + id, Rating: 80})
	}
	return out, nil
}

func dialInProcess(t *testing.T, impl imdbplugin.Provider) *imdbplugin.Client {
	t.Helper()
	server := rpc.NewServer()
	if err := server.RegisterName("Plugin", &imdbplugin.Server{Impl: impl}); err != nil {
		t.Fatal(err)
	}
	serverConn, clientConn := net.Pipe()
	go server.ServeConn(serverConn)
	t.Cleanup(func() { serverConn.Close(); clientConn.Close() })
	return imdbplugin.NewClient(rpc.NewClient(clientConn))
}

func TestGetTitleViaRPC(t *testing.T) {
	rating := 77
	one := 1
	want := &imdbplugin.FullTitle{
		Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: "tt0903747",
			Name:   "Breaking Bad",
			Rating: rating, Votes: 1000}, Type: "tvSeries", IsSeries: true,
			YearStart: 2008,

			Genres: []string{"Crime"},
			Akas:   []imdbplugin.Aka{{Text: "Breaking Bad", Country: "US"}},
		},
		Seasons: []imdbplugin.Season{{
			Number: 1,
			Episodes: []imdbplugin.Episode{
				{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Name: "Pilot"}, Episode: &one},
			},
		}},
	}
	client := dialInProcess(t, &fakeProvider{title: want})

	got, err := client.GetTitle(context.Background(), "tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title.Name != "Breaking Bad" || got.Title.Rating != 77 {
		t.Fatalf("title mismatch: %+v", got.Title)
	}
	if len(got.Seasons) != 1 || len(got.Seasons[0].Episodes) != 1 || got.Seasons[0].Episodes[0].ID != "tt0959621" {
		t.Fatalf("episodes mismatch: %+v", got.Seasons)
	}
}

func TestGetTitleNotFound(t *testing.T) {
	client := dialInProcess(t, &fakeProvider{err: imdbplugin.ErrNotFound})
	_, err := client.GetTitle(context.Background(), "tt0000000")
	if !errors.Is(err, imdbplugin.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetImagesViaRPC(t *testing.T) {
	client := dialInProcess(t, &fakeProvider{pages: map[string]*imdbplugin.ImagePage{
		"": {
			Images: []imdbplugin.Image{{ID: "rm1", URL: "http://img/1", Width: 1200, Height: 1800}},
			Next:   "cursor1",
		},
		"cursor1": {
			Images: []imdbplugin.Image{{ID: "rm2", URL: "http://img/2", Width: 1800, Height: 1200}},
		},
	}})

	var ids []string
	after := ""
	for {
		page, err := client.GetImages(context.Background(), "tt0111161", after)
		if err != nil {
			t.Fatal(err)
		}
		for _, img := range page.Images {
			ids = append(ids, img.ID)
		}
		if page.Next == "" {
			break
		}
		after = page.Next
	}
	if len(ids) != 2 || ids[0] != "rm1" || ids[1] != "rm2" {
		t.Fatalf("ids = %v", ids)
	}
}

func TestGetImagesNotFound(t *testing.T) {
	client := dialInProcess(t, &fakeProvider{err: ErrNotFound})
	_, err := client.GetImages(context.Background(), "tt0000000", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

const pluginChildEnv = "STREMTHRU_TEST_IMDB_PLUGIN_CHILD"

// TestMain lets the test binary double as the plugin process.
func TestMain(m *testing.M) {
	if os.Getenv(pluginChildEnv) != "" {
		imdbplugin.Serve(&fakeProvider{title: &imdbplugin.FullTitle{
			Title: imdbplugin.Title{TitleSummary: imdbplugin.TitleSummary{ID: "tt0903747", Name: "Breaking Bad"}},
		}})
		return
	}
	os.Exit(m.Run())
}

func TestClientRelaunchesAfterPluginExit(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(exe, filepath.Join(dir, "imdb")); err != nil {
		t.Fatal(err)
	}
	t.Setenv(pluginChildEnv, "1")
	oldDir := config.PluginDir
	config.PluginDir = dir
	t.Cleanup(func() { config.PluginDir = oldDir })

	c := &Client{}
	t.Cleanup(c.Close)

	if _, err := c.GetTitle(context.Background(), "tt0903747"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	first := c.client
	first.Kill()

	got, err := c.GetTitle(context.Background(), "tt0903747")
	if err != nil {
		t.Fatalf("call after plugin exit: %v", err)
	}
	if got.Title.Name != "Breaking Bad" {
		t.Fatalf("title = %+v", got.Title)
	}
	if c.client == first {
		t.Fatal("plugin process was not relaunched")
	}
}

func TestGetEpisodesViaRPC(t *testing.T) {
	client := dialInProcess(t, &fakeProvider{})
	got, err := client.GetEpisodes(context.Background(), []string{"tt1", "tt2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].ID != "tt2" || got[1].Name != "ep tt2" || got[1].Rating != 80 {
		t.Fatalf("got %+v", got)
	}
}

func TestGetEpisodeListViaRPC(t *testing.T) {
	one := 1
	want := []imdbplugin.Season{{Number: 1, Episodes: []imdbplugin.Episode{
		{TitleSummary: imdbplugin.TitleSummary{ID: "tt0959621", Description: "After receiving a devastating diagnosis, a struggling high school chemistry teacher partners with a former student to produce methamphetamine and secure his family's financial future."}, Episode: &one},
	}}}
	got, err := dialInProcess(t, &fakeProvider{seasons: want}).GetEpisodeList(context.Background(), "tt0903747")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGetEpisodeListNotFound(t *testing.T) {
	_, err := dialInProcess(t, &fakeProvider{err: imdbplugin.ErrNotFound}).GetEpisodeList(context.Background(), "tt0000000")
	if !errors.Is(err, imdbplugin.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetRatingsViaRPC(t *testing.T) {
	want := []imdbplugin.Rating{{ID: "tt0959621", Rating: 90, Votes: 41000}}
	got, err := dialInProcess(t, &fakeProvider{ratings: want}).GetRatings(context.Background(), []string{"tt0959621"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
