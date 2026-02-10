package shared

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
)

func IsMethod(r *http.Request, method string) bool {
	return r.Method == method
}

func GetQueryInt(queryParams url.Values, name string, defaultValue int) (int, error) {
	if qVal, ok := queryParams[name]; ok {
		v := qVal[0]
		if v == "" {
			return defaultValue, nil
		}

		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, errors.New("invalid " + name)
		}
		return val, nil
	}
	return defaultValue, nil
}

func ReadRequestBodyJSON[T any](r *http.Request, payload T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ErrorUnsupportedMediaType(r)
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err == nil {
		return err
	}
	if err == io.EOF {
		return ErrorBadRequest(r, "missing body")
	}
	error := core.NewAPIError("failed to decode body")
	error.Cause = err
	return error
}

type response struct {
	Data  any   `json:"data,omitempty"`
	Error error `json:"error,omitempty"`
}

func (res response) send(w http.ResponseWriter, r *http.Request, statusCode int) {
	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(res); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func SendError(w http.ResponseWriter, r *http.Request, err error) {
	var e core.StremThruError
	if sterr, ok := err.(core.StremThruError); ok {
		e = sterr
	} else if aerr, ok := err.(*server.APIError); ok {
		e = &core.Error{
			RequestId:  aerr.RequestId,
			Type:       core.ErrorTypeAPI,
			Code:       aerr.Code,
			Msg:        aerr.Message,
			Method:     aerr.Method,
			Path:       aerr.Path,
			StatusCode: aerr.StatusCode,
			Cause:      aerr.Cause,
		}
	} else {
		e = &core.Error{Cause: err}
	}
	e.Pack(r)

	ctx := server.GetReqCtx(r)
	ctx.Error = err

	res := &response{}
	res.Error = e.GetError()

	res.send(w, r, e.GetStatusCode())
}

func SendResponse(w http.ResponseWriter, r *http.Request, statusCode int, data any, err error) {
	if err != nil {
		SendError(w, r, err)
		return
	}

	res := &response{}
	res.Data = data

	res.send(w, r, statusCode)
}

func SendHTML(w http.ResponseWriter, statusCode int, data bytes.Buffer) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	data.WriteTo(w)
}

func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		core.LogError(r, "failed to encode json", err)
	}
}

func SendXML(w http.ResponseWriter, r *http.Request, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)
	encoder := xml.NewEncoder(w)
	encoder.Indent("", " ")
	w.Write([]byte(xml.Header))
	if err := encoder.Encode(v); err != nil {
		core.LogError(r, "failed to encode xml", err)
	}
}

func copyHeaders(src http.Header, dest http.Header, stripIpHeaders bool) {
	for key, values := range src {
		if stripIpHeaders {
			switch strings.ToLower(key) {
			case "x-client-ip", "x-forwarded-for", "cf-connecting-ip", "do-connecting-ip", "fastly-client-ip", "true-client-ip", "x-real-ip", "x-cluster-client-ip", "x-forwarded", "forwarded-for", "forwarded", "x-appengine-user-ip", "cf-pseudo-ipv4":
				continue
			}
		}
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

var proxyHttpClientByTunnelType = map[config.TunnelType]*http.Client{
	config.TUNNEL_TYPE_NONE: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_NONE)
		return &http.Client{
			Transport: transport,
		}
	}(),
	config.TUNNEL_TYPE_AUTO: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_AUTO)
		return &http.Client{
			Transport: transport,
		}
	}(),
	config.TUNNEL_TYPE_FORCED: func() *http.Client {
		transport := config.DefaultHTTPTransport.Clone()
		transport.Proxy = config.Tunnel.GetProxy(config.TUNNEL_TYPE_FORCED)
		return &http.Client{
			Transport: transport,
		}
	}(),
}

func ProxyResponse(w http.ResponseWriter, r *http.Request, url string, tunnelType config.TunnelType) (bytesWritten int64, err error) {
	request, err := http.NewRequest(r.Method, url, nil)
	if err != nil {
		e := ErrorInternalServerError(r, "failed to create request")
		e.Cause = err
		SendError(w, r, e)
		return
	}

	copyHeaders(r.Header, request.Header, true)

	proxyHttpClient := proxyHttpClientByTunnelType[tunnelType]

	response, err := proxyHttpClient.Do(request)
	if err != nil {
		e := ErrorBadGateway(r, "failed to request url")
		e.Cause = err
		SendError(w, r, e)
		return
	}
	defer response.Body.Close()

	copyHeaders(response.Header, w.Header(), false)

	w.WriteHeader(response.StatusCode)

	return io.Copy(w, response.Body)
}

func extractRequestScheme(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")

	if scheme == "" {
		scheme = r.URL.Scheme
	}

	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	return scheme
}

func extractRequestHost(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		host = r.Host
	}

	return host
}

func GetReversedHostname(r *http.Request) string {
	hostname := extractRequestHost(r)
	hostname, _, _ = strings.Cut(hostname, ":")
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return "local.stremthru"
	}
	parts := strings.Split(hostname, ".")
	slices.Reverse(parts)
	return strings.Join(parts, ".")
}

func ExtractRequestBaseURL(r *http.Request) *url.URL {
	return &url.URL{
		Scheme: extractRequestScheme(r),
		Host:   extractRequestHost(r),
	}
}

func GetClientIP(r *http.Request, ctx *storecontext.Context) string {
	if !ctx.IsProxyAuthorized {
		return core.GetClientIP(r)
	}
	if ctx.Store != nil && config.StoreTunnel.GetTypeForAPI(string(ctx.Store.GetName())) == config.TUNNEL_TYPE_NONE {
		return config.IP.GetMachineIP()
	}
	return ""
}
