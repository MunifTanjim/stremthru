package endpoint

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	store_video "github.com/MunifTanjim/stremthru/internal/store/video"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func handleProxyLinkAccess(w http.ResponseWriter, r *http.Request) {
	ctx := server.GetReqCtx(r)
	ctx.RedactURLPathValues(r, "token")

	isGetReq := shared.IsMethod(r, http.MethodGet)
	if !isGetReq && !shared.IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	encodedToken := r.PathValue("token")
	if encodedToken == "" {
		shared.ErrorBadRequest(r, "missing token").Send(w, r)
		return
	}

	user, link, headers, tunnelType, err := shared.UnwrapProxyLinkToken(encodedToken)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if headers != nil {
		for k, v := range headers {
			r.Header.Set(k, v)
		}
	}

	if isGetReq && user != "" {
		cpStore := contentProxyConnectionStore.WithScope(user)

		if limit := config.ContentProxyConnectionLimit.Get(user); limit > 0 {
			activeConnectionCount, err := cpStore.Count()
			if err != nil {
				ctx.Log.Error("[proxy] failed to count connections", "error", err)
			} else if activeConnectionCount >= limit {
				store_video.Redirect(store_video.StoreVideoNameContentProxyLimitReached, w, r)
				return
			}
		}

		if err := cpStore.Set(ctx.RequestId, contentProxyConnection{IP: core.GetRequestIP(r), Link: link}); err != nil {
			ctx.Log.Error("[proxy] failed to record connection", "error", err)
		} else {
			defer cpStore.Del(ctx.RequestId)
		}
	}

	var bytesWritten int64
	if torrentHash := r.URL.Query().Get("torrent_hash"); torrentHash != "" {
		bytesWritten, err = shared.ProxyTorrentResponse(w, r, link, tunnelType, user, torrentHash)
	} else {
		bytesWritten, err = shared.ProxyResponse(w, r, link, tunnelType)
	}
	ctx.Log.Info("[proxy] connection closed", "user", user, "size", util.ToSize(bytesWritten), "error", err)
}

type proxifyLinksData struct {
	Items      []string `json:"items"`
	TotalItems int      `json:"total_items"`
}

func handleProxifyLinks(w http.ResponseWriter, r *http.Request) {
	ctx := server.GetReqCtx(r)

	isGetReq := shared.IsMethod(r, http.MethodGet)
	if !isGetReq && !shared.IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	isAuthorized, user, password := server.GetProxyAuthorization(r, true)
	if !isAuthorized {
		w.Header().Add(server.HEADER_STREMTHRU_AUTHENTICATE, "Basic")
		shared.ErrorForbidden(r).Send(w, r)
		return
	}

	err := r.ParseForm()
	if err != nil {
		shared.ErrorBadRequest(r, "failed to parse data").Send(w, r)
		return
	}

	var links []string
	if isGetReq {
		links = r.Form["url"]
	} else {
		links = r.PostForm["url"]
	}
	count := len(links)
	if count == 0 {
		shared.ErrorBadRequest(r, "missing url").Send(w, r)
		return
	}

	shouldRedirect := isGetReq && r.Form.Get("redirect") != ""

	if shouldRedirect && count > 1 {
		shared.ErrorBadRequest(r, "can not redirect for multiple urls").Send(w, r)
		return
	}

	reqHeadersByBlob := map[string]map[string]string{}
	fallbackReqHeaders := r.Form.Get("req_headers")

	expiresIn := 0 * time.Second
	if exp := r.Form.Get("exp"); exp != "" {
		if c := rune(exp[len(exp)-1]); '0' <= c && c <= '9' {
			exp += "s"
		}
		exp, err := time.ParseDuration(exp)
		if err != nil {
			shared.ErrorBadRequest(r, "invalid expiration").Send(w, r)
			return
		}
		expiresIn = exp
	}

	shouldEncrypt := r.URL.Query().Get("token") == ""
	if !shouldEncrypt {
		ctx.RedactURLQueryParams(r, "token")
	}

	proxyLinks := make([]string, count)
	for i, link := range links {
		idx := strconv.Itoa(i)
		var reqHeaders map[string]string
		reqHeadersBlob := r.Form.Get("req_headers[" + idx + "]")
		if reqHeadersBlob == "" {
			reqHeadersBlob = fallbackReqHeaders
		}
		if headers, ok := reqHeadersByBlob[reqHeadersBlob]; ok {
			reqHeaders = headers
		} else {
			reqHeaders = map[string]string{}
			for header := range strings.SplitSeq(reqHeadersBlob, "\n") {
				if k, v, ok := strings.Cut(header, ": "); ok {
					reqHeaders[k] = v
				}
			}
			reqHeadersByBlob[reqHeadersBlob] = reqHeaders
		}
		filename := r.Form.Get("filename[" + idx + "]")
		proxyLink, err := shared.CreateProxyLink(r, link, reqHeaders, config.TUNNEL_TYPE_AUTO, expiresIn, user, password, shouldEncrypt, filename)
		if err != nil {
			SendError(w, r, err)
			return
		}
		proxyLinks[i] = proxyLink
	}

	if shouldRedirect {
		http.Redirect(w, r, proxyLinks[0], http.StatusFound)
		return
	}

	data := proxifyLinksData{
		Items:      proxyLinks,
		TotalItems: count,
	}

	SendResponse(w, r, 200, data, nil)
}

func AddProxyEndpoints(mux *http.ServeMux) {
	withCors := server.Middleware(shared.EnableCORS)

	mux.HandleFunc("/v0/proxy", withCors(handleProxifyLinks))
	mux.HandleFunc("/v0/proxy/{token}", withCors(handleProxyLinkAccess))
	mux.HandleFunc("/v0/proxy/{token}/{filename}", withCors(handleProxyLinkAccess))
}
