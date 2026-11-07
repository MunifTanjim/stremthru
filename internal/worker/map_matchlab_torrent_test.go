package worker

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
)

func matchLabPointer(s string) *string { return &s }
func mkSID(first, last string) matchlab.Response {
	return matchlab.Response{SIDStart: matchLabPointer(first), SIDEnd: matchLabPointer(last)}
}

func TestMatchLabSingleSeasonEnumeratesEpisodeInterval(t *testing.T) {
	pack := torrent_info.TorrentInfo{
		Hash: "abc", TorrentTitle: "Supernatural.S02",
		Seasons: torrent_info.CommaSeperatedInt{2},
	}
	whole := mkSID("tt0460681:2:1", "tt0460681:2:22")
	files := []matchLabFileResult{
		{Path: "/Supernatural/S02E01.mkv", Match: mkSID("tt0460681:2:1", "tt0460681:2:1")},
		{Path: "/Supernatural/S02E03.mkv", Match: mkSID("tt0460681:2:3", "tt0460681:2:3")},
	}
	out := buildMatchLabMapping(pack, whole, nil, files, nil)
	if !reflect.DeepEqual(out.IMDbIDs, []string{"tt0460681"}) {
		t.Fatalf("IMDb IDs: %v", out.IMDbIDs)
	}
	if !reflect.DeepEqual(out.Seasons, []int{2}) {
		t.Fatalf("seasons: %v", out.Seasons)
	}
	if !reflect.DeepEqual(out.Episodes, []int{1, 2, 3}) {
		t.Fatalf("episodes: %v", out.Episodes)
	}
	if out.Category != torrent_info.TorrentInfoCategorySeries {
		t.Fatal("expected series")
	}
}

func TestMatchLabSingleSeasonFromVerifiedPackEndpoints(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "abc", TorrentTitle: "Anime.Title.1-7"}
	whole := mkSID("tt1234567:1:1", "tt1234567:1:7")
	out := buildMatchLabMapping(pack, whole, nil, nil, nil)
	if !reflect.DeepEqual(out.Seasons, []int{1}) {
		t.Fatalf("seasons: %v", out.Seasons)
	}
	if !reflect.DeepEqual(out.Episodes, []int{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("episodes: %v", out.Episodes)
	}
}

func TestMatchLabMultiSeasonDropsUnscopedEpisodes(t *testing.T) {
	pack := torrent_info.TorrentInfo{
		Hash: "abc", TorrentTitle: "Supernatural S01-S03",
		Seasons:  torrent_info.CommaSeperatedInt{1, 2, 3},
		Episodes: torrent_info.CommaSeperatedInt{1, 2, 3},
	}
	whole := mkSID("tt0460681:1:1", "tt0460681:3:22")
	files := []matchLabFileResult{
		{Path: "/Supernatural/S01E01.mkv", Match: mkSID("tt0460681:1:1", "tt0460681:1:1")},
		{Path: "/Supernatural/S03E22.mkv", Match: mkSID("tt0460681:3:22", "tt0460681:3:22")},
	}
	out := buildMatchLabMapping(pack, whole, nil, files, nil)
	if !reflect.DeepEqual(out.Seasons, []int{1, 2, 3}) {
		t.Fatalf("seasons: %v", out.Seasons)
	}
	if len(out.Episodes) != 0 {
		t.Fatalf("multi-season must leave episodes blank: %v", out.Episodes)
	}
}

func TestMatchLabMixedAnimeIdentitiesAreIndependent(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "abc", TorrentTitle: "Anime.Collection"}
	files := []matchLabFileResult{
		{
			Path: "/A/01.mkv",
			Match: matchlab.Response{
				SIDStart: matchLabPointer("tt1234567:1:1"), SIDEnd: matchLabPointer("tt1234567:1:1"),
				ASIDStart: matchLabPointer("100:1"), ASIDEnd: matchLabPointer("100:1"),
			},
		},
		{
			Path: "/B/01.mkv",
			Match: matchlab.Response{
				SIDStart: matchLabPointer("tt2345678:1:1"), SIDEnd: matchLabPointer("tt2345678:1:1"),
				ASIDStart: matchLabPointer("200:1"), ASIDEnd: matchLabPointer("200:1"),
			},
		},
	}
	out := buildMatchLabMapping(pack, matchlab.Response{}, nil, files, map[string]int{"100": 1, "200": 2})
	if !reflect.DeepEqual(out.IMDbIDs, []string{"tt1234567", "tt2345678"}) {
		t.Fatalf("IMDb IDs: %v", out.IMDbIDs)
	}
	if len(out.Seasons) != 0 || len(out.Episodes) != 0 {
		t.Fatalf("mixed collection cannot have one global episode range: %+v", out)
	}
	if len(out.AniDB) < 2 {
		t.Fatalf("missing anime mappings: %+v", out.AniDB)
	}
	seen := map[string]bool{}
	for _, a := range out.AniDB {
		if a.SeasonType != anidb.TorrentSeasonTypeAnime {
			continue
		}
		seen[a.TId] = true
		if a.EpisodeStart != 1 || a.EpisodeEnd != 1 {
			t.Fatalf("unexpected AniDB range: %+v", a)
		}
	}
	if !seen["100"] || !seen["200"] {
		t.Fatalf("missing independent anime IDs: %v", seen)
	}
}

func TestMatchLabAnonymousAnimeSeasonSentinel(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "abc", TorrentTitle: "Anime"}
	match := matchlab.Response{ASIDStart: matchLabPointer("42:7"), ASIDEnd: matchLabPointer("42:7")}
	out := buildMatchLabMapping(pack, match, nil, nil, nil)
	found := false
	for _, a := range out.AniDB {
		if a.SeasonType == anidb.TorrentSeasonTypeAnime && a.TId == "42" {
			found = true
			if a.Season != -1 {
				t.Fatalf("expected -1 unknown AniDB season, got %d", a.Season)
			}
		}
	}
	if !found {
		t.Fatalf("no anime mapping: %+v", out.AniDB)
	}
}

func TestMatchLabCombinedPath(t *testing.T) {
	file := torrent_stream.File{Path: "/Show/S02E04.mkv"}
	got := matchlabFilePath("Supernatural S01-S12", file)
	if got != "Supernatural S01-S12/Show/S02E04.mkv" {
		t.Fatalf("wrong combined title: %s", got)
	}
	if !strings.Contains(got, "S02E04") {
		t.Fatal("child episode was removed")
	}
}

func TestMatchLabNoInventedFileSID(t *testing.T) {
	r := mkSID("tt0460681:1:1", "tt0460681:1:7")
	if r.SingleSID() != "" {
		t.Fatal("cannot assign a range to a single file")
	}
}

func TestMatchLabVerifiedAnimeSegmentsDoNotFillGaps(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "xyz", TorrentTitle: "Anime.E1-E10"}
	whole := matchlab.Response{
		SIDStart: matchLabPointer("tt1234567:1:1"), SIDEnd: matchLabPointer("tt1234567:1:10"),
		ASIDStart: matchLabPointer("42:1"), ASIDEnd: matchLabPointer("42:10"),
	}
	segments := []matchlab.EpisodeSegment{
		{Season: 1, StartEpisode: 1, EndEpisode: 3, AniDBID: "42", StartAnimeEpisode: 1, EndAnimeEpisode: 3},
		{Season: 1, StartEpisode: 8, EndEpisode: 10, AniDBID: "42", StartAnimeEpisode: 8, EndAnimeEpisode: 10},
	}
	out := buildMatchLabMapping(pack, whole, segments, nil, map[string]int{"42": 1})
	var exact []int
	for _, a := range out.AniDB {
		if a.TId == "42" && a.SeasonType == anidb.TorrentSeasonTypeAnime {
			exact = append(exact, a.Episodes...)
		}
	}
	if !reflect.DeepEqual(exact, []int{1, 2, 3, 8, 9, 10}) {
		t.Fatalf("crosswalk gap incorrectly filled: %v", exact)
	}
}

func TestMatchLabMixedSeasonSegmentsPreserveAniDBIdentity(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "xyz", TorrentTitle: "One Piece"}
	files := []matchLabFileResult{
		{Path: "/ArcA/010.mkv",
			Match: matchlab.Response{ASIDStart: matchLabPointer("69:10"), ASIDEnd: matchLabPointer("69:10")},
			Segments: []matchlab.EpisodeSegment{
				{Season: 3, StartEpisode: 7, EndEpisode: 7, AniDBID: "69", StartAnimeEpisode: 10, EndAnimeEpisode: 10},
			},
		},
		{Path: "/ArcB/001.mkv",
			Match: matchlab.Response{ASIDStart: matchLabPointer("70:1"), ASIDEnd: matchLabPointer("70:1")},
			Segments: []matchlab.EpisodeSegment{
				{Season: 4, StartEpisode: 1, EndEpisode: 1, AniDBID: "70", StartAnimeEpisode: 1, EndAnimeEpisode: 1},
			},
		},
	}
	out := buildMatchLabMapping(pack, matchlab.Response{}, nil, files, map[string]int{"69": 1, "70": 2})
	ids := map[string]bool{}
	for _, a := range out.AniDB {
		if a.SeasonType == anidb.TorrentSeasonTypeAnime {
			ids[a.TId] = true
		}
	}
	if !ids["69"] || !ids["70"] {
		t.Fatalf("mixed AniDB title IDs missing: %v", ids)
	}
}

func TestMatchLabAbsoluteAnimeSegments(t *testing.T) {
	pack := torrent_info.TorrentInfo{Hash: "abs-hash", TorrentTitle: "One Piece E768-E770"}
	first, last := 768, 770
	segments := []matchlab.EpisodeSegment{{
		Season: 19, StartEpisode: 2, EndEpisode: 4,
		AniDBID: "69", StartAnimeEpisode: 2, EndAnimeEpisode: 4,
		StartAbsolute: &first, EndAbsolute: &last,
	}}
	r := matchlab.Response{
		SIDStart:  matchLabPointer("tt1234567:19:2"),
		SIDEnd:    matchLabPointer("tt1234567:19:4"),
		ASIDStart: matchLabPointer("69:2"),
		ASIDEnd:   matchLabPointer("69:4"),
	}
	mapped := buildMatchLabMapping(pack, r, segments, nil, map[string]int{"69": 1})
	found := false
	for _, row := range mapped.AniDB {
		if row.TId == "69" && row.SeasonType == anidb.TorrentSeasonTypeAbsolute {
			found = true
			if row.Season != 19 || !reflect.DeepEqual([]int(row.Episodes), []int{768, 769, 770}) {
				t.Fatalf("wrong absolute season/episodes: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("missing absolute AniDB mapping")
	}
}

func TestMatchLabRepresentativeUsesHighestEpisodeInOneSeason(t *testing.T) {
	torrent := torrent_info.TorrentInfo{Hash: "h", TorrentTitle: "Example Show S01"}
	files := []matchLabFileResult{
		{Path: "/Example Show S01E02.mkv", Match: mkSID("tt1111111:1:2", "tt1111111:1:2")},
		{Path: "/Example Show S01E17.mkv", Match: mkSID("tt2222222:1:17", "tt2222222:1:17")},
	}
	if got := representativeSeriesID(files); got != "tt2222222" {
		t.Fatalf("highest episode should identify series, got %q", got)
	}
	mapped := buildMatchLabMapping(torrent, matchlab.Response{}, nil, files, nil)
	if !reflect.DeepEqual(mapped.IMDbIDs, []string{"tt2222222"}) {
		t.Fatalf("unexpected IMDb IDs: %v", mapped.IMDbIDs)
	}
}

func TestMatchLabRepresentativeUsesHighestSeasonInMultiSeasonPack(t *testing.T) {
	files := []matchLabFileResult{
		{Path: "/S01E24.mkv", Match: mkSID("tt1111111:1:24", "tt1111111:1:24")},
		{Path: "/S03E01.mkv", Match: mkSID("tt2222222:3:1", "tt2222222:3:1")},
		{Path: "/S02E12.mkv", Match: mkSID("tt3333333:2:12", "tt3333333:2:12")},
	}
	if got := representativeSeriesID(files); got != "tt2222222" {
		t.Fatalf("highest season should identify series, got %q", got)
	}
}

func TestMatchLabTimeoutGrowsForLargePacks(t *testing.T) {
	if got := matchLabTorrentTimeout(1); got != 2*time.Minute {
		t.Fatalf("small file timeout = %s", got)
	}
	if got := matchLabTorrentTimeout(800); got <= 2*time.Minute {
		t.Fatalf("large pack timeout too short = %s", got)
	}
	if got := matchLabTorrentTimeout(50000); got != 20*time.Minute {
		t.Fatalf("timeout must be capped: %s", got)
	}
}

func TestMatchLabVerifiedSeasonsOverrideStaleParser(t *testing.T) {
	// A previously parsed pack claimed S01-S04, but verified file SIDs
	// establish only S01-S03. The extra season must not survive remapping.
	pack := torrent_info.TorrentInfo{
		Hash: "override", TorrentTitle: "Example Show S01-S04",
		Seasons:  torrent_info.CommaSeperatedInt{1, 2, 3, 4},
		Episodes: torrent_info.CommaSeperatedInt{77, 78},
	}
	files := []matchLabFileResult{
		{Path: "/S01E01.mkv", Match: mkSID("tt1234567:1:1", "tt1234567:1:1")},
		{Path: "/S03E08.mkv", Match: mkSID("tt1234567:3:8", "tt1234567:3:8")},
	}
	mapped := buildMatchLabMapping(pack, matchlab.Response{}, nil, files, nil)
	if !reflect.DeepEqual(mapped.Seasons, []int{1, 2, 3}) {
		t.Fatalf("stale parser season survived: %v", mapped.Seasons)
	}
	if len(mapped.Episodes) != 0 {
		t.Fatalf("multi-season torrent cannot retain stale parser episodes: %v", mapped.Episodes)
	}
}

func TestMatchLabVerifiedSingleSeasonOverridesStaleValues(t *testing.T) {
	pack := torrent_info.TorrentInfo{
		Hash: "override-single", TorrentTitle: "Example Show S05",
		Seasons:  torrent_info.CommaSeperatedInt{5},
		Episodes: torrent_info.CommaSeperatedInt{37, 38, 39},
	}
	files := []matchLabFileResult{
		{Path: "/S02E03.mkv", Match: mkSID("tt1234567:2:3", "tt1234567:2:3")},
		{Path: "/S02E05.mkv", Match: mkSID("tt1234567:2:5", "tt1234567:2:5")},
	}
	mapped := buildMatchLabMapping(pack, matchlab.Response{}, nil, files, nil)
	if !reflect.DeepEqual(mapped.Seasons, []int{2}) {
		t.Fatalf("expected corrected verified season, got %v", mapped.Seasons)
	}
	if !reflect.DeepEqual(mapped.Episodes, []int{3, 4, 5}) {
		t.Fatalf("expected corrected verified episodes, got %v", mapped.Episodes)
	}
}

func TestMatchLabUnscopedIdentityKeepsParsedSeasonOnlyWhenUnverified(t *testing.T) {
	// A bare IMDb ID carries no season/episode evidence. Do not erase
	// already parsed coordinates until MatchLab verifies replacements.
	pack := torrent_info.TorrentInfo{
		Hash: "unscoped", TorrentTitle: "Example Show S02E07",
		Seasons:  torrent_info.CommaSeperatedInt{2},
		Episodes: torrent_info.CommaSeperatedInt{7},
	}
	mapped := buildMatchLabMapping(pack, mkSID("tt1234567", "tt1234567"), nil, nil, nil)
	if !reflect.DeepEqual(mapped.Seasons, []int{2}) || !reflect.DeepEqual(mapped.Episodes, []int{7}) {
		t.Fatalf("bare IMDb ID must not discard parser coordinates: seasons=%v, episodes=%v", mapped.Seasons, mapped.Episodes)
	}
}
