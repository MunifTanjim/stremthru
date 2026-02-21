package main

import (
	"log"
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/endpoint"
	"github.com/MunifTanjim/stremthru/internal/job"
	"github.com/MunifTanjim/stremthru/internal/posthog"
	"github.com/MunifTanjim/stremthru/internal/shared"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
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
			string(store.StoreNameQBittorrent),
		},
	})

	posthog.Init()
	defer posthog.Close()

	database := db.Open()
	defer db.Close()
	db.Ping()
	RunSchemaMigration(database.URI, database)

	defer cache.ClosePersistentCaches()
	defer usenetmanager.Close()

	stopWorkers := worker.InitWorkers()
	defer stopWorkers()

	stopJobs := job.InitJobs()
	defer stopJobs()

	mux := http.NewServeMux()

	endpoint.AddRootEndpoint(mux)
	endpoint.AddDashEndpoint(mux)
	endpoint.AddAuthEndpoints(mux)
	endpoint.AddHealthEndpoints(mux)
	endpoint.AddMetaEndpoints(mux)
	endpoint.AddProxyEndpoints(mux)
	endpoint.AddStoreEndpoints(mux)
	endpoint.AddStremioEndpoints(mux)
	endpoint.AddTorrentEndpoints(mux)
	endpoint.AddTorznabEndpoints(mux)
	endpoint.AddNewznabEndpoints(mux)
	endpoint.AddExperimentEndpoints(mux)
	endpoint.AddEndpoints(mux)

	handler := shared.RootServerContext(mux)

	server := &http.Server{Addr: config.ListenAddr, Handler: handler}

	if config.IsPublicInstance {
		server.SetKeepAlivesEnabled(false)
	}

	log.Println("stremthru listening on " + config.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start stremthru: %v", err)
	}
}
