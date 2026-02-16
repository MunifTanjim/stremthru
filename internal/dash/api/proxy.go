package dash_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

type proxifyLinkRequest struct {
	URL        string `json:"url"`
	Exp        string `json:"exp"`
	ReqHeaders string `json:"req_headers"`
	Filename   string `json:"filename"`
	Encrypt    *bool  `json:"encrypt"`
}

type proxifyLinkResponse struct {
	URL string `json:"url"`
}

func handleProxifyLink(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ctx := GetReqCtx(r)

	var payload proxifyLinkRequest
	if err := ReadRequestBodyJSON(r, &payload); err != nil {
		ErrorBadRequest(r).WithMessage("failed to parse request body").Send(w, r)
		return
	}

	if payload.URL == "" {
		ErrorBadRequest(r).WithMessage("missing url").Send(w, r)
		return
	}

	user := ctx.Session.User
	password := config.UserAuth.GetPassword(user)

	expiresIn := 0 * time.Second
	if payload.Exp != "" {
		exp := payload.Exp
		if c := rune(exp[len(exp)-1]); '0' <= c && c <= '9' {
			exp += "s"
		}
		d, err := time.ParseDuration(exp)
		if err != nil {
			ErrorBadRequest(r).WithMessage("invalid expiration").Send(w, r)
			return
		}
		expiresIn = d
	}

	reqHeaders := map[string]string{}
	if payload.ReqHeaders != "" {
		for header := range strings.SplitSeq(payload.ReqHeaders, "\n") {
			if k, v, ok := strings.Cut(header, ": "); ok {
				reqHeaders[k] = v
			}
		}
	}

	shouldEncrypt := payload.Encrypt == nil || *payload.Encrypt
	proxyLink, err := shared.CreateProxyLink(r, payload.URL, reqHeaders, config.TUNNEL_TYPE_AUTO, expiresIn, user, password, shouldEncrypt, payload.Filename)
	if err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, proxifyLinkResponse{URL: proxyLink})
}

func AddProxyEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/proxy", authed(handleProxifyLink))
}
