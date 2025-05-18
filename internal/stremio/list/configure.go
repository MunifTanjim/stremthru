package stremio_list

import (
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
)

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	udErr := userDataError{}
	if err != nil {
		if e, ok := err.(userDataError); !ok {
			SendError(w, r, err)
			return
		} else {
			udErr = e
		}
	}

	td := getTemplateData(ud, udErr, w, r)

	if action := r.Header.Get("x-addon-configure-action"); action != "" {
		switch action {
		case "add-mdblist-list":
			if td.IsAuthed || len(td.MDBList.Lists) < MaxPublicInstanceMDBListListCount {
				td.MDBList.Lists = append(td.MDBList.Lists, TemplateDataMDBListList{
					URL: "",
				})
			}
		case "remove-mdblist-list":
			end := len(td.MDBList.Lists) - 1
			if end == 0 {
				end = 1
			}
			td.MDBList.Lists = slices.Clone(td.MDBList.Lists[0:end])
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/list/" + ud.GetEncoded() + "/manifest.json").String()
		}

		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
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
