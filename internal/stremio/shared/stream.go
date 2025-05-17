package stremio_shared

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
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

var parse_season_episode = ptt.GetPartialParser([]string{"seasons", "episodes"})

func getSeasonEpisode(title string) (season, episode int) {
	season, episode = -1, -1
	r := parse_season_episode(title)
	if err := r.Error(); err != nil {
		log.Error("failed to parse season episode", "title", title, "error", err)
		return season, episode
	}
	if len(r.Seasons) > 0 {
		season = r.Seasons[0]
	}
	if len(r.Episodes) > 0 {
		episode = r.Episodes[0]
	}
	return season, episode
}

func MatchFileByStremId(files []store.MagnetFile, sid string, magnetHash string, storeCode store.StoreCode) *store.MagnetFile {
	if f, err := torrent_stream.GetFile(magnetHash, sid); err != nil {
		log.Error("failed to get file by strem id", "hash", magnetHash, "sid", sid, "error", err)
	} else if f != nil {
		if file := MatchFileByIdx(files, f.Idx, storeCode); file != nil {
			log.Debug("matched file by strem id - fileidx", "hash", magnetHash, "sid", sid, "filename", file.Name, "fileidx", file.Idx, "store", storeCode)
			return file
		}
		if file := MatchFileByName(files, f.Name); file != nil {
			log.Debug("matched file by strem id - filename", "hash", magnetHash, "sid", sid, "filename", file.Name, "fileidx", file.Idx, "store", storeCode)
			return file
		}
	}
	if parts := strings.SplitN(sid, ":", 3); len(parts) == 3 {
		expectedSeason, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Warn("failed to parse season from strem id", "sid", sid, "error", err)
			return nil
		}
		expectedEpisode, err := strconv.Atoi(parts[2])
		if err != nil {
			log.Warn("failed to parse episode from strem id", "sid", sid, "error", err)
			return nil
		}
		for i := range files {
			f := &files[i]
			if season, episode := getSeasonEpisode(f.Name); season == expectedSeason && episode == expectedEpisode {
				return f
			}
		}
	}
	return nil
}
