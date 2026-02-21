package newz

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	usenetmanager "github.com/MunifTanjim/stremthru/internal/usenet/manager"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb"
	"github.com/MunifTanjim/stremthru/internal/usenet/nzb_info"
	usenet_pool "github.com/MunifTanjim/stremthru/internal/usenet/pool"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/stremthru"
)

func handleStoreNewzCheck(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

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

	params := &store.CheckNewzParams{
		Hashes: hashes,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := newzStore.CheckNewz(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 200, data)
}

type AddNewzPayload struct {
	Link string `json:"link"`
}

func addNewz(ctx *storecontext.Context, newzStore store.NewzStore, link string, file *multipart.FileHeader) (*store.AddNewzData, error) {
	params := &store.AddNewzParams{
		Link:     link,
		File:     file,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	return newzStore.AddNewz(params)
}

func handleStoreNewzAdd(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

	var data *store.AddNewzData
	var err error
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		payload := &AddNewzPayload{}
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

		data, err = addNewz(ctx, newzStore, payload.Link, nil)

	case strings.Contains(contentType, "multipart/form-data"):
		r.Body = http.MaxBytesReader(w, r.Body, config.Newz.NZBFileMaxSize)
		if err := r.ParseMultipartForm(util.ToBytes("10MB")); err != nil {
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
					Message:      "missing nzb file",
				}).Send(w, r)
				return
			}
			if len(fileHeaders) > 1 {
				server.ErrorBadRequest(r).Append(server.Error{
					LocationType: server.LocationTypeBody,
					Location:     "file",
					Message:      "multiple nzb files provided, only one allowed",
				}).Send(w, r)
				return
			}
			fileHeader = fileHeaders[0]
		}

		data, err = addNewz(ctx, newzStore, "", fileHeader)

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

func handleStoreNewzList(w http.ResponseWriter, r *http.Request) {
	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

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

	params := &store.ListNewzParams{
		Limit:    limit,
		Offset:   offset,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := newzStore.ListNewz(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 200, data)
}

func handleStoreNewzGet(w http.ResponseWriter, r *http.Request) {
	newzId := r.PathValue("newzId")
	if newzId == "" {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypePath,
			Location:     "newzId",
			Message:      "missing newz id",
		}).Send(w, r)
		return
	}

	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

	params := &store.GetNewzParams{
		Id:       newzId,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := newzStore.GetNewz(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 200, data)
}

func handleStoreNewzRemove(w http.ResponseWriter, r *http.Request) {
	newzId := r.PathValue("newzId")
	if newzId == "" {
		server.ErrorBadRequest(r).Append(server.Error{
			LocationType: server.LocationTypePath,
			Location:     "newzId",
			Message:      "missing newz id",
		}).Send(w, r)
		return
	}

	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

	params := &store.RemoveNewzParams{
		Id: newzId,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := newzStore.RemoveNewz(params)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	server.SendData(w, r, 200, data)
}

type GenerateNewzLinkPayload struct {
	Link string `json:"link"`
}

func handleStoreNewzLinkGenerate(w http.ResponseWriter, r *http.Request) {
	payload := &GenerateNewzLinkPayload{}
	if err := server.ReadRequestBodyJSON(r, payload); err != nil {
		server.SendError(w, r, err)
		return
	}

	ctx := storecontext.Get(r)
	newzStore := ctx.Store.(store.NewzStore)

	params := &store.GenerateNewzLinkParams{
		Link:     payload.Link,
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := newzStore.GenerateNewzLink(params)
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

func handleStoreNewzStreamFile(w http.ResponseWriter, r *http.Request) {
	ctx := server.GetReqCtx(r)
	ctx.RedactURLPathValues(r, "token")

	token := r.PathValue("token")

	_, id, path, err := stremthru.UnwrapNewzStreamToken(token)
	if err != nil {
		server.SendError(w, r, err)
		return
	}

	nzbInfo, err := nzb_info.GetByHash(id)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	nzbFile, err := nzb_info.FetchNZBFile(nzbInfo.URL, nzbInfo.Name, ctx.Log)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	nzbDoc, err := nzb.ParseBytes(nzbFile.Blob)
	if err != nil {
		server.SendError(w, r, err)
		return
	}

	pool, err := usenetmanager.GetPool()
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	if pool == nil {
		server.ErrorBadRequest(r).WithMessage("no NNTP providers configured").Send(w, r)
		return
	}

	streamConfig := &usenet_pool.StreamConfig{
		Password: nzbInfo.Password,
	}
	stream, err := pool.StreamByContentPath(r.Context(), nzbDoc, strings.Split(path, "::"), streamConfig)
	if err != nil {
		server.SendError(w, r, err)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", stream.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, stream.Name, nzbFile.Mod, stream)
}
