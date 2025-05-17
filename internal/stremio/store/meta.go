package stremio_store

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_store_usenet "github.com/MunifTanjim/stremthru/internal/stremio/store/usenet"
	stremio_store_webdl "github.com/MunifTanjim/stremthru/internal/stremio/store/webdl"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"golang.org/x/sync/singleflight"
)

var client = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

var cinemetaBaseUrl = func() *url.URL {
	url, err := url.Parse("https://v3-cinemeta.strem.io/")
	if err != nil {
		panic(err)
	}
	return url
}()

var metaCache = cache.NewCache[stremio.MetaHandlerResponse](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "stremio:store:catalog",
})

var fetchMetaGroup singleflight.Group

func fetchMeta(sType, imdbId, clientIp string) (stremio.MetaHandlerResponse, error) {
	var meta stremio.MetaHandlerResponse

	cacheKey := sType + ":" + imdbId
	if !metaCache.Get(cacheKey, &meta) {
		m, err, _ := fetchMetaGroup.Do(cacheKey, func() (any, error) {
			r, err := client.FetchMeta(&stremio_addon.FetchMetaParams{
				BaseURL:  cinemetaBaseUrl,
				Type:     sType,
				Id:       imdbId + ".json",
				ClientIP: clientIp,
			})
			return r.Data, err
		})
		if err != nil {
			return meta, err
		}
		meta = m.(stremio.MetaHandlerResponse)
		metaCache.Add(cacheKey, meta)
	}

	return meta, nil
}

func getPosterUrl(imdbId string) string {
	return "https://images.metahub.space/poster/small/" + imdbId + "/img"
}

func getMetaPreviewDescription(description string, r *ptt.Result, includeSeriesMeta bool) string {
	if r.Title != "" {
		description += " [ ✏️ " + r.Title + " ]"
	}
	if includeSeriesMeta {
		meta := ""
		if len(r.Seasons) > 0 {
			meta += "S" + strconv.Itoa(r.Seasons[0])
		}
		if len(r.Episodes) > 0 {
			if meta != "" {
				meta += " · "
			}
			meta += "E"
			meta += strconv.Itoa(r.Episodes[0])
		}
		if meta != "" {
			description += " [ " + meta + " ]"
		}
	}
	if r.Year != "" || r.Date != "" {
		description += " [ 📅 "
		if r.Year != "" {
			description += r.Year
			if r.Date != "" {
				description += " | "
			}
		}
		if r.Date != "" {
			description += r.Date
		}
		description += " ]"
	}
	if r.Resolution != "" {
		description += " [ 🎥 " + r.Resolution + " ]"
	}
	if r.Quality != "" {
		description += " [ 💿 " + r.Quality + " ]"
	}
	if r.Codec != "" {
		description += " [ 🎞️ " + r.Codec + " ]"
	}
	if len(r.HDR) > 0 {
		description += " [ 📺 " + strings.Join(r.HDR, ",") + " ]"
	}
	if audioCount, channelCount := len(r.Audio), len(r.Channels); audioCount > 0 || channelCount > 0 {
		description += " [ 🎧 "
		if audioCount > 0 {
			description += strings.Join(r.Audio, ",")
			if channelCount > 0 {
				description += " | "
			}
		}
		if channelCount > 0 {
			description += strings.Join(r.Channels, ",")
		}
		description += " ]"
	}
	if r.ThreeD != "" {
		description += " [ 🎲 " + r.ThreeD + " ]"
	}
	if r.Network != "" {
		description += " [ 📡 " + r.Network + " ]"
	}
	if r.Group != "" {
		description += " [ ⚙️ " + r.Group + " ]"
	}
	if r.Site != "" {
		description += " [ 🔗 " + r.Site + " ]"
	}
	return description
}

func getMetaPreviewDescriptionForTorrent(hash, name string) string {
	description := "[ 🧲 " + hash + " ]"

	r, err := util.ParseTorrentTitle(name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", name)
		return description
	}

	return getMetaPreviewDescription(description, r, false)
}

func getMetaPreviewDescriptionForUsenet(hash, name string, largestFilename string) string {
	description := "[ 🌐 " + hash + " ]"

	r, err := util.ParseTorrentTitle(name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", name)
		return description
	}

	if largestFilename != "" && largestFilename != name {
		description += " [ 📁 " + name + " ]"

		if fr, err := util.ParseTorrentTitle(largestFilename); err == nil {
			if r.Title == "" {
				r.Title = fr.Title
			} else {
				r.Title = fr.Title + " | " + r.Title
			}
			if r.Year == "" {
				r.Year = fr.Year
			}
			if r.Date == "" {
				r.Date = fr.Date
			}
			if r.Resolution == "" {
				r.Resolution = fr.Resolution
			}
			if r.Quality == "" {
				r.Quality = fr.Quality
			}
			if r.Codec == "" {
				r.Codec = fr.Codec
			}
			if len(r.HDR) == 0 {
				r.HDR = fr.HDR
			}
			if len(r.Audio) == 0 {
				r.Audio = fr.Audio
			}
			if len(r.Channels) == 0 {
				r.Channels = fr.Channels
			}
			if r.ThreeD == "" {
				r.ThreeD = fr.ThreeD
			}
			if r.Network == "" {
				r.Network = fr.Network
			}
			if r.Group == "" {
				r.Group = fr.Group
			} else {
				r.Group = fr.Group + " | " + r.Group
			}
			if r.Site == "" {
				r.Site = fr.Site
			}
		} else {
			pttLog.Warn("failed to parse", "error", err, "title", name)
		}
	}

	return getMetaPreviewDescription(description, r, false)
}

func getMetaPreviewDescriptionForWebDL(hash, name string, includeSeriesMeta bool) string {
	description := "[ 📥 " + hash + " ]"

	r, err := util.ParseTorrentTitle(name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", name)
		return description
	}

	return getMetaPreviewDescription(description, r, includeSeriesMeta)
}

type contentInfo struct {
	*store.GetMagnetData
	largestFilename string
}

func getStoreContentInfo(s store.Store, storeToken string, id string, clientIp string, idr *ParsedId) (*contentInfo, error) {
	if idr.isUsenet {
		if s.GetName() != store.StoreNameTorBox {
			return nil, nil
		}

		params := &stremio_store_usenet.GetNewsParams{
			Id: id,
		}
		params.APIKey = storeToken
		news, err := stremio_store_usenet.GetNews(params, s.GetName())
		if err != nil {
			return nil, err
		}
		cInfo := &store.GetMagnetData{
			AddedAt: news.AddedAt,
			Hash:    news.Hash,
			Id:      news.Id,
			Name:    news.GetLargestFileName(),
			Size:    news.Size,
			Status:  news.Status,
		}
		for _, f := range news.Files {
			cInfo.Files = append(cInfo.Files, store.MagnetFile{
				Idx:  f.Idx,
				Name: f.Name,
				Size: f.Size,
				Link: f.Link,
			})

		}
		return &contentInfo{cInfo, news.GetLargestFileName()}, nil
	}

	if idr.isWebDL {
		if s.GetName() != store.StoreNameTorBox {
			return nil, nil
		}

		params := &stremio_store_webdl.GetWebDLParams{
			Id: id,
		}
		params.APIKey = storeToken
		webdl, err := stremio_store_webdl.GetWebDL(params, s.GetName())
		if err != nil {
			return nil, err
		}
		cInfo := &store.GetMagnetData{
			AddedAt: webdl.AddedAt,
			Hash:    webdl.Hash,
			Id:      webdl.Id,
			Name:    webdl.Name,
			Size:    webdl.Size,
			Status:  webdl.Status,
		}
		for _, f := range webdl.Files {
			cInfo.Files = append(cInfo.Files, store.MagnetFile{
				Idx:  f.Idx,
				Name: f.Name,
				Size: f.Size,
				Link: f.Link,
			})

		}
		return &contentInfo{cInfo, ""}, nil
	}

	params := &store.GetMagnetParams{
		Id:       id,
		ClientIP: clientIp,
	}
	params.APIKey = storeToken
	magnet, err := s.GetMagnet(params)
	if err != nil {
		return nil, err
	}

	return &contentInfo{magnet, ""}, nil
}

func getStoreActionMeta(r *http.Request, storeCode string, eud string) stremio.Meta {
	released := time.Now().UTC()
	meta := stremio.Meta{
		Id:          getStoreActionId(storeCode),
		Type:        ContentTypeOther,
		Name:        "StremThru Store Actions",
		Description: "Actions for StremThru Store",
		Released:    &released,
		Videos: []stremio.MetaVideo{
			{
				Id:       getStoreActionIdPrefix(storeCode) + "clear_cache",
				Title:    "Clear Cache",
				Released: released,
				Streams: []stremio.Stream{
					{
						URL:         ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/_/action/" + getStoreActionIdPrefix(storeCode) + "clear_cache").String(),
						Name:        "Clear Cache",
						Description: "Clear Cached Data for StremThru Store",
					},
				},
			},
		},
	}
	return meta
}

func handleMeta(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	id := getId(r)
	idr, err := parseId(id)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idStoreCode := idr.getStoreCode()
	idPrefix := getIdPrefix(idStoreCode)

	if !strings.HasPrefix(id, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+id).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "").Send(w, r)
		return
	}

	if id == getStoreActionId(idStoreCode) {
		eud, err := ud.GetEncoded()
		if err != nil {
			SendError(w, r, err)
			return
		}

		res := stremio.MetaHandlerResponse{
			Meta: getStoreActionMeta(r, idStoreCode, eud),
		}

		SendResponse(w, r, 200, res)
		return
	}

	if idr.storeCode == store.StoreCodeRealDebrid && id == getRDWebDLsId(idStoreCode) {
		res := stremio.MetaHandlerResponse{
			Meta: getRDWebDLsMeta(r, ctx, idStoreCode),
		}

		SendResponse(w, r, 200, res)
		return
	}

	cInfo, err := getStoreContentInfo(ctx.Store, ctx.StoreAuthToken, strings.TrimPrefix(id, idPrefix), ctx.ClientIP, idr)
	if err != nil {
		SendError(w, r, err)
		return
	}

	meta := stremio.Meta{
		Id:       id,
		Type:     ContentTypeOther,
		Name:     cInfo.Name,
		Released: &cInfo.AddedAt,
		Videos:   []stremio.MetaVideo{},
	}

	sType, sId := "", ""

	if idr.isUsenet {
		meta.Description = getMetaPreviewDescriptionForUsenet(cInfo.Hash, cInfo.Name, cInfo.largestFilename)
	} else if idr.isWebDL {
		meta.Description = getMetaPreviewDescriptionForWebDL(cInfo.Hash, cInfo.Name, false)
	} else {
		meta.Description = getMetaPreviewDescriptionForTorrent(cInfo.Hash, cInfo.Name)

		if stremIdByHashes, err := torrent_stream.GetStremIdByHashes([]string{cInfo.Hash}); err != nil {
			log.Error("failed to get strem id by hashes", "error", err)
		} else {
			if sid := stremIdByHashes.Get(cInfo.Hash); sid != "" {
				sid, _, isSeries := strings.Cut(sid, ":")
				sId = sid
				if isSeries {
					sType = "series"
				} else {
					sType = "movie"
				}
			}
		}
	}

	metaVideoByKey := map[string]*stremio.MetaVideo{}
	if sId != "" {
		if r, err := fetchMeta(sType, sId, core.GetRequestIP(r)); err != nil {
			log.Error("failed to fetch meta", "error", err)
		} else {
			m := r.Meta
			meta.Description += " " + m.Description
			meta.Poster = m.Poster
			meta.Background = m.Background
			meta.Links = m.Links
			meta.Logo = m.Logo
			meta.Released = m.Released

			if sType == "series" {
				for i := range m.Videos {
					video := &m.Videos[i]
					key := video.Season.String() + ":" + video.Episode.String()
					metaVideoByKey[key] = video
				}
			}
		}
	}

	tInfo := torrent_info.TorrentInfoInsertData{
		Hash:         cInfo.Hash,
		TorrentTitle: cInfo.Name,
		Size:         cInfo.Size,
		Source:       torrent_info.TorrentInfoSource(ctx.Store.GetName().Code()),
		Files:        []torrent_info.TorrentInfoInsertDataFile{},
	}

	tpttr, err := util.ParseTorrentTitle(cInfo.Name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", cInfo.Name)
	}

	for _, f := range cInfo.Files {
		if !core.HasVideoExtension(f.Name) {
			continue
		}

		videoId := id + ":" + url.PathEscape(f.Link)
		video := stremio.MetaVideo{
			Id:        videoId,
			Title:     f.Name,
			Available: true,
			Released:  cInfo.AddedAt,
		}

		season, episode := -1, -1
		pttr, err := util.ParseTorrentTitle(f.Name)
		if err != nil {
			pttLog.Warn("failed to parse", "error", err, "title", f.Name)
		} else {
			if len(pttr.Seasons) > 0 {
				season = pttr.Seasons[0]
				video.Season = stremio.ZeroIndexedInt(season)
			} else if len(tpttr.Seasons) == 1 {
				season = tpttr.Seasons[0]
				video.Season = stremio.ZeroIndexedInt(season)
			}
			if len(pttr.Episodes) > 0 {
				episode = pttr.Episodes[0]
				video.Episode = stremio.ZeroIndexedInt(episode)
			}
		}
		if season != -1 && episode != -1 {
			key := strconv.Itoa(season) + ":" + strconv.Itoa(episode)
			if sType == "series" {
				if metaVideo, ok := metaVideoByKey[key]; ok {
					video.Released = metaVideo.Released
					video.Thumbnail = metaVideo.Thumbnail
					video.Title = metaVideo.Name + "\n📄 " + f.Name
				} else {
					video.Title = pttr.Title + "\n📄 " + f.Name
				}
			}
		}

		meta.Videos = append(meta.Videos, video)

		tInfo.Files = append(tInfo.Files, torrent_info.TorrentInfoInsertDataFile{
			Name: f.Name,
			Idx:  f.Idx,
			Size: f.Size,
		})
	}

	if !idr.isUsenet && !idr.isWebDL {
		go torrent_info.Upsert([]torrent_info.TorrentInfoInsertData{tInfo}, "", ctx.Store.GetName().Code() != store.StoreCodeRealDebrid)
	}

	res := stremio.MetaHandlerResponse{
		Meta: meta,
	}

	SendResponse(w, r, 200, res)
}
