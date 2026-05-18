package dash_api

import (
	"net/http"
	"strconv"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anidb"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/imdb_title"
	"github.com/MunifTanjim/stremthru/internal/imdb_torrent"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_mapping_review"
	"github.com/MunifTanjim/stremthru/internal/worker"
)

var mappingReviewPeer = func() *peer.APIClient {
	if !config.HasPeer || config.PeerAuthToken == "" {
		return nil
	}
	return peer.NewAPIClient(&peer.APIClientConfig{
		BaseURL: config.PeerURL,
		APIKey:  config.PeerAuthToken,
	})
}()

type MappingReviewFileRequest struct {
	Path        string `json:"path"`
	PrevSeason  int    `json:"prev_season"`
	Season      int    `json:"season"`
	PrevEpisode int    `json:"prev_episode"`
	Episode     int    `json:"episode"`
}

type SuggestedMappingRequest struct {
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}

type MappingReviewRequest struct {
	Hash              string                               `json:"hash"`
	Target            torrent_mapping_review.MappingTarget `json:"target"`
	Reason            torrent_mapping_review.ReviewReason  `json:"reason"`
	PrevId            string                               `json:"prev_id"`
	MappingId         string                               `json:"mapping_id"`
	Files             []MappingReviewFileRequest           `json:"files"`
	SuggestedMappings []SuggestedMappingRequest            `json:"suggested_mappings"`
	Comment           string                               `json:"comment"`
}

type MappingReviewWithTitles struct {
	torrent_mapping_review.MappingReview
	HashTitle       string   `json:"hash_title,omitempty"`
	PrevIdTitles    []string `json:"prev_id_titles,omitempty"`
	MappingIdTitles []string `json:"mapping_id_titles,omitempty"`
}

type ListTorrentReviewRequestsResponse struct {
	Items      []MappingReviewWithTitles `json:"items"`
	NextCursor string                    `json:"next_cursor"`
}

type PreviewMappingRequest struct {
	AniDBId string `json:"anidb_id"`
}

type PreviewMappingResponse struct {
	Mappings    []AniDBMappingPreview `json:"mappings"`
	AniDBTitles []string              `json:"anidb_titles"`
}

type AniDBMappingPreview struct {
	TId     string `json:"tid"`
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}

func collectReviewHashes(items []torrent_mapping_review.MappingReview) []string {
	seen := make(map[string]struct{})
	hashes := []string{}
	for _, item := range items {
		if item.Hash != "" {
			if _, ok := seen[item.Hash]; !ok {
				seen[item.Hash] = struct{}{}
				hashes = append(hashes, item.Hash)
			}
		}
	}
	return hashes
}

func collectReviewIdsByTarget(items []torrent_mapping_review.MappingReview) (imdbIds, anidbIds []string) {
	seenImdb := make(map[string]struct{})
	seenAnidb := make(map[string]struct{})

	for _, item := range items {
		ids := []string{item.PrevId, item.MappingId}
		for _, id := range ids {
			if id == "" {
				continue
			}
			switch item.Target {
			case torrent_mapping_review.MappingTargetIMDB:
				if _, ok := seenImdb[id]; !ok {
					seenImdb[id] = struct{}{}
					imdbIds = append(imdbIds, id)
				}
			case torrent_mapping_review.MappingTargetAniDB:
				if _, ok := seenAnidb[id]; !ok {
					seenAnidb[id] = struct{}{}
					anidbIds = append(anidbIds, id)
				}
			}
		}
	}
	return imdbIds, anidbIds
}

func buildIMDBTitleMap(titles []imdb_title.IMDBTitle) map[string]string {
	m := make(map[string]string)
	for _, t := range titles {
		m[t.TId] = t.Title
	}
	return m
}

func buildAniDBTitleMap(titles anidb.AniDBTitles) map[string][]string {
	m := make(map[string][]string)
	for _, t := range titles {
		m[t.TId] = append(m[t.TId], t.Value)
	}
	return m
}

func getTitlesForId(target torrent_mapping_review.MappingTarget, id string, imdbMap map[string]string, anidbMap map[string][]string) []string {
	if id == "" {
		return nil
	}
	switch target {
	case torrent_mapping_review.MappingTargetIMDB:
		if title, ok := imdbMap[id]; ok {
			return []string{title}
		}
	case torrent_mapping_review.MappingTargetAniDB:
		if titles, ok := anidbMap[id]; ok {
			return titles
		}
	}
	return nil
}

func handleListTorrentReviewRequests(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	target := q.Get("target")
	cursor := q.Get("cursor")
	limitStr := q.Get("limit")

	limit := 20
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	result, err := torrent_mapping_review.List(torrent_mapping_review.ListParams{
		Status: status,
		Target: target,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		SendError(w, r, err)
		return
	}

	rawItems := result.Items
	if rawItems == nil {
		rawItems = []torrent_mapping_review.MappingReview{}
	}

	// Collect unique identifiers
	hashes := collectReviewHashes(rawItems)
	imdbIds, anidbIds := collectReviewIdsByTarget(rawItems)

	// Batch fetch titles (errors logged, not fatal)
	log := GetReqCtx(r).Log

	hashTitles := make(map[string]torrent_info.TorrentInfo)
	if len(hashes) > 0 {
		if titles, err := torrent_info.GetByHashes(hashes); err != nil {
			log.Warn("failed to fetch torrent info for hashes", "error", err)
		} else {
			hashTitles = titles
		}
	}

	imdbTitleMap := make(map[string]string)
	if len(imdbIds) > 0 {
		if titles, err := imdb_title.ListByIds(imdbIds); err != nil {
			log.Warn("failed to fetch imdb titles", "error", err)
		} else {
			imdbTitleMap = buildIMDBTitleMap(titles)
		}
	}

	anidbTitleMap := make(map[string][]string)
	if len(anidbIds) > 0 {
		if titles, err := anidb.GetTitlesByIds(anidbIds); err != nil {
			log.Warn("failed to fetch anidb titles", "error", err)
		} else {
			anidbTitleMap = buildAniDBTitleMap(titles)
		}
	}

	// Merge titles into response
	items := make([]MappingReviewWithTitles, len(rawItems))
	for i, review := range rawItems {
		items[i] = MappingReviewWithTitles{
			MappingReview:   review,
			PrevIdTitles:    getTitlesForId(review.Target, review.PrevId, imdbTitleMap, anidbTitleMap),
			MappingIdTitles: getTitlesForId(review.Target, review.MappingId, imdbTitleMap, anidbTitleMap),
		}
		if info, ok := hashTitles[review.Hash]; ok {
			items[i].HashTitle = info.TorrentTitle
		}
	}

	SendData(w, r, 200, ListTorrentReviewRequestsResponse{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

type AniDBMappingInput struct {
	TId     string `json:"tid"`
	SType   string `json:"s_type"`
	S       int    `json:"s"`
	EpStart int    `json:"ep_start"`
	EpEnd   int    `json:"ep_end"`
}

type ResolveTorrentReviewPayload struct {
	MappingId string              `json:"mapping_id,omitempty"`
	Mappings  []AniDBMappingInput `json:"mappings,omitempty"`
}

func handleResolveTorrentReviewRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	review, err := torrent_mapping_review.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if review == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	payload := &ResolveTorrentReviewPayload{}
	if err := ReadRequestBodyJSON(r, payload); err != nil {
		SendError(w, r, err)
		return
	}

	mappingId := payload.MappingId
	if mappingId == "" {
		mappingId = review.MappingId
	}

	log := GetReqCtx(r).Log

	switch review.Target {
	case torrent_mapping_review.MappingTargetIMDB:
		if err := imdb_torrent.Delete([]imdb_torrent.IMDBTorrent{{TId: review.PrevId, Hash: review.Hash}}); err != nil {
			SendError(w, r, err)
			return
		}
		items := []imdb_torrent.IMDBTorrent{
			{Hash: review.Hash, TId: mappingId},
		}
		if err := imdb_torrent.Insert(items); err != nil {
			SendError(w, r, err)
			return
		}
	case torrent_mapping_review.MappingTargetAniDB:
		if err := anidb.DeleteTorrentByTidAndHash(review.PrevId, review.Hash); err != nil {
			SendError(w, r, err)
			return
		}

		// For AniDB: require either mappings or mapping_id
		if review.Target == torrent_mapping_review.MappingTargetAniDB && len(payload.Mappings) == 0 && mappingId == "" {
			ErrorBadRequest(r).WithMessage("either mappings array or mapping_id required").Send(w, r)
			return
		}

		var items []anidb.AniDBTorrent

		if len(payload.Mappings) > 0 {
			// Use provided mappings from preview flow
			for _, m := range payload.Mappings {
				if m.EpStart < 0 || m.EpEnd < 0 || m.EpStart > m.EpEnd {
					ErrorBadRequest(r).WithMessage("invalid episode range").Send(w, r)
					return
				}
				eps := make([]int, 0, m.EpEnd-m.EpStart+1)
				for i := m.EpStart; i <= m.EpEnd; i++ {
					eps = append(eps, i)
				}
				items = append(items, anidb.AniDBTorrent{
					TId:          m.TId,
					Hash:         review.Hash,
					SeasonType:   anidb.TorrentSeasonType(m.SType),
					Season:       m.S,
					EpisodeStart: m.EpStart,
					EpisodeEnd:   m.EpEnd,
					Episodes:     eps,
				})
			}
		} else if mappingId != "" {
			// Legacy flow: auto-map with forced ID
			tInfo, err := torrent_info.GetByHash(review.Hash)
			if err != nil {
				SendError(w, r, err)
				return
			}
			if tInfo == nil {
				ErrorNotFound(r).WithMessage("torrent info not found").Send(w, r)
				return
			}

			items = worker.MapTorrentToAniDB(review.Hash, *tInfo, func(msg string, e error, args ...any) {
				if e != nil {
					log.Error(msg, append([]any{"error", e}, args...)...)
				} else {
					log.Debug(msg, args...)
				}
			})

			for i := range items {
				items[i].TId = mappingId
			}

			if len(items) == 0 {
				items = []anidb.AniDBTorrent{{Hash: review.Hash, TId: mappingId}}
			}
		}

		if len(items) > 0 {
			if err := anidb.UpsertTorrents(items); err != nil {
				SendError(w, r, err)
				return
			}
		}
	}

	if err := torrent_mapping_review.Resolve(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, struct{}{})
}

func handlePreviewTorrentReviewMapping(w http.ResponseWriter, r *http.Request) {
	log := GetReqCtx(r).Log

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	review, err := torrent_mapping_review.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if review == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if review.Status != torrent_mapping_review.ReviewStatusPending {
		ErrorBadRequest(r).WithMessage("review is not pending").Send(w, r)
		return
	}

	if review.Target != torrent_mapping_review.MappingTargetAniDB {
		ErrorBadRequest(r).WithMessage("preview only supported for anidb").Send(w, r)
		return
	}

	payload := &PreviewMappingRequest{}
	if err := ReadRequestBodyJSON(r, payload); err != nil {
		SendError(w, r, err)
		return
	}

	if payload.AniDBId == "" {
		ErrorBadRequest(r).WithMessage("anidb_id required").Send(w, r)
		return
	}

	tInfo, err := torrent_info.GetByHash(review.Hash)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if tInfo == nil {
		ErrorNotFound(r).WithMessage("torrent info not found").Send(w, r)
		return
	}

	items := worker.MapTorrentToAniDB(review.Hash, *tInfo, func(msg string, e error, args ...any) {
		if e != nil {
			log.Error(msg, append([]any{"error", e}, args...)...)
		} else {
			log.Debug(msg, args...)
		}
	})

	mappings := make([]AniDBMappingPreview, len(items))
	for i, item := range items {
		mappings[i] = AniDBMappingPreview{
			TId:     payload.AniDBId,
			SType:   string(item.SeasonType),
			S:       item.Season,
			EpStart: item.EpisodeStart,
			EpEnd:   item.EpisodeEnd,
		}
	}

	titles := []string{}
	anidbTitles, err := anidb.GetTitlesByIds([]string{payload.AniDBId})
	if err == nil {
		for _, t := range anidbTitles {
			titles = append(titles, t.Value)
		}
	}

	SendData(w, r, 200, PreviewMappingResponse{
		Mappings:    mappings,
		AniDBTitles: titles,
	})
}

func handleRejectTorrentReviewRequest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	review, err := torrent_mapping_review.GetById(id)
	if err != nil {
		SendError(w, r, err)
		return
	}
	if review == nil {
		ErrorNotFound(r).Send(w, r)
		return
	}

	if review.Status != torrent_mapping_review.ReviewStatusPending {
		ErrorBadRequest(r).WithMessage("review is not pending").Send(w, r)
		return
	}

	// Delete existing mappings based on target
	if review.Target == torrent_mapping_review.MappingTargetIMDB {
		if review.PrevId != "" {
			if err := imdb_torrent.Delete([]imdb_torrent.IMDBTorrent{{TId: review.PrevId, Hash: review.Hash}}); err != nil {
				SendError(w, r, err)
				return
			}
		}
	} else {
		if review.PrevId != "" {
			if err := anidb.DeleteTorrentByTidAndHash(review.PrevId, review.Hash); err != nil {
				SendError(w, r, err)
				return
			}
		}
	}

	if err := torrent_mapping_review.Reject(id); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, struct{}{})
}

func handleCreateReviewRequest(w http.ResponseWriter, r *http.Request) {
	payload := &MappingReviewRequest{}
	if err := ReadRequestBodyJSON(r, payload); err != nil {
		SendError(w, r, err)
		return
	}

	if payload.Hash == "" || payload.Target == "" || payload.Reason == "" {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	clientIP := GetReqCtx(r).ClientIP

	files := make([]torrent_mapping_review.FileCorrection, len(payload.Files))
	for i, f := range payload.Files {
		files[i] = torrent_mapping_review.FileCorrection{
			Path:        f.Path,
			PrevSeason:  f.PrevSeason,
			Season:      f.Season,
			PrevEpisode: f.PrevEpisode,
			Episode:     f.Episode,
		}
	}

	suggestedMappings := make([]torrent_mapping_review.SuggestedMapping, len(payload.SuggestedMappings))
	for i, sm := range payload.SuggestedMappings {
		suggestedMappings[i] = torrent_mapping_review.SuggestedMapping{
			SType:   sm.SType,
			S:       sm.S,
			EpStart: sm.EpStart,
			EpEnd:   sm.EpEnd,
		}
	}

	item := torrent_mapping_review.MappingReview{
		Hash:              payload.Hash,
		Target:            payload.Target,
		Reason:            payload.Reason,
		PrevId:            payload.PrevId,
		MappingId:         payload.MappingId,
		Files:             files,
		SuggestedMappings: suggestedMappings,
		Comment:           payload.Comment,
		IP:                clientIP,
	}

	if err := torrent_mapping_review.Insert(item); err != nil {
		SendError(w, r, err)
		return
	}

	if mappingReviewPeer != nil {
		log := GetReqCtx(r).Log
		if _, err := mappingReviewPeer.ForwardMappingReview(&peer.ForwardMappingReviewParams{
			Hash:      item.Hash,
			Target:    item.Target,
			Reason:    item.Reason,
			PrevId:    item.PrevId,
			MappingId: item.MappingId,
			Files:     item.Files,
			Comment:   item.Comment,
			IP:        item.IP,
		}); err != nil {
			log.Error("failed to forward mapping review to peer", "error", core.PackError(err))
		}
	}

	SendData(w, r, 201, struct{}{})
}

func AddTorrentReviewRequestsEndpoint(router *http.ServeMux) {
	authed := EnsureAuthed
	router.HandleFunc("GET /torrent/review-requests", authed(handleListTorrentReviewRequests))
	router.HandleFunc("POST /torrent/review-requests", authed(handleCreateReviewRequest))
	router.HandleFunc("PATCH /torrent/review-requests/{id}/resolve", authed(handleResolveTorrentReviewRequest))
	router.HandleFunc("POST /torrent/review-requests/{id}/preview", authed(handlePreviewTorrentReviewMapping))
	router.HandleFunc("POST /torrent/review-requests/{id}/reject", authed(handleRejectTorrentReviewRequest))
}
