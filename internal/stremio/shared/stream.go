package stremio_shared

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

func MatchFileByIdx(files []store.MagnetFile, idx int, storeCode store.StoreCode) *store.MagnetFile {
	if idx == -1 || storeCode != store.StoreCodeRealDebrid {
		return nil
	}
	for i := range files {
		f := &files[i]
		if f.Idx == idx {
			return f
		}
	}
	return nil
}

func MatchFileByLargestSize(files []store.MagnetFile) (file *store.MagnetFile) {
	for i := range files {
		f := &files[i]
		if file == nil || file.Size < f.Size {
			file = f
		}
	}
	return file
}

func MatchFileByName(files []store.MagnetFile, name string) *store.MagnetFile {
	if name == "" {
		return nil
	}
	for i := range files {
		f := &files[i]
		if f.Name == name {
			return f
		}
	}
	return nil
}

func MatchFileByPattern(files []store.MagnetFile, pattern *regexp.Regexp) *store.MagnetFile {
	if pattern == nil {
		return nil
	}
	for i := range files {
		f := &files[i]
		if pattern.MatchString(f.Name) {
			return f
		}
	}
	return nil
}

var parse_season_episode = ptt.GetPartialParser([]string{"releaseTypes", "seasons", "episodes"})

var digits_regex = regexp.MustCompile(`\b(\d+)\b`)

type seasonEpisodeData struct {
	Season      int
	Episode     int
	ReleaseType string
}

func ParseSeasonEpisodeFromName(title string, extractDigitsAsEpisodeAgressively bool) seasonEpisodeData {
	data := seasonEpisodeData{-1, -1, ""}
	r := parse_season_episode(title)
	if err := r.Error(); err != nil {
		log.Error("failed to parse season episode", "title", title, "error", err)
		return data
	}
	if len(r.Seasons) > 0 {
		data.Season = r.Seasons[0]
	}
	if len(r.Episodes) > 0 {
		data.Episode = r.Episodes[0]
	}
	if extractDigitsAsEpisodeAgressively && data.Season == -1 && data.Episode == -1 {
		matches := digits_regex.FindAllString(title, 2)
		if len(matches) == 1 {
			data.Episode, _ = strconv.Atoi(matches[0])
		}
	}
	if len(r.ReleaseTypes) > 0 {
		data.ReleaseType = r.ReleaseTypes[0]
	}
	return data
}

func matchFileByIMDBStremId(files []store.MagnetFile, sid string) *store.MagnetFile {
	parts := strings.SplitN(sid, ":", 3)
	if len(parts) != 3 {
		return nil
	}
	expectedSeason, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Warn("failed to parse season from imdb strem id", "error", err, "sid", sid)
		return nil
	}
	expectedEpisode := -1
	if len(parts) == 3 {
		expectedEpisode, err = strconv.Atoi(parts[2])
		if err != nil {
			log.Warn("failed to parse episode from imdb strem id", "error", err, "sid", sid)
			return nil
		}
	}
	for i := range files {
		f := &files[i]
		if d := ParseSeasonEpisodeFromName(f.Name, false); d.Season == expectedSeason && d.Episode == expectedEpisode {
			return f
		}
	}
	return nil
}

func MatchFileByStremId(files []store.MagnetFile, sid string, magnetHash string, storeCode store.StoreCode) *store.MagnetFile {
	if f, err := torrent_stream.GetFile(magnetHash, sid); err != nil {
		log.Error("failed to get file by strem id", "hash", magnetHash, "sid", sid, "error", err)
	} else if f != nil {
		if file := MatchFileByIdx(files, f.Idx, storeCode); file != nil {
			log.Debug("matched file by strem id - fileidx", "hash", magnetHash, "sid", sid, "filename", file.Name, "fileidx", file.Idx, "store.name", storeCode.Name())
			return file
		}
		if file := MatchFileByName(files, f.Name); file != nil {
			log.Debug("matched file by strem id - filename", "hash", magnetHash, "sid", sid, "filename", file.Name, "fileidx", file.Idx, "store.name", storeCode.Name())
			return file
		}
	}

	nsid, err := torrent_stream.NormalizeStreamId(sid)
	if err != nil {
		log.Error("failed to normalize strem id", "error", err, "sid", sid)
		return nil
	}

	if nsid.IsAnime {
		anidbId, season, episode := nsid.Id, nsid.Season, nsid.Episode

		tInfo, err := torrent_info.GetByHash(magnetHash)
		if err != nil {
			log.Error("failed to get torrent info by hash", "error", err, "hash", magnetHash)
			return nil
		}
		expectedEpisode := util.SafeParseInt(episode, -1)
		expectedSeason := util.SafeParseInt(season, -1)

		filesForSeason := []*store.MagnetFile{}
		dataByName := map[string]seasonEpisodeData{}

		minEpisode := 99999
		for i := range files {
			f := &files[i]
			d := ParseSeasonEpisodeFromName(f.Name, true)
			dataByName[f.Name] = d
			if d.ReleaseType == "" && (d.Episode != -1) && ((d.Season == -1 && expectedSeason == 1) || d.Season == expectedSeason) {
				filesForSeason = append(filesForSeason, f)
				if d.Episode < minEpisode {
					minEpisode = d.Episode
				}
			}
		}

		if len(filesForSeason) == 0 {
			tvdbMaps, err := anidb.GetTVDBEpisodeMaps(anidbId, false)
			if err != nil {
				log.Error("failed to get tvdb episode maps for anidb id", "error", err, "anidb_id", anidbId)
				return nil
			}

			absMap := tvdbMaps.GetAbsoluteOrderSeasonMap()
			if absMap != nil {
				expectedEpisode = expectedEpisode + absMap.Offset
				for i := range files {
					f := &files[i]
					d := dataByName[f.Name]
					if d.Episode == expectedEpisode {
						return f
					}
				}
			}
		} else {
			for _, f := range filesForSeason {
				d := dataByName[f.Name]
				if d.Episode == expectedEpisode || (len(tInfo.Episodes) == 0 && minEpisode > 1 && d.Episode-minEpisode+1 == expectedEpisode) {
					return f
				}
			}
		}

		return nil
	}

	return matchFileByIMDBStremId(files, sid)
}
