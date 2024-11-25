package core

import (
	"net"
	"net/http"
	"strings"
)

var ipRequestHeaders = []string{
	"X-Client-Ip",         // Amazon EC2 / Heroku / others
	"X-Forwarded-For",     // Load-balancers (AWS ELB) / proxies.
	"Cf-Connecting-Ip",    // Cloudflare
	"Do-Connecting-Ip",    // DigitalOcean
	"Fastly-Client-Ip",    // Fastly / Firebase
	"True-Client-Ip",      // Akamai / Cloudflare
	"X-Real-Ip",           // nginx
	"X-Cluster-Client-Ip", // Rackspace LB / Riverbed's Stingray
	"X-Forwarded",
	"Forwarded-For",
	"Forwarded",
	"X-Appengine-User-Ip", // Google Cloud App Engine
	"Cf-Pseudo-IPv4",      // Cloudflare fallback
}

func isCorrectIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func getClientIPFromXForwardedFor(headers string) (string, bool) {
	if headers == "" {
		return "", false
	}
	forwardedIps := strings.Split(headers, ",")
	for _, ip := range forwardedIps {
		if ip, _, found := strings.Cut(strings.TrimSpace(ip), ":"); found && isCorrectIP(ip) {
			return ip, true
		}
	}
	return "", false
}

// Credit: https://github.com/pbojinov/request-ip/blob/e1d0f4b89edf26c77cf62b5ef662ba1a0bd1c9fd/src/index.js#L55
func GetClientIP(r *http.Request) string {
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

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err != nil && isCorrectIP(host) {
		return host
	}

	return ""
}
