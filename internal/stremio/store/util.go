package stremio_store

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var ExtractRequestBaseURL = shared.ExtractRequestBaseURL

var SendResponse = stremio_shared.SendResponse
var SendHTML = stremio_shared.SendHTML
var GetPathValue = stremio_shared.GetPathValue

func getId(r *http.Request) string {
	return GetPathValue(r, "id")
}

func parseStremId(sid string) (sType, sId string, season, episode int) {
	season, episode = -1, -1
	stremId, stremSpecs, isSeries := strings.Cut(sid, ":")
	sId = stremId
	if isSeries {
		sType = "series"
		if strS, strEp, ok := strings.Cut(stremSpecs, ":"); ok {
			intS, errS := strconv.Atoi(strS)
			intEp, errEp := strconv.Atoi(strEp)
			if errS == nil && errEp == nil {
				season = intS
				episode = intEp
			}
		}
	} else {
		sType = "movie"
	}
	return sType, sId, season, episode
}

func getContentType(r *http.Request) (string, *core.APIError) {
	contentType := r.PathValue("contentType")
	if contentType != ContentTypeOther {
		return "", shared.ErrorBadRequest(r, "unsupported type: "+contentType)
	}
	return contentType, nil
}
