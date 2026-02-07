package znabsearch

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/logger"
	nznc "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tznc "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

type QueryMeta struct {
	Titles     []string
	Year       int
	Season, Ep int
}

func GetQueryMeta(log *logger.Logger, sid string) (*QueryMeta, *torrent_stream.NormalizedStremId, error) {
	nsid, err := torrent_stream.NormalizeStreamId(sid)
	if err != nil {
		return nil, nil, err
	}

	meta := &QueryMeta{
		Titles: []string{},
	}

	if nsid.IsAnime {
		if aniEp := util.SafeParseInt(nsid.Episode, -1); aniEp != -1 {
			tvdbMaps, err := anidb.GetTVDBEpisodeMaps(nsid.Id, false)
			if err != nil {
				log.Error("failed to get AniDB-TVDB episode maps", "error", err, "anidb_id", nsid.Id)
				return nil, nsid, err
			}
			if epMap := tvdbMaps.GetByAnidbEpisode(aniEp); epMap != nil {
				ep := epMap.GetTMDBEpisode(aniEp)
				titles, err := anidb.GetTitlesByIds([]string{nsid.Id})
				if err != nil {
					log.Error("failed to get AniDB titles", "error", err, "anidb_id", nsid.Id)
					return nil, nsid, err
				}
				if len(titles) == 0 {
					log.Warn("AniDB title not found", "anidb_id", nsid.Id)
					return meta, nsid, nil
				}
				meta.Titles = make([]string, 0, len(titles))
				meta.Season = epMap.TVDBSeason
				meta.Ep = ep
				seenTitle := util.NewSet[string]()
				for i := range titles {
					title := &titles[i]
					if seenTitle.Has(title.Value) {
						continue
					}
					seenTitle.Add(title.Value)
					meta.Titles = append(meta.Titles, title.Value)
					if meta.Year == 0 && title.Year != "" {
						meta.Year = util.SafeParseInt(title.Year, 0)
					}
				}
			}
		}
	} else {
		it, err := imdb_title.Get(nsid.Id)
		if err != nil {
			log.Error("failed to get IMDB title", "error", err, "imdb_id", nsid.Id)
			return nil, nsid, err
		}
		if it == nil {
			log.Warn("IMDB title not found", "imdb_id", nsid.Id)
			return meta, nsid, nil
		}
		meta.Titles = append(meta.Titles, it.Title)
		if it.OrigTitle != "" && it.OrigTitle != it.Title {
			meta.Titles = append(meta.Titles, it.OrigTitle)
		}
		if it.Year > 0 {
			meta.Year = it.Year
		}
		if nsid.IsSeries() {
			meta.Season = util.SafeParseInt(nsid.Season, 0)
			meta.Ep = util.SafeParseInt(nsid.Episode, 0)
		}
	}
	return meta, nsid, nil
}

type IndexerQuery struct {
	Header  http.Header
	Query   url.Values
	IsExact bool
}

type QueryBuilderConfig struct {
	Meta *QueryMeta
	NSId *torrent_stream.NormalizedStremId
}

func BuildQueriesForTorznab(client tznc.Indexer, conf QueryBuilderConfig) (map[string][]IndexerQuery, error) {
	nsid, meta := conf.NSId, conf.Meta

	queriesBySid := map[string][]IndexerQuery{}

	query, err := client.NewSearchQuery(func(caps znab.Caps) znab.Function {
		if nsid.IsSeries() && caps.SupportsFunction(znab.FunctionSearchTV) {
			return znab.FunctionSearchTV
		}
		if caps.SupportsFunction(znab.FunctionSearchMovie) {
			return znab.FunctionSearchMovie
		}
		return znab.FunctionSearch
	})
	if err != nil {
		return nil, err
	}

	query.SetLimit(-1)
	if !nsid.IsAnime && query.IsSupported(znab.SearchParamIMDBId) {
		query.Set(znab.SearchParamIMDBId, nsid.Id)
		sid := nsid.ToClean()
		isExact := !nsid.IsSeries()

		if nsid.IsSeries() {
			if query.IsSupported(znab.SearchParamSeason) && nsid.Season != "" {
				query.Set(znab.SearchParamSeason, nsid.Season)
				if query.IsSupported(znab.SearchParamEp) && nsid.Episode != "" {
					query.Set(znab.SearchParamEp, nsid.Episode)
					isExact = true
					sid = nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
				} else {
					sid = nsid.ToClean() + ":" + nsid.Season
				}
			}
		}

		queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
			Query:   query.Clone().Values(),
			IsExact: isExact,
		})
	} else {
		query.SetT(znab.FunctionSearch)
		supportsYear := query.IsSupported(znab.SearchParamYear)
		if supportsYear && meta.Year != 0 {
			query.Set(znab.SearchParamYear, strconv.Itoa(meta.Year))
		}

		for _, title := range meta.Titles {
			var q strings.Builder
			q.WriteString(title)

			if nsid.IsSeries() {
				sid := nsid.ToClean()
				queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
					Query: query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
				})

				if meta.Season > 0 {
					q.WriteString(" S")
					q.WriteString(util.ZeroPadInt(meta.Season, 2))
					sid := nsid.ToClean() + ":" + nsid.Season
					queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
						Query: query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
					})

					if meta.Ep > 0 {
						q.WriteString("E")
						q.WriteString(util.ZeroPadInt(meta.Ep, 2))
						sid := nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
						queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
							Query: query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
						})
					}
				}
			} else if meta.Year > 0 {
				if !supportsYear {
					q.WriteString(" ")
					q.WriteString(strconv.Itoa(meta.Year))
				}
				sid := nsid.ToClean()
				queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
					Query: query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
				})
			}
		}
	}

	return queriesBySid, nil
}

func BuildQueriesForNewznab(client nznc.Indexer, conf QueryBuilderConfig) (map[string][]IndexerQuery, error) {
	nsid, meta := conf.NSId, conf.Meta

	queriesBySid := map[string][]IndexerQuery{}

	query, err := client.NewSearchQuery(func(caps *znab.Caps) znab.Function {
		if nsid.IsSeries() && caps.SupportsFunction(znab.FunctionSearchTV) {
			return znab.FunctionSearchTV
		}
		if caps.SupportsFunction(znab.FunctionSearchMovie) {
			return znab.FunctionSearchMovie
		}
		return znab.FunctionSearch
	})
	if err != nil {
		return nil, err
	}

	query.SetLimit(-1)
	if !nsid.IsAnime && query.IsSupported(znab.SearchParamIMDBId) {
		query.Set(znab.SearchParamIMDBId, strings.TrimPrefix(nsid.Id, "tt"))
		sid := nsid.ToClean()
		isExact := !nsid.IsSeries()

		if nsid.IsSeries() {
			if query.IsSupported(znab.SearchParamSeason) && nsid.Season != "" {
				query.Set(znab.SearchParamSeason, nsid.Season)
				if query.IsSupported(znab.SearchParamEp) && nsid.Episode != "" {
					query.Set(znab.SearchParamEp, nsid.Episode)
					isExact = true
					sid = nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
				} else {
					sid = nsid.ToClean() + ":" + nsid.Season
				}
			}
		}

		queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
			Header:  query.GetHeader(),
			Query:   query.Clone().Values(),
			IsExact: isExact,
		})
	} else {
		query.SetT(znab.FunctionSearch)
		supportsYear := query.IsSupported(znab.SearchParamYear)
		if supportsYear && meta.Year != 0 {
			query.Set(znab.SearchParamYear, strconv.Itoa(meta.Year))
		}

		for _, title := range meta.Titles {
			var q strings.Builder
			q.WriteString(title)

			if nsid.IsSeries() {
				sid := nsid.ToClean()
				queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
					Header: query.GetHeader(),
					Query:  query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
				})

				if meta.Season > 0 {
					q.WriteString(" S")
					q.WriteString(util.ZeroPadInt(meta.Season, 2))
					sid := nsid.ToClean() + ":" + nsid.Season
					queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
						Header: query.GetHeader(),
						Query:  query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
					})

					if meta.Ep > 0 {
						q.WriteString("E")
						q.WriteString(util.ZeroPadInt(meta.Ep, 2))
						sid := nsid.ToClean() + ":" + nsid.Season + ":" + nsid.Episode
						queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
							Header: query.GetHeader(),
							Query:  query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
						})
					}
				}
			} else if meta.Year > 0 {
				if !supportsYear {
					q.WriteString(" ")
					q.WriteString(strconv.Itoa(meta.Year))
				}
				sid := nsid.ToClean()
				queriesBySid[sid] = append(queriesBySid[sid], IndexerQuery{
					Header: query.GetHeader(),
					Query:  query.Clone().Set(znab.SearchParamQ, q.String()).Values(),
				})
			}
		}
	}

	return queriesBySid, nil
}
