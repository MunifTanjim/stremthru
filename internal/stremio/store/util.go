package stremio_store

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/shared"
)

var IsMethod = shared.IsMethod
var SendError = shared.SendError
var ExtractRequestBaseURL = shared.ExtractRequestBaseURL

func SendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		LogError(r, "failed to encode json", err)
	}
}

func SendHTML(w http.ResponseWriter, statusCode int, data bytes.Buffer) {
	shared.SendHTML(w, statusCode, data)
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
