package dash_api

import (
	"encoding/json"
	"net/http"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/torrent_mapping_review"
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

type MappingReviewRequest struct {
	Hash      string                     `json:"hash"`
	Target    torrent_mapping_review.MappingTarget `json:"target"`
	Reason    torrent_mapping_review.ReviewReason  `json:"reason"`
	PrevId    string                     `json:"prev_id"`
	MappingId string                     `json:"mapping_id"`
	Files     []MappingReviewFileRequest `json:"files"`
	Comment   string                     `json:"comment"`
}

func handleTorrentMappingReview(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	var req MappingReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	if req.Hash == "" || req.Target == "" || req.Reason == "" {
		ErrorBadRequest(r).Send(w, r)
		return
	}

	clientIP := GetReqCtx(r).ClientIP

	files := make([]torrent_mapping_review.FileCorrection, len(req.Files))
	for i, f := range req.Files {
		files[i] = torrent_mapping_review.FileCorrection{
			Path:        f.Path,
			PrevSeason:  f.PrevSeason,
			Season:      f.Season,
			PrevEpisode: f.PrevEpisode,
			Episode:     f.Episode,
		}
	}

	item := torrent_mapping_review.MappingReview{
		Hash:      req.Hash,
		Target:    req.Target,
		Reason:    req.Reason,
		PrevId:    req.PrevId,
		MappingId: req.MappingId,
		Files:     files,
		Comment:   req.Comment,
		IP:        clientIP,
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

func AddTorrentMappingReviewEndpoint(router *http.ServeMux) {
	authed := EnsureAuthed
	router.HandleFunc("/torrent/mapping/review", authed(handleTorrentMappingReview))
}
