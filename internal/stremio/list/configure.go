package stremio_list

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(ud, w, r)

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud := ud.GetEncoded(); eud != "" {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/manifest.json").String()
			}
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	if ud.encoded != "" {
		for i := range td.Configs {
			conf := &td.Configs[i]
			switch conf.Key {
			case "mdblist_api_key":
				if conf.Default == "" {
					conf.Error = "missing mdblist api key"
					continue
				}

				params := &mdblist.GetMyLimitsParams{}
				params.APIKey = conf.Default
				res, err := mdblistClient.GetMyLimits(params)
				if err != nil {
					if res.StatusCode == 403 {
						conf.Error = "Invalid API Key"
					} else {
						conf.Error = err.Error()
					}
					continue
				}
			}
		}
	}

	hasError := td.HasFieldError()

	if IsMethod(r, http.MethodPost) && !hasError {
		err = udManager.Sync(ud)
		if err != nil {
			SendError(w, r, err)
			return
		}

		stremio_shared.RedirectToConfigurePage(w, r, "list", ud, true)
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/list/" + ud.GetEncoded() + "/manifest.json").String()
	}

	page, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, page)
}
