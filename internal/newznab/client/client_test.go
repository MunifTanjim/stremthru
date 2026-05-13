package newznab_client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
	"github.com/stretchr/testify/assert"
)

func newTestClient(server *httptest.Server) *Client {
	return NewClient(&ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		APIKey:     "default-key",
		UserAgent:  "test-agent",
	})
}

func newMockServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(body))
	}))
}

const validRSSResponse = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:atom="http://www.w3.org/2005/Atom" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/" version="2.0">
<channel>
  <title>Test Indexer</title>
	<item>
    <title>Test.2025.1080p.BDRip.DDP7.1.x265</title>
		<guid isPermaLink="true">http://example.com/details/1337</guid>
		<link>http://example.com/getnzb/1337.nzb&amp;apikey=7001</link>
		<comments>http://example.com/details/1337#comments</comments>
		<pubDate>Thu, 01 Jan 2026 00:00:00 +0000</pubDate>
		<category>Movies > HD</category>
		<description>Test.2025.1080p.BDRip.DDP7.1.x265</description>
		<enclosure url="http://example.com/getnzb/1337.nzb&amp;apikey=7001" length="13377001" type="application/x-nzb"/>
		<newznab:attr name="category" value="2000"/>
		<newznab:attr name="category" value="2040"/>
		<newznab:attr name="size" value="13377001"/>
		<newznab:attr name="guid" value="1337"/>
		<newznab:attr name="sha1" value="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"/>
		<newznab:attr name="files" value="9"/>
		<newznab:attr name="poster" value="Poster"/>
		<newznab:attr name="imdb" value="0000000"/>
		<newznab:attr name="grabs" value="999"/>
		<newznab:attr name="comments" value="0"/>
		<newznab:attr name="password" value="0"/>
		<newznab:attr name="nfo" value="1"/>
		<newznab:attr name="info" value="http://example.com/api?t=getnfo&amp;id=13&amp;apikey=7001&amp;raw=1"/>
		<newznab:attr name="usenetdate" value="Wed, 01 Jan 2025 00:00:00 +0000"/>
		<newznab:attr name="group" value="alt.binaries.boneless"/>
	</item>
</channel>
</rss>`

func TestSearch_ValidResponse(t *testing.T) {
	server := newMockServer(validRSSResponse)
	defer server.Close()

	client := newTestClient(server)
	query := url.Values{}
	query.Set("t", "search")
	query.Set("q", "test")

	results, err := client.Search(query, http.Header{})
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	item := results[0]
	assert.Equal(t, "Test.2025.1080p.BDRip.DDP7.1.x265", item.Title)
	assert.Equal(t, "http://example.com/details/1337", item.GUID)
	assert.Equal(t, int64(13377001), item.Size)
	assert.Equal(t, "http://example.com/getnzb/1337.nzb&apikey=7001", item.DownloadLink)
	assert.Equal(t, []string{"2000", "2040"}, item.Categories)
	assert.Equal(t, "tt0000000", item.IMDB)
	assert.Equal(t, 9, item.Files)
	assert.Equal(t, "Poster", item.Poster)
	assert.Equal(t, "alt.binaries.boneless", item.Group)
	assert.Equal(t, 999, item.Grabs)
	assert.Equal(t, 0, item.Comments)
	assert.False(t, item.Password)
	assert.False(t, item.InnerArchive)
	assert.Equal(t, util.MustParseTime(znab.TimeFormat, "Wed, 01 Jan 2025 00:00:00 +0000"), item.Date)
}
