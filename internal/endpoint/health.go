package endpoint

import (
	"errors"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
)

type HealthData struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := &HealthData{}
	health.Status = "ok"
	SendResponse(w, r, 200, health, nil)
}

type HealthDebugDataIP struct {
	Client  string            `json:"client"`
	Exposed map[string]string `json:"exposed"`
	Machine string            `json:"machine"`
	Tunnel  map[string]string `json:"tunnel"`
}

type HealthDebugDataStore struct {
	Default string   `json:"default"`
	Names   []string `json:"names"`
}

type HealthDebugDataUser struct {
	Name  string               `json:"name"`
	Store HealthDebugDataStore `json:"store"`
}

type HealthDebugData struct {
	Time    string               `json:"time"`
	Version string               `json:"version"`
	User    *HealthDebugDataUser `json:"user,omitempty"`
	IP      *HealthDebugDataIP   `json:"ip,omitempty"`
}

func handleHealthDebug(w http.ResponseWriter, r *http.Request) {
	ctx := context.GetStoreContext(r)

	data := &HealthDebugData{
		Time:    time.Now().Format(time.RFC3339),
		Version: config.Version,
	}

	if ctx.IsProxyAuthorized {
		data.User = &HealthDebugDataUser{
			Name: ctx.ProxyAuthUser,
			Store: HealthDebugDataStore{
				Default: config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser),
				Names:   config.StoreAuthToken.ListStores(ctx.ProxyAuthUser),
			},
		}

		clientIp := core.GetClientIP(r)
		machineIp := config.IP.GetMachineIP()

		ipMapErrs := []error{}
		tunnelIpMap, err := config.IP.GetTunnelIPByProxyHost()
		if err != nil {
			if defaultProxyHost := config.Tunnel.GetDefaultProxyHost(); defaultProxyHost != "" && tunnelIpMap[defaultProxyHost] == "" {
				SendError(w, r, err)
				return
			}
			ipMapErrs = append(ipMapErrs, err)
		}

		tunnel := map[string]string{}
		maps.Copy(tunnel, tunnelIpMap)

		exposedIpMap, err := config.IP.GetTunnelIPByHostname()
		if err != nil {
			if exposedIpMap["*"] == "" {
				SendError(w, r, err)
				return
			}
			ipMapErrs = append(ipMapErrs, err)
		}

		exposed := map[string]string{}
		maps.Copy(exposed, exposedIpMap)
		if os.Getenv("NO_PROXY") == "*" {
			exposed["*"] = machineIp
		}

		if ipMapErr := errors.Join(ipMapErrs...); ipMapErr != nil {
			reqCtx := server.GetReqCtx(r)
			reqCtx.Log.Warn("Failed to get tunnel ip map", "error", ipMapErr)
		}

		for storeName := range config.StoreTunnel {
			switch config.StoreTunnel.GetTypeForAPI(storeName) {
			case config.TUNNEL_TYPE_FORCED:
				exposed[":"+storeName+":api:"] = exposed["*"]
			case config.TUNNEL_TYPE_NONE:
				exposed[":"+storeName+":api:"] = machineIp
			}

			switch config.StoreTunnel.GetTypeForStream(storeName) {
			case config.TUNNEL_TYPE_FORCED:
				exposed[":"+storeName+":stream:"] = exposed["*"]
			case config.TUNNEL_TYPE_NONE:
				exposed[":"+storeName+":stream:"] = machineIp
			}
		}

		data.IP = &HealthDebugDataIP{
			Client:  clientIp,
			Exposed: exposed,
			Machine: machineIp,
			Tunnel:  tunnel,
		}
	}

	SendResponse(w, r, 200, data, nil)
}

func AddHealthEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v0/health", handleHealth)
	mux.HandleFunc("/v0/health/__debug__", StoreMiddleware(ProxyAuthContext, StoreContext)(handleHealthDebug))
}
