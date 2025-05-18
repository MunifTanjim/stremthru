package stremio_torz

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var streamTemplate = stremio_transformer.StreamTemplateDefault

type wrappedStream struct {
	stremio.Stream
	r    *stremio_transformer.StreamExtractorResult
	hash string
}

func (s wrappedStream) IsSortable() bool {
	return s.r != nil
}

func (s wrappedStream) GetQuality() string {
	return s.r.Quality
}

func (s wrappedStream) GetResolution() string {
	return s.r.Resolution
}

func (s wrappedStream) GetSize() string {
	return s.r.Size
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	ctx, err := ud.GetRequestContext(r)
	if err != nil {
		shared.ErrorBadRequest(r, "failed to get request context: "+err.Error()).Send(w, r)
		return
	}

	contentType := r.PathValue("contentType")
	id := stremio_shared.GetPathValue(r, "id")

	isImdbId := strings.HasPrefix(id, "tt")
	if isImdbId {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w, r)
		return
	}

	eud := ud.GetEncoded()

	buddy.PullTorrentsByStremId(id, "")

	hashes, err := torrent_info.ListHashesByStremId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	magnetByHash := map[string]core.MagnetLink{}
	for _, hash := range hashes {
		magnet, err := core.ParseMagnetLink(hash)
		if err != nil {
			continue
		}
		hashes = append(hashes, magnet.Hash)
		magnetByHash[magnet.Hash] = magnet
	}

	isCachedByHash := map[string]string{}
	if len(hashes) > 0 {
		cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
			Magnets:  hashes,
			ClientIP: ctx.ClientIP,
			SId:      id,
		}, log)
		if cmRes.HasErr {
			SendError(w, r, errors.Join(cmRes.Err...))
			return
		}
		isCachedByHash = cmRes.ByHash
	}

	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/torz", eud, "_/strem", id)

	tInfoByHash, err := torrent_info.GetByHashes(hashes)
	if err != nil {
		SendError(w, r, err)
		return
	}

	filesByHashes, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		SendError(w, r, err)
		return
	}

	hashSeen := map[string]struct{}{}
	wrappedStreams := []wrappedStream{}
	for _, hash := range hashes {
		if _, seen := hashSeen[hash]; seen {
			continue
		}

		hashSeen[hash] = struct{}{}

		tInfo, ok := tInfoByHash[hash]
		if !ok {
			continue
		}

		var file *torrent_stream.File
		if files, ok := filesByHashes[hash]; ok {
			for i := range files {
				f := &files[i]
				if f.SId == id {
					file = f
				}
			}
		}
		fName := ""
		fIdx := -1
		fSize := int64(0)
		if file != nil {
			fIdx = file.Idx
			fName = file.Name
			if file.Size > 0 {
				fSize = file.Size
			}
		} else if core.HasVideoExtension(tInfo.TorrentTitle) {
			fName = tInfo.TorrentTitle
		}

		pttr, err := tInfo.ToParsedResult()
		if err != nil {
			SendError(w, r, err)
			return
		}
		data := &stremio_transformer.StreamExtractorResult{
			Hash:   tInfo.Hash,
			TTitle: tInfo.TorrentTitle,
			Result: pttr,
			Addon: stremio_transformer.StreamExtractorResultAddon{
				Name: "Torz",
			},
			Category: contentType,
			File: stremio_transformer.StreamExtractorResultFile{
				Name: fName,
				Idx:  fIdx,
			},
		}
		if fSize > 0 {
			data.File.Size = util.ToSize(fSize)
		}
		wrappedStreams = append(wrappedStreams, wrappedStream{
			hash: hash,
			Stream: stremio.Stream{
				BehaviorHints: &stremio.StreamBehaviorHints{
					Filename:   data.File.Name,
					VideoSize:  fSize,
					BingeGroup: "torz:" + data.Hash,
				},
			},
			r: data,
		})
	}

	stremio_transformer.SortStreams(wrappedStreams, "")

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}
	for _, wStream := range wrappedStreams {
		if storeCode, isCached := isCachedByHash[wStream.hash]; isCached && storeCode != "" {
			storeName := store.StoreCode(strings.ToLower(storeCode)).Name()
			wStream.r.Store.Code = storeCode
			wStream.r.Store.Name = string(storeName)
			wStream.r.Store.IsCached = true
			wStream.r.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
			stream, err := streamTemplate.Execute(&wStream.Stream, wStream.r)
			if err != nil {
				SendError(w, r, err)
				return
			}
			steamUrl := streamBaseUrl.JoinPath(strings.ToLower(storeCode), wStream.hash, strconv.Itoa(wStream.r.File.Idx), "/")
			if wStream.r.File.Name != "" {
				steamUrl = steamUrl.JoinPath(wStream.r.File.Name)
			}
			stream.URL = steamUrl.String()
			cachedStreams = append(cachedStreams, *stream)
		} else {
			stores := ud.GetStores()
			for i := range stores {
				s := &stores[i]
				storeName := s.Store.GetName()
				storeCode := storeName.Code()
				if storeCode == store.StoreCodeEasyDebrid {
					continue
				}

				origStream := wStream.Stream
				wStream.r.Store.Code = strings.ToUpper(string(storeCode))
				wStream.r.Store.Name = string(storeName)
				wStream.r.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
				stream, err := streamTemplate.Execute(&origStream, wStream.r)
				if err != nil {
					SendError(w, r, err)
					return
				}

				steamUrl := streamBaseUrl.JoinPath(string(storeCode), wStream.hash, strconv.Itoa(wStream.r.File.Idx), "/")
				if wStream.r.File.Name != "" {
					steamUrl = steamUrl.JoinPath(wStream.r.File.Name)
				}
				stream.URL = steamUrl.String()
				uncachedStreams = append(uncachedStreams, *stream)
			}
		}
	}

	streams := make([]stremio.Stream, len(cachedStreams)+len(uncachedStreams))
	idx := 0
	for i := range cachedStreams {
		streams[idx] = cachedStreams[i]
		idx++
	}
	for i := range uncachedStreams {
		streams[idx] = uncachedStreams[i]
		idx++
	}

	SendResponse(w, r, 200, &stremio.StreamHandlerResponse{
		Streams: streams,
	})
}
