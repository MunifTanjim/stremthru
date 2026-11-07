package stremio_torz

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	torznab_indexer_syncinfo "github.com/MunifTanjim/stremthru/internal/torznab/indexer/syncinfo"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

// The complete pre-performance request path. This stays available when the
// opt-in performance feature is disabled: synchronous indexer/debrid checks,
// peer pull semantics, and per-file episode fallback are preserved.
var torzLazyPull = config.Stremio.Torz.LazyPull

func getStreamsForHashesLegacy(stremType, stremId string, hashes []string, nsid *torrent_stream.NormalizedStremId) ([]WrappedStream, error) {
	tInfoByHash, err := torrent_info.GetByHashes(hashes)
	if err != nil {
		return nil, err
	}

	filesByHashes, err := torrent_stream.GetFilesByHashes(hashes)
	if err != nil {
		return nil, err
	}

	wrappedStreams := make([]WrappedStream, 0, len(hashes))
	for _, hash := range hashes {
		tInfo, ok := tInfoByHash[hash]
		if !ok {
			continue
		}

		var file *torrent_stream.File
		if files, ok := filesByHashes[hash]; ok {
			idToMatch := stremId
			if nsid.IsAnime {
				if nsid.Id == "" {
					idToMatch = ""
				} else {
					idToMatch = nsid.Id + ":" + nsid.Episode
				}
			}
			if idToMatch != "" {
				for i := range files {
					f := &files[i]
					if f.IsVideo() {
						if f.SId == idToMatch || f.ASId == idToMatch {
							file = f
							break
						}
					}
				}
			}
			if file == nil && !nsid.IsAnime && nsid.IsSeries() && nsid.Episode != "" {
				for i := range files {
					f := &files[i]
					if f.IsVideo() {
						data := stremio_shared.ParseSeasonEpisodeFromName(f.Name, false)
						if data.Season != -1 && strconv.Itoa(data.Season) != nsid.Season {
							continue
						}
						if data.Episode == -1 || strconv.Itoa(data.Episode) != nsid.Episode {
							continue
						}
						file = f
						break
					}
				}
				if file == nil {
					log.Trace("skipping torrent: no matching file found", "hash", tInfo.Hash, "sid", stremId)
					continue
				}
			}
		}

		pttr, err := tInfo.ToParsedResult()
		if err != nil {
			return nil, err
		}
		data := &stremio_transformer.StreamExtractorResult{
			Hash:      tInfo.Hash,
			TTitle:    tInfo.TorrentTitle,
			Result:    pttr,
			Seeders:   tInfo.Seeders,
			IsPrivate: tInfo.Private,
			Indexer: stremio_transformer.StreamExtractorResultIndexer{
				ID:   tInfo.Indexer,
				Name: jackett.GetIndexerName(tInfo.Indexer),
			},
			Addon: stremio_transformer.StreamExtractorResultAddon{
				Name: "Torz",
			},
			Kind:     stremio_transformer.StreamExtractorResultKindTorz,
			Category: stremType,
			File: stremio_transformer.StreamExtractorResultFile{
				Idx: -1,
			},
		}

		fSize := int64(0)
		fVideoHash := ""
		if file != nil {
			data.File.Idx = file.Idx
			data.File.Name = file.Name
			if file.Size > 0 {
				fSize = file.Size
				data.File.Size = util.ToSize(fSize)
			}
			fVideoHash = file.VideoHash

			stremio_transformer.ApplyMediaInfo(data, file.MediaInfo)
		} else if core.HasVideoExtension(tInfo.TorrentTitle) {
			data.File.Name = tInfo.TorrentTitle
		}

		wrappedStreams = append(wrappedStreams, WrappedStream{
			R: data,
			Stream: &stremio.Stream{
				Name:        data.Addon.Name,
				Description: data.TTitle,
				InfoHash:    data.Hash,
				FileIndex:   data.File.Idx,
				BehaviorHints: &stremio.StreamBehaviorHints{
					Filename:   data.File.Name,
					VideoSize:  fSize,
					BingeGroup: "torz:" + data.Hash,
					VideoHash:  fVideoHash,
				},
			},
		})
	}
	return wrappedStreams, nil
}

func handleStreamLegacy(w http.ResponseWriter, r *http.Request) {
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
	isKitsuId := strings.HasPrefix(id, "kitsu:")
	isMALId := strings.HasPrefix(id, "mal:")
	isAnime := isKitsuId || isMALId

	if isImdbId {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else if isAnime {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) && contentType != "anime" {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w, r)
		return
	}

	filter, filter_err := ud.GetFilter()
	if filter_err != nil {
		log.Warn("failed to parse filter expression", "error", filter_err)
		shared.ErrorBadRequest(r, "invalid filter expression: "+filter_err.Error()).Send(w, r)
		return
	}

	eud := ud.GetEncoded()

	pulledHashes := []string{}
	nsid, err := torrent_stream.NormalizeStreamId(id)
	if err == nil {
		cleanSId := nsid.ToClean()
		if torzLazyPull {
			go buddy.PullTorrentsByStremId(cleanSId, "")
		} else {
			hashes := buddy.PullTorrentsByStremId(cleanSId, "")
			pulledHashes = append(pulledHashes, hashes...)
		}

		torznab_indexer_syncinfo.QueueJob(nsid.String())
	} else if !errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
		log.Error("failed to normalize strem id", "error", err, "id", id)
		shared.ErrorInternalServerError(r, "failed to normalize strem id").WithCause(err).Send(w, r)
		return
	} else {
		shared.ErrorBadRequest(r, "unsupported strem id: "+id).Send(w, r)
		return
	}

	hashes, err := torrent_info.ListHashesByStremId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	hashSet := util.NewSet[string]()
	for _, hash := range hashes {
		hashSet.Add(hash)
	}

	if len(pulledHashes) > 0 {
		for _, hash := range pulledHashes {
			if !hashSet.Has(hash) {
				hashSet.Add(hash)
				hashes = append(hashes, hash)
			}
		}
	}

	var wg sync.WaitGroup

	var wrappedStreams []WrappedStream
	var getStreamsError error
	wg.Go(func() {
		wrappedStreams, getStreamsError = getStreamsForHashesLegacy(contentType, id, hashes, nsid)
	})

	var wrappedStreamsFromIndexers []WrappedStream
	var hashesFromIndexers []string
	var getStreamsFromIndexersError error
	wg.Go(func() {
		timeoutCtx, cancel := context.WithTimeout(r.Context(), config.Stremio.Torz.IndexerMaxTimeout)
		defer cancel()
		wrappedStreamsFromIndexers, hashesFromIndexers, getStreamsFromIndexersError = GetStreamsFromIndexers(timeoutCtx, ctx, contentType, id)
		log.Debug("fetched streams from indexers", "count", len(wrappedStreamsFromIndexers))
	})

	wg.Wait()

	if getStreamsError != nil {
		SendError(w, r, getStreamsError)
		return
	}

	if getStreamsFromIndexersError != nil {
		log.Error("failed to get streams from indexers", "error", getStreamsFromIndexersError)
	} else {
		wrappedStreams = append(wrappedStreams, wrappedStreamsFromIndexers...)
		for _, hash := range hashesFromIndexers {
			if !hashSet.Has(hash) {
				hashSet.Add(hash)
				hashes = append(hashes, hash)
			}
		}
	}

	isP2P := ud.IsP2P()

	var isCachedByHash map[string]string
	var hasErrByStoreCode map[string]struct{}
	var checkMagnetError error
	if !isP2P && len(hashes) > 0 {
		cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
			Magnets:  hashes,
			ClientIP: ctx.ClientIP,
			SId:      id,
		}, ctx.Log)
		if cmRes.HasErr && len(cmRes.ByHash) == 0 {
			checkMagnetError = errors.Join(cmRes.Err...)
		} else {
			isCachedByHash = cmRes.ByHash
			hasErrByStoreCode = cmRes.HasErrByStoreCode
		}
	}

	if checkMagnetError != nil {
		SendError(w, r, checkMagnetError)
		return
	}

	wrappedStreams = filterStreams(wrappedStreams, filter)

	stremio_transformer.SortStreams(wrappedStreams, ud.Sort)

	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/torz", eud, "_/strem", id)

	cachedStreams := []stremio.Stream{}
	uncachedStreams := []stremio.Stream{}
	for _, wStream := range wrappedStreams {
		hash := wStream.R.Hash
		if isP2P {
			if wStream.FileIndex == -1 || wStream.R.IsPrivate {
				continue
			}

			wStream.R.Store.Code = "P2P"
			wStream.R.Store.Name = "P2P"
			stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
			if err != nil {
				SendError(w, r, err)
				return
			}
			uncachedStreams = append(uncachedStreams, *stream)
		} else if storeCode, isCached := isCachedByHash[hash]; isCached && storeCode != "" {
			storeName := store.StoreCode(strings.ToLower(storeCode)).Name()
			wStream.R.Store.Code = storeCode
			wStream.R.Store.Name = string(storeName)
			wStream.R.Store.IsCached = true
			wStream.R.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
			stream, err := streamTemplate.Execute(wStream.Stream, wStream.R)
			if err != nil {
				SendError(w, r, err)
				return
			}
			identifier := hash
			steamUrl := streamBaseUrl.JoinPath(strings.ToLower(storeCode), identifier, strconv.Itoa(wStream.R.File.Idx), "/")
			if wStream.R.File.Name != "" {
				steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.File.Name))
			}
			stream.URL = steamUrl.String()
			stream.InfoHash = ""
			stream.FileIndex = 0
			cachedStreams = append(cachedStreams, *stream)
		} else if !ud.CachedOnly {
			stores := ud.GetStores()
			for i := range stores {
				s := &stores[i]
				storeName := s.Store.GetName()
				storeCode := storeName.Code()
				if _, hasErr := hasErrByStoreCode[strings.ToUpper(string(storeCode))]; hasErr || storeCode == store.StoreCodeEasyDebrid {
					continue
				}
				if wStream.R.IsPrivate && (!ud.IncludeUncachedPrivate || wStream.torrentLink == "" || storeCode != store.StoreCodeTorBox) {
					continue
				}

				origStream := *wStream.Stream
				wStream.R.Store.Code = strings.ToUpper(string(storeCode))
				wStream.R.Store.Name = string(storeName)
				wStream.R.Store.IsProxied = ctx.IsProxyAuthorized && config.StoreContentProxy.IsEnabled(string(storeName))
				stream, err := streamTemplate.Execute(&origStream, wStream.R)
				if err != nil {
					SendError(w, r, err)
					return
				}

				identifier := hash
				if wStream.R.IsPrivate {
					identifier += "-" + util.Base64Encode(wStream.torrentLink)
				}
				steamUrl := streamBaseUrl.JoinPath(string(storeCode), identifier, strconv.Itoa(wStream.R.File.Idx), "/")
				if wStream.R.File.Name != "" {
					steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.File.Name))
				}
				stream.URL = steamUrl.String()
				stream.InfoHash = ""
				stream.FileIndex = 0
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

	if isP2P && !torzLazyPull {
		w.Header().Set("Cache-Control", "public, max-age=7200")
	}

	SendResponse(w, r, 200, &stremio.StreamHandlerResponse{
		Streams: streams,
	})
}
