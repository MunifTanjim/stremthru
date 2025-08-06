package main

import (
	"log"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/endpoint"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
)

func main() {
	config.PrintConfig(&config.AppState{
		StoreNames: []string{
			string(store.StoreNameAlldebrid),
			string(store.StoreNameDebridLink),
			string(store.StoreNameEasyDebrid),
			string(store.StoreNameOffcloud),
			string(store.StoreNamePikPak),
			string(store.StoreNamePremiumize),
			string(store.StoreNameRealDebrid),
			string(store.StoreNameTorBox),
		},
	})

	database := db.Open()
	defer db.Close()
	db.Ping()
	RunSchemaMigration(database.URI, database)

	stopWorkers := worker.InitWorkers()
	defer stopWorkers()

	mux := http.NewServeMux()

	endpoint.AddRootEndpoint(mux)
	endpoint.AddAuthEndpoints(mux)
	endpoint.AddHealthEndpoints(mux)
	endpoint.AddMetaEndpoints(mux)
	endpoint.AddProxyEndpoints(mux)
	endpoint.AddStoreEndpoints(mux)
	endpoint.AddStremioEndpoints(mux)
	endpoint.AddTorrentEndpoints(mux)
	endpoint.AddTorznabEndpoints(mux)
	endpoint.AddExperimentEndpoints(mux)

	handler := shared.RootServerContext(mux)

	addr := ":" + config.Port
	if config.Environment == config.EnvDev {
		addr = "localhost" + addr
	}
	server := &http.Server{Addr: addr, Handler: handler}

	if len(config.ProxyAuthPassword) == 0 {
		server.SetKeepAlivesEnabled(false)
	}

	log.Println("stremthru listening on " + addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start stremthru: %v", err)
	}
}
