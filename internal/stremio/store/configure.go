package stremio_store

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
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

	td := getTemplateData(ud)

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
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

	var name_config *configure.Config
	var token_config *configure.Config
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "store_name":
			name_config = conf
		case "store_token":
			token_config = conf
		}
	}

	idr := ParsedId{isST: ud.StoreName == ""}
	idr.storeName = store.StoreName(ud.StoreName)
	idr.storeCode = idr.storeName.Code()
	ctx, err := ud.GetRequestContext(r, &idr)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			if uderr.storeName != "" {
				name_config.Error = uderr.storeName
			}
			if uderr.storeToken != "" {
				token_config.Error = uderr.storeToken
			}
		} else {
			SendError(w, r, err)
			return
		}
	}

	if ctx.Store == nil {
		if ud.StoreName == "" {
			token_config.Error = "Invalid Token"
		} else {
			name_config.Error = "Invalid Store"
		}
	} else if token_config.Error == "" {
		params := &store.GetUserParams{}
		params.APIKey = ctx.StoreAuthToken
		user, err := ctx.Store.GetUser(params)
		if err != nil {
			LogError(r, "failed to get user", err)
			token_config.Error = "Invalid Token"
		} else if user.SubscriptionStatus == store.UserSubscriptionStatusExpired {
			token_config.Error = "Subscription Expired"
		}
	}

	if td.HasError() {
		page, err := getPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	url := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/configure")
	q := url.Query()
	q.Set("try_install", "1")
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}
