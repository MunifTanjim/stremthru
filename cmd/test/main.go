package main

import (
	"context"
	"io"
	"os"

	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/nntp"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
)

var log = logger.Scoped("test")

// Hardcoded NNTP credentials - replace with your actual values
const (
	nntpHost     = "localhost"
	nntpPort     = 119
	nntpUsername = ""
	nntpPassword = ""
	nntpTLS      = false
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: test <nzb-file>")
	}

	nzbPath := os.Args[1]

	nzbFile, err := os.Open(nzbPath)
	if err != nil {
		log.Fatal("Failed to open NZB file", "error", err)
	}

	nzbDoc, err := nzb.Parse(nzbFile)
	nzbFile.Close()
	if err != nil {
		log.Fatal("Failed to parse NZB file", "error", err)
	}

	log.Info("NZB parsed", "files", nzbDoc.FileCount(), "size", util.ToSize(nzbDoc.TotalSize()))

	// Create UsenetPool with hardcoded credentials
	usenetPool, err := usenet_pool.NewPool(&usenet_pool.Config{
		Providers: []usenet_pool.ProviderConfig{{
			PoolConfig: nntp.PoolConfig{
				ConnectionConfig: nntp.ConnectionConfig{
					Host:     nntpHost,
					Port:     nntpPort,
					Username: nntpUsername,
					Password: nntpPassword,
					TLS:      nntpTLS,
				},
				MaxSize: 5,
			},
		}},
	})
	if err != nil {
		log.Fatal("Failed to create Usenet pool", "error", err)
	}
	defer usenetPool.Close()

	ctx := context.Background()
	stream, err := usenetPool.StreamLargestFile(ctx, nzbDoc, nil)
	if err != nil {
		log.Fatal("Failed to create file stream", "error", err)
	}
	defer stream.Close()

	log.Info("Starting download", "name", stream.Name, "size", util.ToSize(stream.Size))

	// Create output file
	outFile, err := os.Create(stream.Name)
	if err != nil {
		log.Fatal("Failed to create output file", "error", err)
	}
	defer outFile.Close()

	// Copy stream to file
	written, err := io.Copy(outFile, stream)
	if err != nil {
		log.Fatal("Failed to write output file", "error", err)
	}

	log.Info("Download completed", "size", util.ToSize(written), "file", stream.Name)
}
