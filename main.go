package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/endpoint"
	imdbmeta_db "github.com/MunifTanjim/stremthru/internal/imdb_meta/db"
	imdbmeta_plugin "github.com/MunifTanjim/stremthru/internal/imdb_meta/plugin"
	"github.com/MunifTanjim/stremthru/internal/job"
	meta_imdb "github.com/MunifTanjim/stremthru/internal/meta/imdb"
	newznab_indexer "github.com/MunifTanjim/stremthru/internal/newznab/indexer"
	newznab_stats "github.com/MunifTanjim/stremthru/internal/newznab/stats"
	"github.com/MunifTanjim/stremthru/internal/posthog"
	"github.com/MunifTanjim/stremthru/internal/shared"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
)

func main() {
	config.PrintConfig(&config.AppState{
		StoreNames: util.SliceMapToString(store.StoreNames),
	})

	posthog.Init()
	defer posthog.Close()

	database := db.Open()
	defer db.Close()
	db.Ping()
	RunSchemaMigration(database.URI, database)

	var imdbSvc *meta_imdb.Service
	if meta_imdb.IsEnabled() {
		imdbDB, err := imdbmeta_db.Open("")
		if err != nil {
			log.Fatalf("failed to open imdb database: %v", err)
		}
		defer imdbDB.Close()
		if err := imdbDB.Migrate(); err != nil {
			log.Fatalf("failed to migrate imdb database: %v", err)
		}
		if err := imdbDB.PurgeBlankValues(); err != nil {
			log.Fatalf("failed to purge blank imdb values: %v", err)
		}
		if err := imdbDB.Ping(); err != nil {
			log.Fatalf("failed to ping imdb database: %v", err)
		}
		imdbClient := &imdbmeta_plugin.Client{}
		defer imdbClient.Close()
		imdbSvc = meta_imdb.NewService(imdbDB, imdbClient)
	} else if config.Feature.HasMeta() {
		log.Printf("imdb meta disabled (plugin binary missing: %s)", config.PluginIMDBPath())
	}

	if err := newznab_indexer.LoadIndexerIDByHostnameMap(); err != nil {
		log.Fatalf("failed to load newznab indexer id by hostname map: %v\n", err)
	}

	defer cache.ClosePersistentCaches()
	defer usenetmanager.Close()

	newznab_stats.InitBackgroundJob()
	defer newznab_stats.CleanupBackgroundJob()

	stopWorkers := worker.InitWorkers()
	defer stopWorkers()

	stopJobs := job.InitJobs()
	defer stopJobs()

	mux := http.NewServeMux()

	endpoint.AddRootEndpoint(mux)
	endpoint.AddDashEndpoint(mux)
	endpoint.AddAuthEndpoints(mux)
	endpoint.AddMaintenanceEndpoint(mux)
	endpoint.AddHealthEndpoints(mux)
	endpoint.AddMetaEndpoints(mux)
	if imdbSvc != nil {
		meta_imdb.AddEndpoints(mux, imdbSvc)
	}
	endpoint.AddProxyEndpoints(mux)
	endpoint.AddStoreEndpoints(mux)
	endpoint.AddStremioEndpoints(mux)
	endpoint.AddTorrentEndpoints(mux)
	endpoint.AddTorznabEndpoints(mux)
	endpoint.AddNewznabEndpoints(mux)
	endpoint.AddExperimentEndpoints(mux)
	endpoint.AddDebugEndpoints(mux)
	endpoint.AddEndpoints(mux)

	handler := shared.RootServerContext(mux)

	server := &http.Server{Addr: config.ListenAddr, Handler: handler}

	if config.IsPublicInstance {
		server.SetKeepAlivesEnabled(false)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("stremthru listening on " + config.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start stremthru: %v", err)
		}
	}()

	sig := <-quit
	log.Printf("received signal: %v, shutting down...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
