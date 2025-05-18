package stremio_torz

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
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(ud, w, r)
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "cached":
			if ud.CachedOnly {
				conf.Default = "checked"
			}
		}
	}

	if action := r.Header.Get("x-addon-configure-action"); action != "" {
		switch action {
		case "add-store":
			if td.IsAuthed || len(td.Stores) < MaxPublicInstanceStoreCount {
				td.Stores = append(td.Stores, StoreConfig{})
			}
		case "remove-store":
			end := len(td.Stores) - 1
			if end == 0 {
				end = 1
			}
			td.Stores = slices.Clone(td.Stores[0:end])
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
		_, err := ud.GetRequestContext(r)
		if err != nil {
			if uderr, ok := err.(*userDataError); ok {
				for i, err := range uderr.storeCode {
					td.Stores[i].Error.Code = err
				}
				for i, err := range uderr.storeToken {
					td.Stores[i].Error.Token = err
				}
			} else {
				SendError(w, r, err)
				return
			}
		}

		if !td.HasStoreError() {
			s := ud.GetUser()
			if s.HasErr {
				for i, err := range s.Err {
					LogError(r, "failed to access store", err)
					if err == nil {
						continue
					}
					var ts *StoreConfig
					if ud.IsStremThruStore() {
						ts = &td.Stores[0]
						if ts.Error.Token != "" {
							ts.Error.Token += "\n"
						}
						ts.Error.Token += string(ud.GetStoreByIdx(i).Store.GetName()) + ": Failed to access store"
					} else {
						ts = &td.Stores[i]
						ts.Error.Token = "Failed to access store"
					}
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

		stremio_shared.RedirectToConfigurePage(w, r, "torz", ud, true)
		return
	}

	if !hasError && ud.HasRequiredValues() {
		td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/torz/" + ud.GetEncoded() + "/manifest.json").String()
	}

	page, err := getPage(td)
	if err != nil {
		SendError(w, r, err)
		return
	}
	SendHTML(w, 200, page)
}
