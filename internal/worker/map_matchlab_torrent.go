package worker

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/matchlab"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
)

// A MatchLab pass owns mapping for the entire torrent hash. Per-file identities
// remain independent, because a single torrent can contain multiple anime.
type matchLabFileResult struct {
	Path     string
	Match    matchlab.Response
	Segments []matchlab.EpisodeSegment
}

type matchLabMapping struct {
	IMDbIDs  []string
	Seasons  []int
	Episodes []int
	Category torrent_info.TorrentInfoCategory
	Files    []matchLabFileResult
	AniDB    []anidb.AniDBTorrent
}

type matchLabSeasonEpisode struct {
	season  int
	episode int
}

type matchLabAnimeGroup struct {
	id      string
	season  int
	kind    anidb.TorrentSeasonType
	episode map[int]struct{}
}

// mergeMatchLabIdentity collects only endpoints the server has verified.
// A partial range does not license inventing all its intermediate episodes.
func mergeMatchLabIdentity(ids map[string]struct{}, coords map[string][]matchLabSeasonEpisode, r matchlab.Response) {
	for _, sid := range []*string{r.SIDStart, r.SIDEnd} {
		if sid == nil {
			continue
		}
		id, season, episode, hasCoords, ok := matchlab.ParseSID(*sid)
		if !ok {
			continue
		}
		ids[id] = struct{}{}
		if hasCoords {
			coords[id] = append(coords[id], matchLabSeasonEpisode{season, episode})
		}
	}
}

// addRange promotes only a verified, bounded same-season interval. At the
// torrent level, the user wants episodes 1..N, never a comma list across
// multiple seasons where the episode number no longer has one meaning.
func addMatchLabRange(coords map[string][]matchLabSeasonEpisode, r matchlab.Response) {
	if r.SIDStart == nil || r.SIDEnd == nil {
		return
	}
	firstID, firstS, firstE, a, ok1 := matchlab.ParseSID(*r.SIDStart)
	lastID, lastS, lastE, b, ok2 := matchlab.ParseSID(*r.SIDEnd)
	if !ok1 || !ok2 || !a || !b || firstID != lastID ||
		firstS != lastS || lastE < firstE || lastE-firstE > 2047 {
		return
	}
	for ep := firstE; ep <= lastE; ep++ {
		coords[firstID] = append(coords[firstID], matchLabSeasonEpisode{firstS, ep})
	}
}

func addMatchLabAnime(groups map[string]*matchLabAnimeGroup, ids map[string]int, r matchlab.Response) {
	if r.ASIDStart == nil && r.ASIDEnd == nil {
		return
	}
	// A matched SID's season is mapped to TVDB-style season coordinates
	// for anime. Keep it distinct from AniDB season numbering.
	tvSeason := -1
	if r.SIDStart != nil {
		_, s, _, found, valid := matchlab.ParseSID(*r.SIDStart)
		if found && valid {
			tvSeason = s
		}
	}
	values := []*string{r.ASIDStart, r.ASIDEnd}
	for _, raw := range values {
		if raw == nil {
			continue
		}
		id, ep, valid := matchlab.ParseASID(*raw)
		if !valid {
			continue
		}
		animeSeason, present := ids[id]
		if !present || animeSeason < 0 {
			animeSeason = -1 // unknown, not an inferred season number
		}
		key := fmt.Sprintf("%s:ani:%d", id, animeSeason)
		g := groups[key]
		if g == nil {
			g = &matchLabAnimeGroup{id: id, season: animeSeason, kind: anidb.TorrentSeasonTypeAnime, episode: map[int]struct{}{}}
			groups[key] = g
		}
		g.episode[ep] = struct{}{}
	}
	if r.ASIDStart != nil && r.ASIDEnd != nil {
		firstID, first, ok1 := matchlab.ParseASID(*r.ASIDStart)
		lastID, last, ok2 := matchlab.ParseASID(*r.ASIDEnd)
		if ok1 && ok2 && firstID == lastID && last >= first && last-first <= 2047 {
			s, present := ids[firstID]
			if !present || s < 0 {
				s = -1
			}
			g := groups[fmt.Sprintf("%s:ani:%d", firstID, s)]
			// The /match response only exposes verified endpoints. Do not
			// interpolate an anime range when mapping segments may be missing.
			if g != nil && first == last {
				g.episode[first] = struct{}{}
			}
		}
	}
	if tvSeason >= 0 && r.ASIDStart != nil && r.ASIDEnd != nil &&
		*r.ASIDStart == *r.ASIDEnd && r.SIDStart != nil &&
		r.SIDEnd != nil && *r.SIDStart == *r.SIDEnd {
		id, _, ok := matchlab.ParseASID(*r.ASIDStart)
		_, season, ep, has, valid := matchlab.ParseSID(*r.SIDStart)
		if ok && has && valid && season == tvSeason {
			key := fmt.Sprintf("%s:tv:%d", id, season)
			g := groups[key]
			if g == nil {
				g = &matchLabAnimeGroup{id: id, season: season, kind: anidb.TorrentSeasonTypeTV, episode: map[int]struct{}{}}
				groups[key] = g
			}
			g.episode[ep] = struct{}{}
		}
	}
}

func addMatchLabSegments(groups map[string]*matchLabAnimeGroup, seasons map[string]int, segments []matchlab.EpisodeSegment) {
	for _, segment := range segments {
		if !segment.Valid() {
			continue
		}
		animeSeason, found := seasons[segment.AniDBID]
		if !found || animeSeason < 0 {
			animeSeason = -1
		}
		key := fmt.Sprintf("%s:ani:%d", segment.AniDBID, animeSeason)
		a := groups[key]
		if a == nil {
			a = &matchLabAnimeGroup{
				id: segment.AniDBID, season: animeSeason,
				kind: anidb.TorrentSeasonTypeAnime, episode: map[int]struct{}{},
			}
			groups[key] = a
		}
		for ep := segment.StartAnimeEpisode; ep <= segment.EndAnimeEpisode; ep++ {
			a.episode[ep] = struct{}{}
		}
		tvKey := fmt.Sprintf("%s:tv:%d", segment.AniDBID, segment.Season)
		tv := groups[tvKey]
		if tv == nil {
			tv = &matchLabAnimeGroup{
				id: segment.AniDBID, season: segment.Season,
				kind: anidb.TorrentSeasonTypeTV, episode: map[int]struct{}{},
			}
			groups[tvKey] = tv
		}
		for ep := segment.StartEpisode; ep <= segment.EndEpisode; ep++ {
			tv.episode[ep] = struct{}{}
		}
		if segment.StartAbsolute != nil && segment.EndAbsolute != nil {
			key := fmt.Sprintf("%s:abs:%d", segment.AniDBID, segment.Season)
			absolute := groups[key]
			if absolute == nil {
				absolute = &matchLabAnimeGroup{
					id: segment.AniDBID, season: segment.Season,
					kind:    anidb.TorrentSeasonTypeAbsolute,
					episode: map[int]struct{}{},
				}
				groups[key] = absolute
			}
			for ep := *segment.StartAbsolute; ep <= *segment.EndAbsolute; ep++ {
				absolute.episode[ep] = struct{}{}
			}
		}
	}
}

func sortedInts(set map[int]struct{}) []int {
	out := make([]int, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// representativeSeriesID returns the file whose verified last episode has
// the highest season, then episode. Only comparisons of real SID coordinates
// are allowed; filename numbering is never re-parsed locally.
func representativeSeriesID(files []matchLabFileResult) string {
	selected := ""
	bestSeason, bestEpisode := -1, -1
	for _, file := range files {
		for _, endpoint := range []*string{file.Match.SIDEnd, file.Match.SIDStart} {
			if endpoint == nil {
				continue
			}
			id, season, episode, has, ok := matchlab.ParseSID(*endpoint)
			if !has || !ok {
				continue
			}
			if season > bestSeason || (season == bestSeason && episode > bestEpisode) ||
				(season == bestSeason && episode == bestEpisode && (selected == "" || id < selected)) {
				selected, bestSeason, bestEpisode = id, season, episode
			}
		}
	}
	return selected
}

// buildMatchLabMapping deliberately trusts the external results, never a
// second local IMDb or AniDB title-search routine.
func buildMatchLabMapping(torrent torrent_info.TorrentInfo, whole matchlab.Response, wholeSegments []matchlab.EpisodeSegment, files []matchLabFileResult, anidbSeasons map[string]int) matchLabMapping {
	result := matchLabMapping{Files: files, Category: torrent.Category}
	ids := map[string]struct{}{}
	coords := map[string][]matchLabSeasonEpisode{}
	animeGroups := map[string]*matchLabAnimeGroup{}

	fileIDs := map[string]struct{}{}
	for _, file := range files {
		mergeMatchLabIdentity(fileIDs, coords, file.Match)
		addMatchLabRange(coords, file.Match)
		addMatchLabAnime(animeGroups, anidbSeasons, file.Match)
		addMatchLabSegments(animeGroups, anidbSeasons, file.Segments)
	}
	fileAnime := false
	for _, file := range files {
		if file.Match.HasAnime() || len(file.Segments) > 0 {
			fileAnime = true
			break
		}
	}
	if len(fileIDs) > 1 && !fileAnime {
		// Ordinary season packs should identify one main TV series. The
		// highest verified season/episode wins when candidates disagree.
		// Mixed anime packs are exempt because each file may be different.
		if representative := representativeSeriesID(files); representative != "" {
			fileIDs = map[string]struct{}{representative: {}}
			for id := range coords {
				if id != representative {
					delete(coords, id)
				}
			}
		}
	}
	for id := range fileIDs {
		ids[id] = struct{}{}
	}
	wholeIDs := map[string]struct{}{}
	wholeCoords := map[string][]matchLabSeasonEpisode{}
	mergeMatchLabIdentity(wholeIDs, wholeCoords, whole)
	if len(fileIDs) == 0 {
		for id := range wholeIDs {
			ids[id] = struct{}{}
		}
		for id, values := range wholeCoords {
			coords[id] = append(coords[id], values...)
		}
		addMatchLabRange(coords, whole)
	} else if len(fileIDs) == 1 {
		// A torrent-title match is evidence only when it identifies the same
		// work as the independently matched file(s).
		for id := range wholeIDs {
			if _, ok := fileIDs[id]; ok && len(coords[id]) == 0 {
				// Trust root coordinates only when the file names could not
				// establish the episode span independently.
				coords[id] = append(coords[id], wholeCoords[id]...)
				addMatchLabRange(coords, whole)
			}
		}
	}
	// Ignore an unrelated root-level AniDB identity for mixed collections:
	// file-specific IDs have stronger evidence than a generic parent name.
	rootASIDMatchesFiles := len(files) == 0
	fileAnimeFound := false
	rootAnimeIDs := map[string]struct{}{}
	for _, v := range []*string{whole.ASIDStart, whole.ASIDEnd} {
		if v == nil {
			continue
		}
		if id, _, ok := matchlab.ParseASID(*v); ok {
			rootAnimeIDs[id] = struct{}{}
		}
	}
	for _, file := range files {
		for _, v := range []*string{file.Match.ASIDStart, file.Match.ASIDEnd} {
			if v == nil {
				continue
			}
			id, _, ok := matchlab.ParseASID(*v)
			if !ok {
				continue
			}
			fileAnimeFound = true
			if _, matched := rootAnimeIDs[id]; matched {
				rootASIDMatchesFiles = true
			}
		}
	}
	if !fileAnimeFound {
		rootASIDMatchesFiles = true
	}
	if rootASIDMatchesFiles {
		addMatchLabAnime(animeGroups, anidbSeasons, whole)
		addMatchLabSegments(animeGroups, anidbSeasons, wholeSegments)
	}

	for id := range ids {
		result.IMDbIDs = append(result.IMDbIDs, id)
	}
	slices.Sort(result.IMDbIDs)

	if len(ids) == 1 {
		var id string
		for id = range ids {
		}
		seasonSet := map[int]struct{}{}
		episodeSet := map[int]struct{}{}
		for _, ep := range coords[id] {
			if ep.season < 0 || ep.season > 9999 || ep.episode < 0 || ep.episode > 99999 {
				continue
			}
			seasonSet[ep.season] = struct{}{}
			episodeSet[ep.episode] = struct{}{}
		}
		result.Seasons = sortedInts(seasonSet)
		if len(result.Seasons) > 1 {
			// Validated endpoints delimit the requested season interval.
			// The torrent parser can also expose intermediate season numbers.
			first, last := result.Seasons[0], result.Seasons[len(result.Seasons)-1]
			if last-first < 2048 {
				result.Seasons = result.Seasons[:0]
				for season := first; season <= last; season++ {
					result.Seasons = append(result.Seasons, season)
				}
			}
		}
		// MatchLab's verified SID coordinates are authoritative. Never
		// reintroduce stale seasons from the earlier /parse result: a
		// corrected IMDb match must replace torrent_info.seasons and
		// torrent_info.episodes, not merge with the old values.
		if len(result.Seasons) > 1 {
			// A multi-season torrent must not expose an unscoped episode list.
			result.Episodes = []int{}
		} else if len(result.Seasons) == 1 {
			eps := sortedInts(episodeSet)
			if len(eps) > 0 && eps[len(eps)-1]-eps[0] < 2048 {
				for ep := eps[0]; ep <= eps[len(eps)-1]; ep++ {
					result.Episodes = append(result.Episodes, ep)
				}
			}
		}
		if len(result.Seasons) > 0 {
			result.Category = torrent_info.TorrentInfoCategorySeries
		} else if result.Category == "" {
			result.Category = torrent_info.TorrentInfoCategoryMovie
		}
	} else if len(ids) > 1 {
		// Multiple independently identified works: there is no meaningful
		// global single-season / episode numbering for this torrent.
		result.Seasons = []int{}
		result.Episodes = []int{}
	}

	if len(result.Seasons) == 0 && len(ids) < 2 {
		// Parsed /parse data is the only other source of season information.
		result.Seasons = append([]int(nil), torrent.Seasons...)
		result.Episodes = append([]int(nil), torrent.Episodes...)
	}
	if len(result.Seasons) > 1 {
		result.Episodes = []int{}
	}
	if len(result.Seasons) == 1 && len(result.Episodes) > 2048 {
		result.Episodes = []int{}
	}
	if torrent.Category != torrent_info.TorrentInfoCategoryXXX &&
		(len(result.Seasons) > 0 || len(result.Episodes) > 0) {
		result.Category = torrent_info.TorrentInfoCategorySeries
	}
	if len(animeGroups) > 0 && result.Category == "" {
		// Mixed AniDB seasons should still be categorized as TV when
		// verified coordinates show episodic (non-movie) content.
		for _, group := range animeGroups {
			for ep := range group.episode {
				if ep > 0 {
					result.Category = torrent_info.TorrentInfoCategorySeries
					break
				}
			}
			if result.Category != "" {
				break
			}
		}
	}

	for _, group := range animeGroups {
		eps := sortedInts(group.episode)
		if len(eps) == 0 {
			continue
		}
		entry := anidb.AniDBTorrent{
			TId: group.id, Hash: torrent.Hash, SeasonType: group.kind,
			Season: group.season, EpisodeStart: eps[0],
			EpisodeEnd: eps[len(eps)-1], Episodes: db.CommaSeperatedInt(eps),
		}
		result.AniDB = append(result.AniDB, entry)
	}
	slices.SortFunc(result.AniDB, func(a, b anidb.AniDBTorrent) int {
		return strings.Compare(fmt.Sprintf("%s:%s:%d", a.TId, a.SeasonType, a.Season),
			fmt.Sprintf("%s:%s:%d", b.TId, b.SeasonType, b.Season))
	})
	return result
}

func matchlabFilePath(torrentTitle string, file torrent_stream.File) string {
	// A complete path lets MatchLab inherit title from a parent folder while
	// letting the leaf's episode coordinate win over a pack-level season.
	return strings.TrimRight(torrentTitle, "/") + "/" + strings.TrimLeft(strings.ReplaceAll(file.Path, "\\", "/"), "/")
}

func matchlabBestFileResult(ctx context.Context, torrentTitle string, file torrent_stream.File, whole matchlab.Response) (matchlab.Response, string, error) {
	combinedTitle := matchlabFilePath(torrentTitle, file)
	combined, err := matchlab.Default.Match(ctx, combinedTitle)
	if err != nil {
		return combined, combinedTitle, err
	}
	// Nested folders named "Season 2" or "TV" can incorrectly supply
	// generic context. Preserve the raw torrent title while retaining the
	// file's exact leaf episode when no identity was returned.
	if combined.SIDStart == nil && combined.ASIDStart == nil {
		leaf := path.Base(strings.ReplaceAll(file.Path, "\\", "/"))
		fallbackTitle := strings.TrimRight(torrentTitle, "/") + "/" + leaf
		if leaf != "." && leaf != "" && fallbackTitle != combinedTitle {
			fallback, err := matchlab.Default.Match(ctx, fallbackTitle)
			if err != nil {
				return combined, combinedTitle, err
			}
			if fallback.SIDStart != nil || fallback.ASIDStart != nil {
				combined, combinedTitle = fallback, fallbackTitle
			}
		}
	}
	// For anime, a parent folder may refer to a different series in a mixed
	// release. An independently verified AniDB identity from the leaf wins.
	if combined.HasAnime() || whole.HasAnime() {
		leafTitle := strings.TrimLeft(strings.ReplaceAll(file.Path, "\\", "/"), "/")
		if strings.TrimSpace(leafTitle) != "" {
			leaf, err := matchlab.Default.Match(ctx, leafTitle)
			if err != nil {
				return combined, combinedTitle, err
			}
			if leaf.HasAnime() {
				return leaf, leafTitle, nil
			}
		}
	}
	return combined, combinedTitle, nil
}

func matchlabNeedsDetail(r matchlab.Response) bool {
	if !r.HasAnime() {
		return false
	}
	if r.ASIDStart != nil && r.ASIDEnd != nil && *r.ASIDStart != *r.ASIDEnd {
		return true
	}
	if r.SIDStart != nil && r.SIDEnd != nil && *r.SIDStart != *r.SIDEnd {
		return true
	}
	return false
}

func matchlabVerifiedSegments(ctx context.Context, title string, r matchlab.Response) ([]matchlab.EpisodeSegment, error) {
	if !matchlabNeedsDetail(r) {
		return nil, nil
	}
	d, err := matchlab.Default.Debug(ctx, title)
	if err != nil {
		return nil, err
	}
	if d.Selected == nil || d.Selected.EpisodeRange == nil {
		return nil, nil
	}
	if r.SIDStart != nil && d.Selected.IMDbID != "" {
		id, _, _, _, valid := matchlab.ParseSID(*r.SIDStart)
		if valid && id != d.Selected.IMDbID {
			return nil, errors.New("MatchLab debug identity changed between requests")
		}
	}
	return d.Selected.EpisodeRange.Segments, nil
}

// Store changes in one transaction. A stale old IMDb/AniDB mapping must not
// survive a new authoritative MatchLab pass, and a failed HTTP request must
// never create a false no-match.
func saveMatchLabMapping(t torrent_info.TorrentInfo, m matchLabMapping, fileRevision int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	seasons := intsCSV(m.Seasons)
	episodes := intsCSV(m.Episodes)
	res, err := tx.Exec(
		"UPDATE torrent_info SET seasons = ?, episodes = ?, category = ?, matchlab_mapped_at = "+db.CurrentTimestamp+", matchlab_retry_at = NULL, updated_at = "+db.CurrentTimestamp+
			" WHERE hash = ? AND t_title = ? AND parser_input = ? AND matchlab_mapped_at IS NULL"+
			" AND parser_version = ? AND matchlab_file_revision = ?",
		seasons, episodes, m.Category, t.Hash, t.TorrentTitle, t.TorrentTitle, matchlab.ParserVersion, fileRevision,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	} // changed during an in-flight HTTP request

	if _, err = tx.Exec("DELETE FROM imdb_torrent WHERE hash = ?", t.Hash); err != nil {
		return err
	}
	for _, id := range m.IMDbIDs {
		if _, err = tx.Exec("INSERT INTO imdb_torrent (hash, tid) VALUES (?, ?)", t.Hash, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("DELETE FROM anidb_torrent WHERE hash = ?", t.Hash); err != nil {
		return err
	}
	for _, a := range m.AniDB {
		if _, err = tx.Exec("INSERT INTO anidb_torrent (tid, hash, s_type, s, ep_start, ep_end, eps) VALUES (?, ?, ?, ?, ?, ?, ?)",
			a.TId, t.Hash, a.SeasonType, a.Season, a.EpisodeStart, a.EpisodeEnd, a.Episodes); err != nil {
			return err
		}
	}

	for _, f := range m.Files {
		sid, asid := f.Match.SingleSID(), f.Match.SingleASID()
		if sid == "" {
			sid = "*" // StremThru's explicit unmatched file sentinel
		}
		// MatchLab is authoritative in this mode. Retaining stale local
		// SID/ASID values would leak obsolete identities into search results.
		if _, err = tx.Exec(
			"UPDATE torrent_stream SET sid = ?, asid = ?, uat = "+db.CurrentTimestamp+
				" WHERE h = ? AND p = ?", sid, asid, t.Hash, f.Path,
		); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	torrent_stream.InvalidateFilesByHash(t.Hash)
	return nil
}

func intsCSV(values []int) string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = strconv.Itoa(v)
	}
	return strings.Join(out, ",")
}

func doMatchLabTorrent(ctx context.Context, t torrent_info.TorrentInfo, files torrent_stream.Files, fileRevision int64) error {
	whole, err := matchlab.Default.Match(ctx, t.TorrentTitle)
	if err != nil {
		return fmt.Errorf("match torrent name: %w", err)
	}
	wholeSegments, err := matchlabVerifiedSegments(ctx, t.TorrentTitle, whole)
	if err != nil {
		return fmt.Errorf("resolve torrent anime segments: %w", err)
	}
	mappedFiles := make([]matchLabFileResult, 0, len(files))
	animeIDs := map[string]struct{}{}
	for _, f := range files {
		f.Normalize()
		if !f.IsVideo() || f.Path == "" {
			continue
		}
		r, resolvedTitle, err := matchlabBestFileResult(ctx, t.TorrentTitle, f, whole)
		if err != nil {
			return fmt.Errorf("match file %q: %w", f.Path, err)
		}
		segments, err := matchlabVerifiedSegments(ctx, resolvedTitle, r)
		if err != nil {
			return fmt.Errorf("resolve anime file %q: %w", f.Path, err)
		}
		mappedFiles = append(mappedFiles, matchLabFileResult{Path: f.Path, Match: r, Segments: segments})
		for _, id := range []*string{r.ASIDStart, r.ASIDEnd} {
			if id == nil {
				continue
			}
			aid, _, ok := matchlab.ParseASID(*id)
			if ok {
				animeIDs[aid] = struct{}{}
			}
		}
	}
	for _, segment := range wholeSegments {
		animeIDs[segment.AniDBID] = struct{}{}
	}
	for _, file := range mappedFiles {
		for _, segment := range file.Segments {
			animeIDs[segment.AniDBID] = struct{}{}
		}
	}
	for _, id := range []*string{whole.ASIDStart, whole.ASIDEnd} {
		if id == nil {
			continue
		}
		aid, _, ok := matchlab.ParseASID(*id)
		if ok {
			animeIDs[aid] = struct{}{}
		}
	}
	seasonMap := map[string]int{}
	if len(animeIDs) > 0 {
		ids := make([]string, 0, len(animeIDs))
		for id := range animeIDs {
			ids = append(ids, id)
		}
		titles, err := anidb.GetTitlesByIds(ids)
		if err == nil {
			for _, id := range ids {
				seasonMap[id] = titles.GetSeason(id)
			}
		}
		// Absence of local anime title metadata must not stop MatchLab from
		// supplying ASID. Preserve unknown season=-1 (query sentinel).
	}
	mapping := buildMatchLabMapping(t, whole, wholeSegments, mappedFiles, seasonMap)
	return saveMatchLabMapping(t, mapping, fileRevision)
}

// Very large torrent packs need more time than single-episode requests.
// Keep a hard cap so an unhealthy server cannot block a worker forever.
func matchLabTorrentTimeout(fileCount int) time.Duration {
	extra := time.Duration(max(0, fileCount-100)) * 250 * time.Millisecond
	return min(20*time.Minute, 2*time.Minute+extra)
}

// The existing map-imdb-torrent worker slot becomes the sole mapping
// executor. It does not require local IMDb or AniDB datasets to have synced.
func InitMapMatchLabTorrentWorker(conf *WorkerConfig) *Worker {
	conf.Executor = func(w *Worker) error {
		if err := matchlab.Default.Health(context.Background()); err != nil {
			return err
		}
		for {
			pending, err := torrent_info.GetMatchLabPendingHashes(200)
			if err != nil {
				return err
			}
			if len(pending) == 0 {
				return nil
			}
			hashes := make([]string, len(pending))
			for i, entry := range pending {
				hashes[i] = entry.Hash
			}
			tInfos, err := torrent_info.GetByHashes(hashes)
			if err != nil {
				return err
			}
			allFiles, err := torrent_stream.GetFilesByHashes(hashes)
			if err != nil {
				return err
			}
			// Parallelism is deliberately capped below the HTTP client's own
			// in-flight request limit. Work stays bounded even for large packs.
			sem := make(chan struct{}, 4)
			var wg sync.WaitGroup
			var mutex sync.Mutex
			var errs []error
			for _, entry := range pending {
				hash := entry.Hash
				t, ok := tInfos[hash]
				if !ok || !t.IsParsed() {
					continue
				}
				wg.Add(1)
				sem <- struct{}{}
				go func(hash string, t torrent_info.TorrentInfo, files torrent_stream.Files, revision int64) {
					defer wg.Done()
					defer func() { <-sem }()
					// A deadline bounds all requests of a torrent, not each
					// API call independently.
					ctx, cancel := context.WithTimeout(context.Background(), matchLabTorrentTimeout(len(files)))
					defer cancel()
					if err := doMatchLabTorrent(ctx, t, files, revision); err != nil {
						// A poison torrent or one enormous pack must not
						// permanently starve the other pending hashes.
						retryErr := torrent_info.RetryMatchLabHash(hash, revision, 5*time.Minute)
						mutex.Lock()
						errs = append(errs, fmt.Errorf("%s: %w", hash, err))
						if retryErr != nil {
							errs = append(errs, fmt.Errorf("%s: schedule retry: %w", hash, retryErr))
						}
						mutex.Unlock()
					}
				}(hash, t, allFiles[hash], entry.FileRevision)
			}
			wg.Wait()
			if len(errs) > 0 {
				w.Log.Warn("MatchLab deferred failed torrent mappings",
					"failures", len(errs), "total", len(hashes), "error", errors.Join(errs...))
				// Continue past isolated bad torrents, but halt a systemic
				// outage before bombarding the external service.
				if len(errs)*3 >= len(hashes) {
					return errors.Join(errs...)
				}
			}
			w.Log.Info("processed torrents through MatchLab",
				"count", len(hashes), "failures", len(errs))
			if len(hashes) < 200 {
				return nil
			}
		}
	}
	return NewWorker(conf)
}
