package dash_api

import (
	"net/http"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type UsenetConfigIndexerRequestHeader struct {
	Query map[string]map[string]string `json:"query"`
	Grab  map[string]string            `json:"grab"`
}

type UsenetConfig struct {
	NZBCacheSize           string                           `json:"nzb_cache_size"`
	NZBCacheTTL            string                           `json:"nzb_cache_ttl"`
	NZBMaxFileSize         string                           `json:"nzb_max_file_size"`
	SegmentCacheSize       string                           `json:"segment_cache_size"`
	StreamBufferSize       string                           `json:"stream_buffer_size"`
	MaxConnectionPerStream int                              `json:"max_connection_per_stream"`
	IndexerRequestHeader   UsenetConfigIndexerRequestHeader `json:"indexer_request_header"`
}

func flattenHeader(h http.Header) map[string]string {
	flat := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			flat[k] = v[0]
		}
	}
	return flat
}

func handleGetUsenetConfig(w http.ResponseWriter, r *http.Request) {
	queryHeaders := make(map[string]map[string]string, len(config.Newz.IndexerRequestHeader.Query))
	for qt, h := range config.Newz.IndexerRequestHeader.Query {
		queryHeaders[string(qt)] = flattenHeader(h)
	}

	data := UsenetConfig{
		NZBCacheSize:           util.ToSize(config.NewzNZBCacheSize),
		NZBCacheTTL:            config.NewzNZBCacheTTL.String(),
		NZBMaxFileSize:         util.ToSize(config.NewzNZBMaxFileSize),
		SegmentCacheSize:       util.ToSize(config.NewzSegmentCacheSize),
		StreamBufferSize:       util.ToSize(config.NewzStreamBufferSize),
		MaxConnectionPerStream: config.NewzMaxConnectionPerStream,
		IndexerRequestHeader: UsenetConfigIndexerRequestHeader{
			Query: queryHeaders,
			Grab:  flattenHeader(config.Newz.IndexerRequestHeader.Grab),
		},
	}
	SendData(w, r, 200, data)
}

func AddUsenetConfigEndpoints(router *http.ServeMux) {
	authed := EnsureAuthed

	router.HandleFunc("/usenet/config", authed(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetUsenetConfig(w, r)
		default:
			ErrorMethodNotAllowed(r).Send(w, r)
		}
	}))
}
