package stremio_torz

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/anime"
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
var torzLazyPull = config.Stremio.Torz.LazyPull

type WrappedStream struct {
	*stremio.Stream
	R *stremio_transformer.StreamExtractorResult
}

func (s WrappedStream) IsSortable() bool {
	return s.R != nil
}

func (s WrappedStream) GetQuality() string {
	return s.R.Quality
}

func (s WrappedStream) GetResolution() string {
	return s.R.Resolution
}

func (s WrappedStream) GetSize() string {
	return s.R.Size
}

func (s WrappedStream) GetHDR() string {
	return strings.Join(s.R.HDR, "|")
}

func GetStreamsForHashes(stremType, stremId string, hashes []string) ([]WrappedStream, error) {
	isKitsuId := strings.HasPrefix(stremId, "kitsu:")
	isMALId := strings.HasPrefix(stremId, "mal:")
	isAnime := isKitsuId || isMALId

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
			if isAnime {
				var anidbId, episode string
				var err error

				if isKitsuId {
					kitsuId, kitsuEpisode, _ := strings.Cut(strings.TrimPrefix(stremId, "kitsu:"), ":")
					anidbId, _, err = anime.GetAniDBIdByKitsuId(kitsuId)
					episode = kitsuEpisode
				} else if isMALId {
					malId, malEpisode, _ := strings.Cut(strings.TrimPrefix(stremId, "mal:"), ":")
					anidbId, _, err = anime.GetAniDBIdByMALId(malId)
					episode = malEpisode
				}
				if err != nil || anidbId == "" {
					if err != nil {
						log.Error("failed to get anidb id for anime", "id", stremId, "error", err)
					}
					idToMatch = ""
				} else {
					idToMatch = anidbId + ":" + episode
				}
			}
			if idToMatch != "" {
				for i := range files {
					f := &files[i]
					if core.HasVideoExtension(f.Name) {
						if f.SId == idToMatch || f.ASId == idToMatch {
							file = f
						}
					}
				}
			}
		}
		fName := ""
		fIdx := -1
		fSize := int64(0)
		fVideoHash := ""
		if file != nil {
			fIdx = file.Idx
			fName = file.Name
			if file.Size > 0 {
				fSize = file.Size
			}
			fVideoHash = file.VideoHash
		} else if core.HasVideoExtension(tInfo.TorrentTitle) {
			fName = tInfo.TorrentTitle
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
			Addon: stremio_transformer.StreamExtractorResultAddon{
				Name: "Torz",
			},
			Category: stremType,
			File: stremio_transformer.StreamExtractorResultFile{
				Name: fName,
				Idx:  fIdx,
			},
		}
		if fSize > 0 {
			data.File.Size = util.ToSize(fSize)
		}
		wrappedStreams = append(wrappedStreams, WrappedStream{
			R: data,
			Stream: &stremio.Stream{
				Name:        data.Addon.Name,
				Description: data.TTitle,
				InfoHash:    data.Hash,
				FileIndex:   fIdx,
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

	eud := ud.GetEncoded()

	if nsid, err := torrent_stream.NormalizeStreamId(id); err == nil {
		cleanSId := nsid.ToClean()
		if torzLazyPull {
			go buddy.PullTorrentsByStremId(cleanSId, "")
		} else {
			buddy.PullTorrentsByStremId(cleanSId, "")
		}
	} else if !errors.Is(err, torrent_stream.ErrUnsupportedStremId) {
		log.Error("failed to normalize strem id", "error", err, "id", id)
	}

	hashes, err := torrent_info.ListHashesByStremId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	var wg sync.WaitGroup

	isP2P := ud.IsP2P()

	var isCachedByHash map[string]string
	var hasErrByStoreCode map[string]struct{}
	var checkMagnetError error
	if !isP2P && len(hashes) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cmRes := ud.CheckMagnet(&store.CheckMagnetParams{
				Magnets:  hashes,
				ClientIP: ctx.ClientIP,
				SId:      id,
			}, log)
			if cmRes.HasErr && len(cmRes.ByHash) == 0 {
				checkMagnetError = errors.Join(cmRes.Err...)
				return
			}
			isCachedByHash = cmRes.ByHash
			hasErrByStoreCode = cmRes.HasErrByStoreCode
		}()
	}

	var wrappedStreams []WrappedStream
	var getStreamsError error

	wg.Add(1)
	go func() {
		defer wg.Done()
		wrappedStreams, getStreamsError = GetStreamsForHashes(contentType, id, hashes)
	}()

	wg.Wait()

	if checkMagnetError != nil {
		SendError(w, r, checkMagnetError)
		return
	}

	if getStreamsError != nil {
		SendError(w, r, getStreamsError)
		return
	}

	stremio_transformer.SortStreams(wrappedStreams, "")

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
			steamUrl := streamBaseUrl.JoinPath(strings.ToLower(storeCode), hash, strconv.Itoa(wStream.R.File.Idx), "/")
			if wStream.R.File.Name != "" {
				steamUrl = steamUrl.JoinPath(url.PathEscape(wStream.R.File.Name))
			}
			stream.URL = steamUrl.String()
			stream.InfoHash = ""
			stream.FileIndex = 0
			cachedStreams = append(cachedStreams, *stream)
		} else if !ud.CachedOnly && !wStream.R.IsPrivate {
			stores := ud.GetStores()
			for i := range stores {
				s := &stores[i]
				storeName := s.Store.GetName()
				storeCode := storeName.Code()
				if _, hasErr := hasErrByStoreCode[strings.ToUpper(string(storeCode))]; hasErr || storeCode == store.StoreCodeEasyDebrid {
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

				steamUrl := streamBaseUrl.JoinPath(string(storeCode), hash, strconv.Itoa(wStream.R.File.Idx), "/")
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
