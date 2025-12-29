package core

import (
	"net"
	"net/http"
	"strings"
)

var ipRequestHeaders = []string{
	"X-Client-Ip",         // Amazon EC2 / Heroku / others
	"Cf-Connecting-Ip",    // Cloudflare
	"Do-Connecting-Ip",    // DigitalOcean
	"Fastly-Client-Ip",    // Fastly / Firebase
	"True-Client-Ip",      // Akamai / Cloudflare
	"X-Real-Ip",           // nginx
	"X-Cluster-Client-Ip", // Rackspace LB / Riverbed's Stingray
	"X-Forwarded",
	"X-Forwarded-For", // Load-balancers (AWS ELB) / proxies.
	"Forwarded-For",
	"Forwarded",
	"X-Appengine-User-Ip", // Google Cloud App Engine
	"Cf-Pseudo-IPv4",      // Cloudflare fallback
}

func isCorrectIP(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && !ip.IsPrivate() && !ip.IsLoopback()
}

func getClientIPFromXForwardedFor(headers string) (string, bool) {
	if headers == "" {
		return "", false
	}
	for ip := range strings.SplitSeq(headers, ",") {
		if ip, _, _ := strings.Cut(strings.TrimSpace(ip), ":"); isCorrectIP(ip) {
			return ip, true
		}
	}
	return "", false
}

// Credit: https://github.com/pbojinov/request-ip/blob/e1d0f4b89edf26c77cf62b5ef662ba1a0bd1c9fd/src/index.js#L55
func GetRequestIP(r *http.Request) string {
	for _, header := range ipRequestHeaders {
		switch header {
		case "X-Forwarded-For":
			if host, ok := getClientIPFromXForwardedFor(r.Header.Get(header)); ok {
				return host
			}
		default:
			if host := r.Header.Get(header); isCorrectIP(host) {
				return host
			}
		}
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && isCorrectIP(host) {
		return host
	}

	return ""
}

func GetClientIP(r *http.Request) string {
	ip := r.URL.Query().Get("client_ip")
	if isCorrectIP(ip) {
		return ip
	}
	return GetRequestIP(r)
}

func GetRequestIPHeaders(r *http.Request) map[string]string {
	ipHeaders := make(map[string]string)
	for _, header := range ipRequestHeaders {
		if values := r.Header.Values(header); len(values) != 0 {
			ipHeaders[header] = strings.Join(values, " ")
		}
	}
	return ipHeaders
}
