package stremio_meta

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/meta"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/stremio"
)

func handleMeta(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	contentType, err := getContentType(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idType := meta.IdTypeUnknown
	switch stremio.ContentType(contentType) {
	case stremio.ContentTypeMovie:
		idType = meta.IdTypeMovie
	case stremio.ContentTypeSeries:
		idType = meta.IdTypeShow
	}

	idStr := getId(r)
	idProvider, id := meta.ParseId(idStr)

	var idMap *meta.IdMap

	switch idProvider {
	case meta.IdProviderIMDB, meta.IdProviderTVDB:
		idMap, err = meta.GetIdMap(idType, idStr)
		if err != nil {
			SendError(w, r, err)
			return
		}
		if idMap == nil {
			break
		}
	default:
		shared.ErrorBadRequest(r, "unsupported id").Send(w, r)
		return
	}

	client := tvdb.GetAPIClient()

	if idMap == nil {
		switch idProvider {
		case meta.IdProviderTVDB:
			li := tvdb.TVDBItem{
				Id: util.SafeParseInt(id, -1),
			}
			if li.Id == -1 {
				break
			}
			err := li.Fetch(client)
			if err != nil {
				SendError(w, r, err)
				return
			}
			idMap = li.IdMap
		}
	}

	m := stremio.Meta{
		Id:            idStr,
		Type:          stremio.ContentType(contentType),
		BehaviorHints: &stremio.MetaBehaviorHints{},
	}

	switch idType {
	case meta.IdTypeMovie:
		if idMap == nil {
			break
		}

		switch ud.StreamId.Movie {
		case "", "imdb":
			m.BehaviorHints.DefaultVideoId = idMap.IMDB
		case "tvdb":
			if idMap.TVDB != "" {
				m.BehaviorHints.DefaultVideoId = "tvdb:" + idMap.TVDB
			} else {
				m.BehaviorHints.DefaultVideoId = idMap.IMDB
			}
		}

		switch ud.Provider.Movie {
		case "tvdb":
			if idMap.TVDB == "" {
				break
			}

			res, err := client.FetchMovie(&tvdb.FetchMovieParams{
				Id: util.SafeParseInt(idMap.TVDB, 0),
			})
			if err != nil {
				SendError(w, r, err)
				return
			}
			data := res.Data

			m.Type = stremio.ContentTypeMovie
			m.Id = idStr
			m.IMDBId = idMap.IMDB
			m.MovieDBId = util.SafeParseInt(idMap.TMDB, 0)
			m.TVDBId = stremio.Number(idMap.TVDB)

			m.Awards = data.Awards.Summary().String()
			m.Name = data.Name
			m.Description = data.Translations.GetOverview()
			m.Country = data.OriginalCountry.Name()

			m.Poster = data.GetPoster()
			m.Background = data.GetBackground()
			m.Logo = data.GetClearLogo()

			firstReleaseDate := util.SafeParseTime(time.DateOnly, data.FirstRelease.Date)
			m.Released = &firstReleaseDate
			m.ReleaseInfo = strconv.Itoa(firstReleaseDate.Year())

			m.Runtime = strconv.Itoa(data.Runtime) + " min"

			trailer := data.GetTrailer()
			if trailer, err := url.Parse(trailer); err == nil && strings.HasSuffix(trailer.Host, "youtube.com") {
				m.Trailers = append(m.Trailers, stremio.MetaTrailer{
					Source: trailer.Query().Get("v"),
					Type:   stremio.MetaTrailerTypeTrailer,
				})
			}

			for i := range data.Genres {
				genre := &data.Genres[i]
				m.Genres = append(m.Genres, genre.Name)
				// m.Links = append(m.Links, stremio.MetaLink{
				// 	Name:     genre.Name,
				// 	Category: stremio.MetaLinkCategoryGenres,
				// 	URL:      "",
				// })
			}

			actors := data.Characters.GetActors()
			for i := range actors {
				actor := &actors[i]
				m.Links = append(m.Links, stremio.MetaLink{
					Name:     actor.PersonName,
					Category: stremio.MetaLinkCategoryCast,
					URL:      "stremio:///search?search=" + url.QueryEscape(actor.PersonName),
				})
			}
			directors := data.Characters.GetDirectors()
			for i := range directors {
				director := &directors[i]
				m.Links = append(m.Links, stremio.MetaLink{
					Name:     director.PersonName,
					Category: stremio.MetaLinkCategoryDirectors,
					URL:      "stremio:///search?search=" + url.QueryEscape(director.PersonName),
				})
			}
		}
	}

	res := stremio.MetaHandlerResponse{
		Meta: m,
	}

	SendResponse(w, r, 200, res)
}
