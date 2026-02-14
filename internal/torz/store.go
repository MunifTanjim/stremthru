package torz

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	store_util "github.com/MunifTanjim/stremthru/internal/store/util"
	"github.com/MunifTanjim/stremthru/store"
)

func handleStoreTorzCheck(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)

	queryParams := r.URL.Query()
	magnet, ok := queryParams["magnet"]
	if !ok {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypeQuery,
			Location:     "magnet",
			Message:      "missing magnet",
		}).Send(w, r)
		return
	}

	magnets := []string{}
	for _, m := range magnet {
		magnets = append(magnets, strings.FieldsFunc(m, func(r rune) bool {
			return r == ','
		})...)
	}

	rCtx := server.GetReqCtx(r)
	rCtx.ReqQuery.Set("magnet", "..."+strconv.Itoa(len(magnets))+" items...")

	if len(magnets) == 0 {
		server.ErrorBadRequest(r).WithMessage("missing magnet").Send(w, r)
		return
	}

	if len(magnets) > 500 {
		server.ErrorBadRequest(r).WithMessage("too many magnets, max allowed 500").Send(w, r)
		return
	}

	params := &store.CheckMagnetParams{
		Magnets: magnets,
	}
	params.APIKey = ctx.StoreAuthToken
	if ctx.ClientIP != "" {
		params.ClientIP = ctx.ClientIP
	}
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

func addTorz(ctx *storecontext.Context, link string, torrent *multipart.FileHeader) (*store.AddMagnetData, error) {
	params := &store.AddMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Magnet = link
	if ctx.ClientIP != "" {
		params.ClientIP = ctx.ClientIP
	}
	if torrent != nil {
		params.Torrent = torrent
		if _, _, err := params.GetTorrentMeta(); err != nil {
			return nil, server.ErrorBadRequest(nil).WithMessage("invalid torrent file").WithCause(err)
		}
	}
	data, err := ctx.Store.AddMagnet(params)
	if err == nil {
		buddy.TrackMagnet(ctx.Store, data.Hash, data.Name, data.Size, data.Private, data.Files, "", data.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)
	}
	return data, err
}

func handleStoreTorzAdd(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)

	var data *store.AddMagnetData
	var err error
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		payload := &AddTorzPayload{}
		if err := server.ReadRequestBodyJSON(r, payload); err != nil {
			server.SendError(w, r, err)
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
			data, err = addTorz(ctx, payload.Link, nil)
		} else {
			fileHeader, fetchErr := shared.FetchTorrentFile(payload.Link, 1<<20)
			if fetchErr != nil {
				server.ErrorBadRequest(r).Append(server.Error{
					LocationType: server.LocationTypeBody,
					Location:     "link",
					Message:      "unable to fetch torrent file",
				}).Send(w, r)
				return
			}
			data, err = addTorz(ctx, "", fileHeader)
		}

	case strings.Contains(contentType, "multipart/form-data"):
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseMultipartForm(512 << 10); err != nil {
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

		data, err = addTorz(ctx, "", fileHeader)

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
		limit = 500
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
