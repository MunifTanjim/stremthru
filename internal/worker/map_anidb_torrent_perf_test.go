package worker

import (
	"slices"
	"strings"
	"testing"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/agnivade/levenshtein"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
	"github.com/stretchr/testify/assert"
)

// referenceSortAniDBTitles reproduces the pre-optimization comparator verbatim,
// so TestSortAniDBTitles can assert the optimized decorate-sort-undecorate
// produces identical ordering across every branch.
func referenceSortAniDBTitles(titles anidb.AniDBTitles, tInfo torrent_info.TorrentInfo, tYear string) anidb.AniDBTitles {
	tTitle := strings.ToLower(tInfo.Title)
	tTorrentTitle := strings.ToLower(tInfo.TorrentTitle)
	if len(tInfo.ReleaseTypes) == 1 {
		releaseType := strings.ToLower(tInfo.ReleaseTypes[0])
		if !strings.Contains(tTorrentTitle, releaseType) {
			switch releaseType {
			case "ova":
				releaseType = "oav"
			case "oad":
				releaseType = "oda"
			default:
				releaseType = ""
			}
		}
		if releaseType != "" {
			tTitle += " " + releaseType
		}
	}

	slices.SortStableFunc(titles, func(a, b anidb.AniDBTitle) int {
		if tYear != "" {
			if a.Year == b.Year {
				return levenshtein.ComputeDistance(tTitle, strings.ToLower(a.Value)) - levenshtein.ComputeDistance(tTitle, strings.ToLower(b.Value))
			}
			if a.Year == tYear {
				return -1
			}
			if b.Year == tYear {
				return 1
			}
		}
		return levenshtein.ComputeDistance(tTitle, strings.ToLower(a.Value)) - levenshtein.ComputeDistance(tTitle, strings.ToLower(b.Value))
	})

	return titles
}

func TestSortAniDBTitles(t *testing.T) {
	values := func(titles anidb.AniDBTitles) []string {
		out := make([]string, len(titles))
		for i := range titles {
			out[i] = titles[i].Value
		}
		return out
	}

	t.Run("orders by ascending edit distance when no year", func(t *testing.T) {
		titles := anidb.AniDBTitles{
			{Value: "bleach"},
			{Value: "one piece"},
			{Value: "one pieces"},
		}
		tInfo := torrent_info.TorrentInfo{Title: "one piece"}

		got := sortAniDBTitles(titles, tInfo, "")

		assert.Equal(t, []string{"one piece", "one pieces", "bleach"}, values(got))
	})

	t.Run("prefers matching year over closer text match", func(t *testing.T) {
		titles := anidb.AniDBTitles{
			{Value: "naruto", Year: "2000"},
			{Value: "totally unrelated show", Year: "2019"},
		}
		tInfo := torrent_info.TorrentInfo{Title: "naruto"}

		got := sortAniDBTitles(titles, tInfo, "2019")

		assert.Equal(t, []string{"totally unrelated show", "naruto"}, values(got))
	})

	t.Run("is stable for equal keys", func(t *testing.T) {
		titles := anidb.AniDBTitles{
			{TId: "first", Value: "x"},
			{TId: "second", Value: "x"},
		}
		tInfo := torrent_info.TorrentInfo{Title: "x"}

		got := sortAniDBTitles(titles, tInfo, "")

		assert.Equal(t, []string{"first", "second"}, []string{got[0].TId, got[1].TId})
	})

	t.Run("applies release type alias to the match title", func(t *testing.T) {
		titles := anidb.AniDBTitles{
			{Value: "gundam"},
			{Value: "gundam oav"},
		}
		tInfo := torrent_info.TorrentInfo{
			Title:        "gundam",
			TorrentTitle: "gundam",
			ReleaseTypes: torrent_info.CommaSeperatedString{"OVA"},
		}

		got := sortAniDBTitles(titles, tInfo, "")

		assert.Equal(t, []string{"gundam oav", "gundam"}, values(got))
	})

	t.Run("matches original comparator across branches", func(t *testing.T) {
		cases := []struct {
			name   string
			titles anidb.AniDBTitles
			tInfo  torrent_info.TorrentInfo
			tYear  string
		}{
			{
				name:   "no year: distance only",
				titles: anidb.AniDBTitles{{Value: "bleach"}, {Value: "one piece"}, {Value: "one pieces"}},
				tInfo:  torrent_info.TorrentInfo{Title: "one piece"},
				tYear:  "",
			},
			{
				name: "year set, all candidates share the year: falls through to distance",
				titles: anidb.AniDBTitles{
					{Value: "naruto shippuden", Year: "2019"},
					{Value: "naruto", Year: "2019"},
					{Value: "bleach", Year: "2019"},
				},
				tInfo: torrent_info.TorrentInfo{Title: "naruto"},
				tYear: "2019",
			},
			{
				name: "year set, differing years, neither matches: falls through to distance",
				titles: anidb.AniDBTitles{
					{Value: "naruto", Year: "2000"},
					{Value: "bleach", Year: "2010"},
				},
				tInfo: torrent_info.TorrentInfo{Title: "naruto"},
				tYear: "2019",
			},
			{
				name: "year set, matching year must sort ahead regardless of input position",
				titles: anidb.AniDBTitles{
					{Value: "naruto", Year: "2000"},
					{Value: "totally unrelated show", Year: "2019"},
					{Value: "another unrelated one", Year: "2005"},
				},
				tInfo: torrent_info.TorrentInfo{Title: "naruto"},
				tYear: "2019",
			},
			{
				name: "year set, mix of matching, shared and other years",
				titles: anidb.AniDBTitles{
					{TId: "1", Value: "a show", Year: "2019"},
					{TId: "2", Value: "b show", Year: "2005"},
					{TId: "3", Value: "c show", Year: "2005"},
					{TId: "4", Value: "naruto", Year: "2000"},
					{TId: "5", Value: "d show", Year: "2019"},
				},
				tInfo: torrent_info.TorrentInfo{Title: "naruto"},
				tYear: "2019",
			},
			{
				name:   "release type alias with year set",
				titles: anidb.AniDBTitles{{Value: "gundam", Year: "2000"}, {Value: "gundam oav", Year: "2019"}},
				tInfo: torrent_info.TorrentInfo{
					Title:        "gundam",
					TorrentTitle: "gundam",
					ReleaseTypes: torrent_info.CommaSeperatedString{"OVA"},
				},
				tYear: "2019",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := sortAniDBTitles(slices.Clone(tc.titles), tc.tInfo, tc.tYear)
				want := referenceSortAniDBTitles(slices.Clone(tc.titles), tc.tInfo, tc.tYear)
				assert.Equal(t, want, got)
			})
		}
	})
}

// TestMaxAniDBTitleMatchRatioMatchesUQRatio pins the optimized match-ratio loop
// to the exact behavior of the original per-candidate fuzzy.UQRatio loop.
func TestMaxAniDBTitleMatchRatioMatchesUQRatio(t *testing.T) {
	// reference reproduces the pre-optimization loop verbatim.
	reference := func(tTitle string, titles anidb.AniDBTitles) int {
		ratio := 0
		for i := range titles {
			ratio = max(ratio, fuzzy.UQRatio(tTitle, titles[i].Value))
			if ratio >= 85 {
				break
			}
		}
		return ratio
	}

	titlesOf := func(vals ...string) anidb.AniDBTitles {
		titles := make(anidb.AniDBTitles, len(vals))
		for i, v := range vals {
			titles[i] = anidb.AniDBTitle{Value: v}
		}
		return titles
	}

	cases := []struct {
		name   string
		tTitle string
		titles anidb.AniDBTitles
	}{
		{"exact match short circuits", "one piece", titlesOf("bleach", "one piece", "naruto")},
		{"exact match first", "naruto", titlesOf("naruto", "totally different")},
		{"no strong match", "attack on titan", titlesOf("bleach", "naruto")},
		{"near match", "one piece", titlesOf("one pieces")},
		{"empty query title", "", titlesOf("one piece", "naruto")},
		{"empty candidate value in middle", "one piece", titlesOf("", "one piece")},
		{"all empty candidates", "one piece", titlesOf("", "")},
		{"no candidates", "one piece", titlesOf()},
		{"non-ascii preserved", "きめつのやいば", titlesOf("きめつのやいば", "naruto")},
		{"punctuation and case", "Re:ZERO", titlesOf("re zero", "re:zero!!!")},
		{"whitespace only candidate", "one piece", titlesOf("   ", "one piece")},
		{"punctuation only candidate stays non-empty", "one piece", titlesOf("!!!", "one piece")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := reference(tc.tTitle, tc.titles)
			got := maxAniDBTitleMatchRatio(tc.tTitle, tc.titles)
			assert.Equal(t, want, got)
		})
	}
}
