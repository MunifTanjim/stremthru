package endpoint

import (
	"io"
	"log"
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

var httpClient = core.DefaultHTTPClient

func ProxyToLink(w http.ResponseWriter, r *http.Request, link string) {
	request, err := http.NewRequest(r.Method, link, nil)
	if err != nil {
		e := shared.ErrorInternalServerError(r, "failed to create request")
		e.Cause = err
		SendError(w, e)
		return
	}

	copyHeaders(r.Header, request.Header)

	response, err := httpClient.Do(request)
	if err != nil {
		error := shared.ErrorBadGateway(r, "failed to request url")
		error.Cause = err
		SendError(w, error)
		return
	}
	defer response.Body.Close()

	copyHeaders(response.Header, w.Header())

	w.WriteHeader(response.StatusCode)

	_, err = io.Copy(w, response.Body)
	if err != nil {
		log.Printf("stream failure: %v", err)
	}
}

func copyHeaders(src http.Header, dest http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		SendError(w, shared.ErrorMethodNotAllowed(r))
		return
	}

	targetUrl := r.URL.Query().Get("url")
	if targetUrl == "" {
		SendError(w, shared.ErrorBadRequest(r, "missing url"))
		return
	}

	targetUrl, err := url.QueryUnescape(targetUrl)
	if err != nil {
		error := shared.ErrorBadRequest(r, "invalid url")
		error.Cause = err
		SendError(w, error)
		return
	}

	if u, err := url.ParseRequestURI(targetUrl); err != nil || u.Scheme == "" || u.Host == "" {
		error := shared.ErrorBadRequest(r, "invalid url")
		error.Cause = err
		SendError(w, error)
		return
	}

	ProxyToLink(w, r, targetUrl)
}

func AddProxyEndpoints(mux *http.ServeMux) {
	withMiddleware := Middleware(ProxyAuthContext, ProxyAuthRequired, ProxyAuthRequired)

	mux.HandleFunc("/proxy", withMiddleware(handleProxy))
	mux.HandleFunc("/v0/proxy", withMiddleware(handleProxy))
}
