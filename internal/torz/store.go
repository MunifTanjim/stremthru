package torz

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/peer_token"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	store_util "github.com/MunifTanjim/stremthru/internal/store/util"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
)

func handleStoreTorzCheck(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)

	queryParams := r.URL.Query()
	hash, ok := queryParams["hash"]
	if !ok {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypeQuery,
			Location:     "hash",
			Message:      "missing hash",
		}).Send(w, r)
		return
	}

	hashes := []string{}
	for _, h := range hash {
		hashes = append(hashes, strings.FieldsFunc(h, func(r rune) bool {
			return r == ','
		})...)
	}

	rCtx := server.GetReqCtx(r)
	rCtx.ReqQuery.Set("hash", "..."+strconv.Itoa(len(hashes))+" items...")

	if len(hashes) == 0 {
		server.ErrorBadRequest(r).WithMessage("missing hash").Send(w, r)
		return
	}

	if len(hashes) > 500 {
		server.ErrorBadRequest(r).WithMessage("too many hashes, max allowed 500").Send(w, r)
		return
	}

	sid := queryParams.Get("sid")

	params := &store.CheckMagnetParams{
		ClientIP:  ctx.ClientIP,
		Magnets:   hashes,
		SId:       sid,
		LocalOnly: queryParams.Get("local_only") != "",
	}
	params.IsTrustedRequest, _ = peer_token.IsValid(peer_token.ExtractFromRequest(r))
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.CheckMagnet(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	if data.Items == nil {
		data.Items = []store.CheckMagnetDataItem{}
	}
	server.SendData(w, r, 200, data)
}

type AddTorzPayload struct {
	Link string `json:"link"`
}

func addTorz(r *http.Request, ctx *storecontext.Context, link string, file *multipart.FileHeader) (*store.AddMagnetData, error) {
	params := &store.AddMagnetParams{
		ClientIP: ctx.ClientIP,
		Magnet:   link,
	}
	params.APIKey = ctx.StoreAuthToken
	if file != nil {
		params.Torrent = file
		if _, _, err := params.GetTorrentMeta(); err != nil {
			return nil, server.ErrorBadRequest(r).WithMessage("invalid torrent file").WithCause(err)
		}
	}
	data, err := ctx.Store.AddMagnet(params)
	if err != nil {
		return nil, err
	}
	buddy.TrackMagnet(ctx.Store, data.Hash, data.Name, data.Size, data.Private, data.Files, "", data.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)
	return data, nil
}

func handleStoreTorzAdd(w http.ResponseWriter, r *http.Request) {
	log := server.GetReqCtx(r).Log

	ctx := storecontext.Get(r)

	var data *store.AddMagnetData
	var err error
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		payload := &AddTorzPayload{}
		if err := server.ReadRequestBodyJSON(r, payload); err != nil {
			server.ErrorBadRequest(r).WithMessage("invalid payload").WithCause(err).Send(w, r)
			return
		}

		if payload.Link == "" {
			server.ErrorBadRequest(r).Append(server.Error{
				LocationType: server.LocationTypeBody,
				Location:     "link",
				Message:      "missing link",
			}).Send(w, r)
			return
		}

		if strings.HasPrefix(payload.Link, "magnet:") {
			data, err = addTorz(r, ctx, payload.Link, nil)
		} else {
			magnet, fileHeader, fetchErr := shared.FetchTorrentFile(payload.Link, "", log)
			if fetchErr != nil {
				server.ErrorBadRequest(r).Append(server.Error{
					LocationType: server.LocationTypeBody,
					Location:     "link",
					Message:      "unable to fetch torrent file",
				}).Send(w, r)
				return
			}
			if magnet != "" {
				data, err = addTorz(r, ctx, magnet, nil)
			} else {
				data, err = addTorz(r, ctx, "", fileHeader)
			}
		}

	case strings.Contains(contentType, "multipart/form-data"):
		r.Body = http.MaxBytesReader(w, r.Body, config.Torz.TorrentFileMaxSize)
		if err := r.ParseMultipartForm(util.ToBytes("512KB")); err != nil {
			server.SendError(w, r, err)
			return
		}

		var fileHeader *multipart.FileHeader
		if r.MultipartForm.File != nil {
			fileHeaders := r.MultipartForm.File["file"]
			if len(fileHeaders) == 0 {
				server.ErrorBadRequest(r).Append(server.Error{
					LocationType: server.LocationTypeBody,
					Location:     "file",
					Message:      "missing torrent file",
				}).Send(w, r)
				return
			}
			if len(fileHeaders) > 1 {
				server.ErrorBadRequest(r).Append(server.Error{
					LocationType: server.LocationTypeBody,
					Location:     "file",
					Message:      "multiple torrent files provided, only one allowed",
				}).Send(w, r)
				return
			}
			fileHeader = fileHeaders[0]
		}

		data, err = addTorz(r, ctx, "", fileHeader)

	default:
		server.ErrorUnsupportedMediaType(r).Send(w, r)
		return
	}

	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 201, data)
}

func handleStoreTorzList(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)

	queryParams := r.URL.Query()
	limit, err := shared.GetQueryInt(queryParams, "limit", 100)
	if err != nil {
		server.ErrorBadRequest(r).WithMessage(err.Error()).Send(w, r)
		return
	}
	if limit > 500 {
		server.ErrorBadRequest(r).WithMessage("limit cannot be greater than 500").Send(w, r)
		return
	}
	offset, err := shared.GetQueryInt(queryParams, "offset", 0)
	if err != nil {
		server.ErrorBadRequest(r).WithMessage(err.Error()).Send(w, r)
		return
	}

	params := &store.ListMagnetsParams{
		Limit:    limit,
		Offset:   offset,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.ListMagnets(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	if data.Items == nil {
		data.Items = []store.ListMagnetsDataItem{}
	}
	go store_util.RecordTorrentInfoFromListMagnets(ctx.Store.GetName().Code(), data.Items)
	server.SendData(w, r, 200, data)
}

func handleStoreTorzGet(w http.ResponseWriter, r *http.Request) {
	torzId := r.PathValue("torzId")
	if torzId == "" {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypePath,
			Location:     "torzId",
			Message:      "missing torz id",
		}).Send(w, r)
		return
	}

	ctx := storecontext.Get(r)

	params := &store.GetMagnetParams{
		Id:       torzId,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.GetMagnet(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	buddy.TrackMagnet(ctx.Store, data.Hash, data.Name, data.Size, data.Private, data.Files, "", data.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)
	server.SendData(w, r, 200, data)
}

func handleStoreTorzRemove(w http.ResponseWriter, r *http.Request) {
	torzId := r.PathValue("torzId")
	if torzId == "" {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypePath,
			Location:     "torzId",
			Message:      "missing torz id",
		}).Send(w, r)
		return
	}

	ctx := storecontext.Get(r)

	params := &store.RemoveMagnetParams{
		Id: torzId,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.RemoveMagnet(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 200, data)
}

type GenerateTorzLinkPayload struct {
	Link string `json:"link"`
}

func handleStoreTorzLinkGenerate(w http.ResponseWriter, r *http.Request) {
	payload := &GenerateTorzLinkPayload{}
	if err := server.ReadRequestBodyJSON(r, payload); err != nil {
		server.SendError(w, r, err)
		return
	}

	ctx := storecontext.Get(r)

	params := &store.GenerateLinkParams{
		Link:     payload.Link,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.GenerateLink(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}

	data.Link, err = shared.ProxyWrapLink(r, ctx, data.Link, "")
	if err != nil {
		server.SendError(w, r, err)
		return
	}

	server.SendData(w, r, 200, data)
}
