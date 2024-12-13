package endpoint

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/shared"
)

func parseBasicAuthToken(token string) (encoded, username, password string, ok bool) {
	decoded := token

	if strings.ContainsRune(decoded, ':') {
		encoded = core.Base64Encode(decoded)
	} else {
		encoded = decoded
		d, err := core.Base64Decode(encoded)
		if err != nil {
			return "", "", "", false
		}
		decoded = d
	}

	username, password, ok = strings.Cut(strings.TrimSpace(decoded), ":")

	return encoded, username, password, ok
}

func extractProxyAuthToken(r *http.Request) (token string, hasToken bool) {
	token = r.Header.Get("Proxy-Authorization")
	r.Header.Del("Proxy-Authorization")
	token = strings.TrimPrefix(token, "Basic ")
	return token, token != ""
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	targetUrl := r.URL.Query().Get("url")
	if targetUrl == "" {
		shared.ErrorBadRequest(r, "missing url").Send(w)
		return
	}

	targetUrl, err := url.QueryUnescape(targetUrl)
	if err != nil {
		e := shared.ErrorBadRequest(r, "invalid url")
		e.Cause = err
		e.Send(w)
		return
	}

	if u, err := url.ParseRequestURI(targetUrl); err != nil || u.Scheme == "" || u.Host == "" {
		e := shared.ErrorBadRequest(r, "invalid url")
		e.Cause = err
		e.Send(w)
		return
	}

	shared.ProxyResponse(w, r, targetUrl)
}

func AddProxyEndpoints(mux *http.ServeMux) {
	withMiddleware := Middleware(ProxyAuthContext, ProxyAuthRequired, ProxyAuthRequired)

	mux.HandleFunc("/proxy", withMiddleware(handleProxy))
	mux.HandleFunc("/v0/proxy", withMiddleware(handleProxy))
}
