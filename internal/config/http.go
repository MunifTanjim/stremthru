package config

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type TunnelType string

const (
	TUNNEL_TYPE_NONE   TunnelType = ""
	TUNNEL_TYPE_AUTO   TunnelType = "a"
	TUNNEL_TYPE_FORCED TunnelType = "f"
)

type TunnelMap map[string]url.URL

func (tm TunnelMap) hasProxy() bool {
	for _, proxyUrl := range tm {
		if proxyUrl.Host != "" {
			return true
		}
	}
	return false
}

func (tm TunnelMap) GetDefaultProxyHost() string {
	if proxy := tm.getProxy("*"); proxy != nil && proxy.Host != "" {
		return proxy.Host
	}
	return ""
}

func (tm TunnelMap) getProxy(hostname string) *url.URL {
	hn := hostname
	for {
		if proxy, ok := tm[hn]; ok {
			if hn != hostname {
				tm[hostname] = proxy
			}
			return &proxy
		}

		_, hn, _ = strings.Cut(hn, ".")
		if hn == "" {
			break
		}
	}
	return nil
}

// If tunnel is configured for `hostname` use that.
// Otherwise fallback to environment proxy, i.e. `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`
func (tm TunnelMap) autoProxy(r *http.Request) (*url.URL, error) {
	proxy := tm.getProxy(r.URL.Hostname())
	if proxy == nil {
		return http.ProxyFromEnvironment(r)
	}
	if proxy.Host == "" {
		return nil, nil
	}
	return proxy, nil
}

// Use the default tunnel, ignore `NO_PROXY`
func (tm TunnelMap) forcedProxy(r *http.Request) (*url.URL, error) {
	if proxy := tm.getProxy(r.URL.Hostname()); proxy != nil && proxy.Host != "" {
		return proxy, nil
	}
	if proxy := tm.getProxy("*"); proxy != nil && proxy.Host != "" {
		return proxy, nil
	}
	return nil, nil
}

func (tm TunnelMap) GetProxy(tunnelType TunnelType) func(req *http.Request) (*url.URL, error) {
	switch tunnelType {
	case TUNNEL_TYPE_AUTO:
		return tm.autoProxy
	case TUNNEL_TYPE_FORCED:
		return tm.forcedProxy
	case TUNNEL_TYPE_NONE:
		return nil
	default:
		panic("invalid tunnel type")
	}
}

func parseTunnel(httpProxy, httpsProxy, tunnel string) TunnelMap {
	tunnelMap := make(TunnelMap)

	defaultProxy := &url.URL{}

	if value := httpProxy; len(value) > 0 {
		if err := os.Setenv("HTTP_PROXY", value); err != nil {
			log.Fatal("failed to set http_proxy")
		}
		if err := os.Setenv("HTTPS_PROXY", value); err != nil {
			log.Fatal("failed to set https_proxy")
		}
		if u, err := url.Parse(value); err == nil {
			defaultProxy = u
		}
	}

	// deprecated
	if value := httpsProxy; len(value) > 0 {
		if err := os.Setenv("HTTPS_PROXY", value); err != nil {
			log.Fatal("failed to set https_proxy")
		}
		if defaultProxy.Host == "" {
			if u, err := url.Parse(value); err == nil {
				defaultProxy = u
			}
		}
	}

	tunnelMap["*"] = *defaultProxy

	tunnelList := strings.FieldsFunc(tunnel, func(c rune) bool {
		return c == ','
	})

	for _, tunnel := range tunnelList {
		if hostname, proxy, ok := strings.Cut(tunnel, ":"); ok {
			if hostname == "*" {
				if proxy == "false" {
					if err := os.Setenv("NO_PROXY", "*"); err != nil {
						log.Fatal("failed to set no_proxy")
					}
				} else if proxy == "true" {
					if err := os.Unsetenv("NO_PROXY"); err != nil {
						log.Fatal("failed to unset no_proxy")
					}
				}
				continue
			}

			switch proxy {
			case "false":
				tunnelMap[hostname] = url.URL{}
			case "true":
				tunnelMap[hostname] = *defaultProxy
			default:
				if u, err := url.Parse(proxy); err == nil {
					tunnelMap[hostname] = *u
				}
			}
		}
	}

	return tunnelMap
}

var Tunnel = func() TunnelMap {
	httpProxy := getEnv("STREMTHRU_HTTP_PROXY")
	// deprecated
	httpsProxy := getEnv("STREMTHRU_HTTPS_PROXY")
	if httpsProxy == "" {
		httpsProxy = httpProxy
	}
	tunnel := getEnv("STREMTHRU_TUNNEL")
	return parseTunnel(httpProxy, httpsProxy, tunnel)
}()

type StoreTunnelConfig struct {
	api    bool
	stream bool
}

type StoreTunnelConfigMap map[string]StoreTunnelConfig

func (stc StoreTunnelConfigMap) isEnabledForAPI(name string) bool {
	if c, ok := stc[name]; ok {
		return c.api
	}
	if name != "*" {
		return stc.isEnabledForAPI("*")
	}
	return true
}

func (stc StoreTunnelConfigMap) GetTypeForAPI(name string) TunnelType {
	enabled := stc.isEnabledForAPI(name)
	if enabled {
		return TUNNEL_TYPE_FORCED
	}
	return TUNNEL_TYPE_NONE
}

func (stc StoreTunnelConfigMap) isEnabledForStream(name string) bool {
	if c, ok := stc[name]; ok {
		return c.stream
	}
	if name != "*" {
		return stc.isEnabledForStream("*")
	}
	return true
}

func (stc StoreTunnelConfigMap) GetTypeForStream(name string) TunnelType {
	enabled := stc.isEnabledForStream(name)
	if enabled {
		return TUNNEL_TYPE_FORCED
	}
	return TUNNEL_TYPE_NONE
}

func parseStoreTunnel(storeTunnel string, tunnelMap TunnelMap) StoreTunnelConfigMap {
	storeTunnelList := strings.FieldsFunc(storeTunnel, func(c rune) bool {
		return c == ','
	})

	contentHostnameByStore := map[string]string{
		"alldebrid":  "debrid.it",
		"debridlink": "debrid.link",
		"premiumize": "energycdn.com",
		"realdebrid": "download.real-debrid.com",
		"torbox":     "tb-cdn.st",
	}

	storeTunnelMap := make(StoreTunnelConfigMap)
	for _, storeTunnel := range storeTunnelList {
		if store, tunnel, ok := strings.Cut(storeTunnel, ":"); ok {
			storeTunnelMap[store] = StoreTunnelConfig{
				api:    tunnel == "true" || tunnel == "api",
				stream: tunnel == "true",
			}

			switch store {
			case "*":
				for _, hostname := range contentHostnameByStore {
					if _, exists := tunnelMap[hostname]; !exists {
						if tunnel == "true" {
							tunnelMap[hostname] = *tunnelMap.getProxy("*")
						} else {
							tunnelMap[hostname] = url.URL{}
						}
					}
				}
			default:
				if hostname, ok := contentHostnameByStore[store]; ok {
					if tunnel == "true" {
						tunnelMap[hostname] = *tunnelMap.getProxy("*")
					} else {
						tunnelMap[hostname] = url.URL{}
					}
				}
			}
		}
	}

	return storeTunnelMap
}

var StoreTunnel = func() StoreTunnelConfigMap {
	return parseStoreTunnel(getEnv("STREMTHRU_STORE_TUNNEL"), Tunnel)
}()

// has auto proxy
var DefaultHTTPTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = Tunnel.GetProxy(TUNNEL_TYPE_AUTO)
	transport.DisableKeepAlives = true
	return transport
}()

var DefaultHTTPClient = func() *http.Client {
	transport := DefaultHTTPTransport.Clone()
	return &http.Client{
		Transport: transport,
		Timeout:   90 * time.Second,
	}
}()

func GetHTTPClient(tunnelType TunnelType) *http.Client {
	transport := DefaultHTTPTransport.Clone()
	transport.Proxy = Tunnel.GetProxy(tunnelType)
	return &http.Client{
		Transport: transport,
		Timeout:   90 * time.Second,
	}
}

func getHTTPClientWithProxy(proxyUrl *url.URL) *http.Client {
	transport := DefaultHTTPTransport.Clone()
	transport.Proxy = func(r *http.Request) (*url.URL, error) {
		return proxyUrl, nil
	}
	return &http.Client{
		Transport: transport,
		Timeout:   90 * time.Second,
	}
}

type IPResolver struct {
	machineIP string

	checker            string
	proxyIpByHostname  map[string]string
	proxyIpByProxyHost map[string]string
	proxyIpMapStaleAt  time.Time
	m                  sync.Mutex
}

func (ipr *IPResolver) getIp(client *http.Client) (string, error) {
	switch ipr.checker {
	case "aws", "amazon":
		req, err := http.NewRequest(http.MethodGet, "https://checkip.amazonaws.com", nil)
		if err != nil {
			return "", err
		}
		res, err := client.Do(req)
		if err != nil {
			return "", err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(body)), nil
	case "akamai":
		req, err := http.NewRequest(http.MethodGet, "https://whatismyip.akamai.com", nil)
		if err != nil {
			return "", err
		}
		res, err := client.Do(req)
		if err != nil {
			return "", err
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(body)), nil
	default:
		return "", errors.New("invalid ip checker: " + ipr.checker)
	}
}

func (ipr *IPResolver) GetMachineIP() string {
	if ipr.machineIP == "" {
		client := GetHTTPClient(TUNNEL_TYPE_NONE)
		client.Timeout = 30 * time.Second
		ip, err := ipr.getIp(client)
		if err != nil {
			log.Panicf("Failed to detect Machine IP: %v\n", err)
		}
		ipr.machineIP = ip
	}
	return ipr.machineIP
}

func (ipr *IPResolver) GetTunnelIP() (string, error) {
	client := GetHTTPClient(TUNNEL_TYPE_FORCED)
	client.Timeout = 30 * time.Second
	ip, err := ipr.getIp(client)
	if err != nil {
		return "", err
	}
	return ip, nil
}

func (ipr *IPResolver) resolveTunnelIPMap() error {
	ipr.m.Lock()
	defer ipr.m.Unlock()

	if !ipr.proxyIpMapStaleAt.Before(time.Now()) {
		return nil
	}

	proxyIpByProxyHost := map[string]string{}
	proxyIpByHostname := map[string]string{}
	errs := []error{}

	for hostname, u := range Tunnel {
		if ip, ok := proxyIpByProxyHost[u.Host]; ok {
			proxyIpByHostname[hostname] = ip
			continue
		}
		var ip string
		if u.Host == "" {
			ip = ipr.GetMachineIP()
		} else {
			client := getHTTPClientWithProxy(&u)
			client.Timeout = 30 * time.Second
			if proxyIp, err := ipr.getIp(client); err == nil {
				ip = proxyIp
			} else {
				errs = append(errs, err)
			}
		}
		proxyIpByHostname[hostname] = ip
		proxyIpByProxyHost[u.Host] = ip
	}

	delete(proxyIpByProxyHost, "")

	ipr.proxyIpByHostname = proxyIpByHostname
	ipr.proxyIpByProxyHost = proxyIpByProxyHost
	ipr.proxyIpMapStaleAt = time.Now().Add(30 * time.Minute)

	return errors.Join(errs...)
}

func (ipr *IPResolver) GetTunnelIPByProxyHost() (map[string]string, error) {
	err := ipr.resolveTunnelIPMap()
	return ipr.proxyIpByProxyHost, err
}

func (ipr *IPResolver) GetTunnelIPByHostname() (map[string]string, error) {
	err := ipr.resolveTunnelIPMap()
	return ipr.proxyIpByHostname, err
}
