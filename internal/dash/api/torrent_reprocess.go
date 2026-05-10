package dash_api

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/worker"
)

type ReprocessRequest struct {
	Hashes  []string `json:"hashes"`
	Targets []string `json:"targets"`
}

type ReprocessResponse struct {
	Mode      string         `json:"mode"`
	Processed int            `json:"processed,omitempty"`
	Parsed    int            `json:"parsed,omitempty"`
	Mapped    map[string]int `json:"mapped,omitempty"`
	Queued    int            `json:"queued,omitempty"`
}

func mapTorrentsToIMDB(tInfos map[string]torrent_info.TorrentInfo, log *logger.Logger) (int, error) {
	items := []imdb_torrent.IMDBTorrent{}

	for hash, tInfo := range tInfos {
		result := worker.MapTorrentToIMDB(hash, tInfo, func(message string, err error, args ...any) {
			if err != nil {
				log.Error(message, append([]any{"error", err}, args...)...)
			} else {
				log.Debug(message, args...)
			}
		})
		if result == nil {
			continue
		}
		items = append(items, *result.Item)
	}

	if err := imdb_torrent.Insert(items); err != nil {
		return 0, err
	}

	mapped := 0
	for _, item := range items {
		if item.TId != "" {
			mapped++
		}
	}
	return mapped, nil
}

func mapTorrentsToAniDB(tInfos map[string]torrent_info.TorrentInfo) (int, error) {
	items := []anidb.AniDBTorrent{}

	for hash, tInfo := range tInfos {
		mapped := worker.MapTorrentToAniDB(hash, tInfo, nil)
		if mapped != nil {
			items = append(items, mapped...)
		}
	}

	if err := anidb.UpsertTorrents(items); err != nil {
		return 0, err
	}

	mappedCount := 0
	seenHashes := util.NewSet[string]()
	for _, item := range items {
		if item.TId != "" {
			if !seenHashes.Has(item.Hash) {
				seenHashes.Add(item.Hash)
				mappedCount++
			}
		}
	}
	return mappedCount, nil
}

func handleReprocessTorrents(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	var req ReprocessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	if len(req.Hashes) == 0 {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	targets := req.Targets
	if len(targets) == 0 {
		targets = []string{"imdb", "anidb"}
	}

	includeIMDB := slices.Contains(targets, "imdb")
	includeAniDB := slices.Contains(targets, "anidb")

	tInfoByHash, err := torrent_info.GetByHashes(req.Hashes)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if len(tInfoByHash) == 0 {
		SendData(w, r, 200, ReprocessResponse{
			Mode:      "sync",
			Processed: 0,
		})
		return
	}

	hashes := make([]string, 0, len(tInfoByHash))
	for hash := range tInfoByHash {
		hashes = append(hashes, hash)
	}

	tInfosToUpdate := make([]*torrent_info.TorrentInfo, 0, len(tInfoByHash))
	for hash, tInfo := range tInfoByHash {
		if err := tInfo.ForceParse(); err != nil {
			continue
		}
		tInfoByHash[hash] = tInfo
		tInfosToUpdate = append(tInfosToUpdate, &tInfo)
	}

	if err := torrent_info.UpsertParsed(tInfosToUpdate); err != nil {
		SendError(w, r, err)
		return
	}

	if includeIMDB {
		if err := imdb_torrent.DeleteByHashes(hashes); err != nil {
			SendError(w, r, err)
			return
		}
	}
	if includeAniDB {
		if err := anidb.DeleteTorrentsByHashes(hashes); err != nil {
			SendError(w, r, err)
			return
		}
	}

	parsed := len(tInfosToUpdate)

	if len(hashes) > 10 {
		SendData(w, r, 200, ReprocessResponse{
			Mode:   "async",
			Parsed: parsed,
			Queued: len(hashes),
		})
		return
	}

	mapped := map[string]int{}

	if includeIMDB {
		imdbMapped, err := mapTorrentsToIMDB(tInfoByHash, log)
		if err != nil {
			SendError(w, r, err)
			return
		}
		mapped["imdb"] = imdbMapped
	}

	if includeAniDB {
		anidbMapped, err := mapTorrentsToAniDB(tInfoByHash)
		if err != nil {
			SendError(w, r, err)
			return
		}
		mapped["anidb"] = anidbMapped
	}

	SendData(w, r, 200, ReprocessResponse{
		Mode:      "sync",
		Parsed:    parsed,
		Processed: len(hashes),
		Mapped:    mapped,
	})
}

func AddTorrentReprocessEndpoint(router *http.ServeMux) {
	authed := EnsureAuthed
	router.HandleFunc("/torrents/reprocess", authed(handleReprocessTorrents))
}
