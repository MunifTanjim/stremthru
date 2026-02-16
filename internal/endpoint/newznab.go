package endpoint

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/newznab"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
)

func isNewznabRequestAuthed(r *http.Request) bool {
	apiKey := r.URL.Query().Get("apikey")
	if apiKey == "" || strings.ContainsRune(apiKey, ':') {
		return false
	}

	auth, err := util.ParseBasicAuth(apiKey)
	if err != nil {
		return false
	}

	return config.UserAuth.GetPassword(auth.Username) == auth.Password
}

func handleNewznab(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")

	if t == "" {
		http.Redirect(w, r, r.URL.Path+"?t=caps", http.StatusTemporaryRedirect)
		return
	}

	o := strings.ToLower(r.URL.Query().Get("o"))
	if o != "" && o != "json" && o != "xml" {
		sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter("invalid output format"), "xml")
		return
	}

	switch t {
	case "caps":
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendZnabResponse(w, r, 200, newznab.StremThruIndexer.Capabilities(), o)
		return
	default:
		if !isNewznabRequestAuthed(r) {
			sendZnabResponse(w, r, 200, znab.ErrorIncorrectUserCreds, o)
			return
		}
	}

	baseURL := shared.ExtractRequestBaseURL(r).JoinPath("/v0/newznab")

	switch t {
	case "search", "tvsearch", "movie":
		query, err := newznab.ParseQuery(r.URL.Query())
		if err != nil {
			sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter(err.Error()), o)
			return
		}
		items, err := newznab.StremThruIndexer.Search(query)
		if err != nil {
			sendZnabResponse(w, r, 200, znab.ErrorUnknownError(err.Error()), o)
			return
		}
		nzbLinkQuery := url.Values{
			"apikey": {r.URL.Query().Get("apikey")},
			"t":      {"get"},
		}
		for i := range items {
			item := &items[i]

			link := baseURL.JoinPath("/api")
			nzbLinkQuery.Set("id", item.GUID)
			link.RawQuery = nzbLinkQuery.Encode()
			item.Link = link.String()
		}
		w.Header().Set("Cache-Control", "public, max-age=300")
		sendZnabResponse(w, r, 200, newznab.Feed{
			Info:  newznab.StremThruIndexer.Info(),
			Items: items,
		}, o)

	case "get":
		handleNewznabGet(w, r, o)

	default:
		w.Header().Set("Cache-Control", "public, max-age=7200")
		sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter(t), o)
	}
}

func handleNewznabGet(w http.ResponseWriter, r *http.Request, o string) {
	log := server.GetReqCtx(r).Log

	nzbId := r.URL.Query().Get("id")
	if nzbId == "" {
		sendZnabResponse(w, r, 200, znab.ErrorMissingParameter("id"), o)
		return
	}

	link, err := newznab.StremThruIndexer.UnwrapLink(nzbId)
	if err != nil {
		sendZnabResponse(w, r, 200, znab.ErrorIncorrectParameter("invalid id"), o)
		return
	}

	if config.NewzNZBLinkMode.Redirect(link.Hostname()) {
		http.Redirect(w, r, link.String(), http.StatusTemporaryRedirect)
		return
	}

	body, headers, err := newznab.StremThruIndexer.Download(link.String())
	if err != nil {
		sendZnabResponse(w, r, 200, znab.ErrorNoSuchItem, o)
		return
	}
	defer body.Close()

	for key, vals := range headers {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}

	w.WriteHeader(200)

	if _, err := io.Copy(w, body); err != nil {
		log.Warn("failed to send nzb body", "error", err)
	}
}

func handleNewznabGetNZB(w http.ResponseWriter, r *http.Request) {
	if !isNewznabRequestAuthed(r) {
		server.ErrorForbidden(r).Send(w, r)
		return
	}

	nzbId := r.PathValue("nzbId")
	if nzbId == "" {
		server.ErrorNotFound(r).Send(w, r)
		return
	}

	hash := nzb_info.HashNZBFileLink(config.BaseURL.JoinPath("/v0/newznab/getnzb", nzbId).String())
	file := nzb_info.GetCachedNZBFile(hash)
	if file == nil {
		server.ErrorNotFound(r).Send(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/x-nzb")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(file.Blob)))
	w.WriteHeader(http.StatusOK)
	w.Write(file.Blob)
}

func AddNewznabEndpoints(mux *http.ServeMux) {
	if !config.Feature.HasVault() {
		return
	}

	mux.HandleFunc("/v0/newznab/api", handleNewznab)
	mux.HandleFunc("/v0/newznab/getnzb/{nzbId}", handleNewznabGetNZB)
}
