package dash_api

import (
	"encoding/json"
	"maps"
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

type ReprocessItem struct {
	TId  string `json:"tid"`
	Hash string `json:"hash"`
}

type ReprocessRequest struct {
	Items   []ReprocessItem `json:"items"`
	Targets []string        `json:"targets"`
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

	if len(req.Items) == 0 {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	targets := req.Targets
	if len(targets) == 0 {
		targets = []string{"imdb", "anidb"}
	}

	includeIMDB := slices.Contains(targets, "imdb")
	includeAniDB := slices.Contains(targets, "anidb")

	hashSet := util.NewSet[string]()
	for _, item := range req.Items {
		hashSet.Add(item.Hash)
	}
	uniqueHashes := hashSet.ToSlice()

	tInfoByHash, err := torrent_info.GetByHashes(uniqueHashes)
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
		imdbPairs := make([]imdb_torrent.IMDBTorrent, len(req.Items))
		for i, item := range req.Items {
			imdbPairs[i] = imdb_torrent.IMDBTorrent{TId: item.TId, Hash: item.Hash}
		}
		if err := imdb_torrent.Delete(imdbPairs); err != nil {
			SendError(w, r, err)
			return
		}
	}
	if includeAniDB {
		anidbPairs := make([]anidb.AniDBTorrent, len(req.Items))
		for i, item := range req.Items {
			anidbPairs[i] = anidb.AniDBTorrent{TId: item.TId, Hash: item.Hash}
		}
		if err := anidb.DeleteTorrentsByTidAndHashPairs(anidbPairs); err != nil {
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
		tInfoToRemap := maps.Clone(tInfoByHash)
		existingIMDB, err := imdb_torrent.GetByHashes(hashes)
		if err != nil {
			SendError(w, r, err)
			return
		}
		for _, item := range existingIMDB {
			if item.TId != "" {
				delete(tInfoToRemap, item.Hash)
			}
		}
		imdbMapped, err := mapTorrentsToIMDB(tInfoToRemap, log)
		if err != nil {
			SendError(w, r, err)
			return
		}
		mapped["imdb"] = imdbMapped
	}

	if includeAniDB {
		tInfoToRemap := maps.Clone(tInfoByHash)
		existingAniDB, err := anidb.GetTorrentsByHashes(hashes)
		if err != nil {
			SendError(w, r, err)
			return
		}
		for _, item := range existingAniDB {
			if item.TId != "" {
				delete(tInfoToRemap, item.Hash)
			}
		}
		anidbMapped, err := mapTorrentsToAniDB(tInfoToRemap)
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
