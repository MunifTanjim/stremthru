package torznab

import (
	"testing"

	"github.com/MunifTanjim/stremthru/internal/torrent_info"
)

func TestToFeedItemPrivateTorrentHidesPeerCounts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		isPrivate  bool
		wantCounts bool
	}{
		{name: "private", isPrivate: true, wantCounts: false},
		{name: "public", isPrivate: false, wantCounts: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := torrent_info.TorrentInfo{
				Hash:         "0123456789abcdef0123456789abcdef01234567",
				TorrentTitle: "Example.Movie.2026.1080p",
				Private:      tc.isPrivate,
				Seeders:      42,
				Leechers:     7,
			}
			item := toFeedItem(info, "tt1234567")
			if item.Title != info.TorrentTitle || item.InfoHash != info.Hash {
				t.Fatal("private torrent metadata must remain visible")
			}
			if tc.wantCounts && (item.Seeders != 42 || item.Leechers != 7) {
				t.Fatalf("expected original peer counts, got seeders=%d leechers=%d", item.Seeders, item.Leechers)
			}
			if !tc.wantCounts && (item.Seeders != 0 || item.Leechers != 0) {
				t.Fatalf("private torrent exposes peer counts: seeders=%d leechers=%d", item.Seeders, item.Leechers)
			}
			foundSeeders, foundLeechers := false, false
			for _, attr := range item.toChannelItem().Attributes {
				switch attr.Name {
				case "seeders":
					foundSeeders = true
				case "leechers":
					foundLeechers = true
				}
			}
			if foundSeeders != tc.wantCounts || foundLeechers != tc.wantCounts {
				t.Fatalf("unexpected peer attributes: seeders=%t leechers=%t", foundSeeders, foundLeechers)
			}
		})
	}
}
