package endpoint

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torznab"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

var regexSeasonEpisode = regexp.MustCompile(`(?i)\s*S(\d+)(?:E(\d+))?\b`)

func sendZnabResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any, o string) {
	switch o {
	case "json":
		shared.SendJSON(w, r, statusCode, data)
	case "xml", "":
		shared.SendXML(w, r, statusCode, data)
	default:
		shared.SendXML(w, r, 200, znab.ErrorIncorrectParameter("invalid output format"))
	}
}

func handleTorznab(w http.ResponseWriter, r *http.Request) {
	reqQuery := r.URL.Query()

	t := reqQuery.Get("t")
	if t == "" {
		http.Redirect(w, r, r.URL.Path+"?t=caps", http.StatusTemporaryRedirect)
		return
	}

	o := strings.ToLower(reqQuery.Get("o"))
	if o != "" && o != "json" && o != "xml" {
		shared.SendXML(w, r, 200, znab.ErrorIncorrectParameter("invalid output format"))
		return
	}

	switch t {
	case "caps":
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendZnabResponse(w, r, 200, torznab.StremThruIndexer.Capabilities(), o)
	case "search", "tvsearch", "movie":
		if server.IsMaintenanceActive() {
			sendZnabResponse(w, r, http.StatusServiceUnavailable, znab.ErrorUnknownError("server is under maintenance"), o)
			return
		}

		query, err := torznab.ParseQuery(reqQuery)
		if err != nil {
			sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter(err.Error()), o)
			return
		}
		if query.Q != "" && query.Season == "" && query.Ep == "" {
			if m := regexSeasonEpisode.FindStringSubmatch(query.Q); m != nil {
				if season := util.SafeParseInt(m[1], -1); season != -1 {
					query.Season = util.IntToString(season)
				}
				if len(m) > 2 {
					if ep := util.SafeParseInt(m[2], -1); ep != -1 {
						query.Ep = util.IntToString(ep)
					}
				}
				query.Q = strings.TrimSpace(regexSeasonEpisode.ReplaceAllString(query.Q, ""))

				http.Redirect(w, r, r.URL.Path+"?"+query.Encode(), http.StatusTemporaryRedirect)
				return
			}
		}
		items, err := torznab.StremThruIndexer.Search(query)
		if err != nil {
			sendZnabResponse(w, r, 200, znab.ErrorUnknownError(err.Error()), o)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendZnabResponse(w, r, 200, torznab.Feed{
			Info:  torznab.StremThruIndexer.Info(),
			Items: items,
		}, o)
	default:
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter(t), o)
	}
}
func AddTorznabEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasTorrentInfo() {
		return
	}

	mux.HandleFunc("/v0/torznab/api", handleTorznab)
}
