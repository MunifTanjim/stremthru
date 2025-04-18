package stremio_store

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/server"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/internal/store/video"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	stremio_transformer "github.com/MunifTanjim/stremthru/internal/stremio/transformer"
	"github.com/MunifTanjim/stremthru/internal/torrent_info"
	"github.com/MunifTanjim/stremthru/internal/torrent_stream"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
	"github.com/paul-mannino/go-fuzzywuzzy"
)

var streamTemplate = func() *stremio_transformer.StreamTemplate {
	tmplBlob := stremio_transformer.StreamTemplateBlob{
		Name: `Store {{.StoreCode}}
{{ if ne .Resolution ""}}{{.Resolution}}{{end}}`,
		Description: `✏️ {{.Title}}
{{if ne .Quality ""}} 💿 {{.Quality}} {{end}}{{if ne .Codec ""}} 🎞️ {{.Codec}} {{end}}{{if gt (len .HDR) 0}} 📺 {{str_join .HDR ","}}{{end}}{{if gt (len .Audio) 0}} 🎧 {{str_join .Audio ","}}{{if gt (len .Channels) 0}} | {{str_join .Channels ","}}{{end}}{{end}}
{{if ne .Size ""}} 📦 {{.Size}}{{end}}{{if ne .Group ""}} ⚙️ {{.Group}}{{end}}
📄 {{.Raw.Name}}`,
	}
	tmpl, err := tmplBlob.Parse()
	if err != nil {
		panic(err)
	}
	return tmpl
}()

type UserData struct {
	StoreName  string `json:"store_name"`
	StoreToken string `json:"store_token"`
	encoded    string `json:"-"`

	idPrefixes []string `json:"-"`
}

func (ud UserData) HasRequiredValues() bool {
	return ud.StoreToken != ""
}

func (ud UserData) GetEncoded() (string, error) {
	if ud.encoded != "" {
		return ud.encoded, nil
	}

	blob, err := json.Marshal(ud)
	if err != nil {
		return "", err
	}
	return core.Base64Encode(string(blob)), nil
}

func (ud *UserData) getIdPrefixes() []string {
	if len(ud.idPrefixes) == 0 {
		if ud.StoreName == "" {
			if user, err := core.ParseBasicAuth(ud.StoreToken); err == nil {
				if password := config.ProxyAuthPassword.GetPassword(user.Username); password != "" && password == user.Password {
					for _, name := range config.StoreAuthToken.ListStores(user.Username) {
						storeCode := "st-" + string(store.StoreName(name).Code())
						ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(storeCode))
					}
				}
			}
		} else {
			storeCode := string(store.StoreName(ud.StoreName).Code())
			ud.idPrefixes = append(ud.idPrefixes, getIdPrefix(storeCode))
		}
	}
	return ud.idPrefixes
}

type userDataError struct {
	storeToken string
	storeName  string
}

func (uderr *userDataError) Error() string {
	var str strings.Builder
	hasSome := false
	if uderr.storeName != "" {
		str.WriteString("store_name: ")
		str.WriteString(uderr.storeName)
		hasSome = true
	}
	if hasSome {
		str.WriteString(", ")
	}
	if uderr.storeToken != "" {
		str.WriteString("store_token: ")
		str.WriteString(uderr.storeToken)
	}
	return str.String()
}

func (ud UserData) GetRequestContext(r *http.Request, idr *ParsedId) (*context.StoreContext, error) {
	rCtx := server.GetReqCtx(r)
	ctx := &context.StoreContext{
		Log: rCtx.Log,
	}

	storeToken := ud.StoreToken
	if idr.isST {
		user, err := core.ParseBasicAuth(storeToken)
		if err != nil {
			return ctx, &userDataError{storeToken: err.Error()}
		}
		password := config.ProxyAuthPassword.GetPassword(user.Username)
		if password != "" && password == user.Password {
			ctx.IsProxyAuthorized = true
			ctx.ProxyAuthUser = user.Username
			ctx.ProxyAuthPassword = user.Password

			if idr.storeName == "" {
				idr.storeName = store.StoreName(config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser))
			}
			storeToken = config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(idr.storeName))
		}
	}

	if storeToken != "" {
		ctx.Store = shared.GetStore(string(idr.storeName))
		ctx.StoreAuthToken = storeToken
	}

	ctx.ClientIP = shared.GetClientIP(r, ctx)

	return ctx, nil
}

func getUserData(r *http.Request) (*UserData, error) {
	data := &UserData{}

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		data.encoded = r.PathValue("userData")
		if data.encoded == "" {
			return data, nil
		}
		blob, err := core.Base64DecodeToByte(data.encoded)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(blob, data)
		return data, err
	}

	if IsMethod(r, http.MethodPost) {
		data.StoreName = r.FormValue("store_name")
		data.StoreToken = r.FormValue("store_token")
		encoded, err := data.GetEncoded()
		if err != nil {
			return nil, err
		}
		data.encoded = encoded
	}

	return data, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/stremio/store/configure", http.StatusFound)
}

func handleManifest(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	manifest := GetManifest(r, ud)

	SendResponse(w, r, 200, manifest)
}

func handleConfigure(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	td := getTemplateData(ud)

	if IsMethod(r, http.MethodGet) {
		if ud.HasRequiredValues() {
			if eud, err := ud.GetEncoded(); err == nil {
				td.ManifestURL = ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/manifest.json").String()
			}
		}

		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	var name_config *configure.Config
	var token_config *configure.Config
	for i := range td.Configs {
		conf := &td.Configs[i]
		switch conf.Key {
		case "store_name":
			name_config = conf
		case "store_token":
			token_config = conf
		}
	}

	idr := ParsedId{isST: ud.StoreName == ""}
	idr.storeName = store.StoreName(ud.StoreName)
	idr.storeCode = idr.storeName.Code()
	ctx, err := ud.GetRequestContext(r, &idr)
	if err != nil {
		if uderr, ok := err.(*userDataError); ok {
			if uderr.storeName != "" {
				name_config.Error = uderr.storeName
			}
			if uderr.storeToken != "" {
				token_config.Error = uderr.storeToken
			}
		} else {
			SendError(w, r, err)
			return
		}
	}

	if ctx.Store == nil {
		if ud.StoreName == "" {
			token_config.Error = "Invalid Token"
		} else {
			name_config.Error = "Invalid Store"
		}
	} else if token_config.Error == "" {
		params := &store.GetUserParams{}
		params.APIKey = ctx.StoreAuthToken
		user, err := ctx.Store.GetUser(params)
		if err != nil {
			LogError(r, "failed to get user", err)
			token_config.Error = "Invalid Token"
		} else if user.SubscriptionStatus == store.UserSubscriptionStatusExpired {
			token_config.Error = "Subscription Expired"
		}
	}

	if td.HasError() {
		page, err := configure.GetPage(td)
		if err != nil {
			SendError(w, r, err)
			return
		}
		SendHTML(w, 200, page)
		return
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	url := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/configure")
	q := url.Query()
	q.Set("try_install", "1")
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}

func getContentType(r *http.Request) (string, *core.APIError) {
	contentType := r.PathValue("contentType")
	if contentType != ContentTypeOther {
		return "", shared.ErrorBadRequest(r, "unsupported type: "+contentType)
	}
	return contentType, nil
}

func getPathParam(r *http.Request, name string) string {
	if value := r.PathValue(name + "Json"); value != "" {
		return strings.TrimSuffix(value, ".json")
	}
	return r.PathValue(name)
}

func getId(r *http.Request) string {
	return getPathParam(r, "id")
}

type ExtraData struct {
	Search string
	Skip   int
	Genre  string
}

func getExtra(r *http.Request) *ExtraData {
	extra := &ExtraData{}
	if extraParams := getPathParam(r, "extra"); extraParams != "" {
		if q, err := url.ParseQuery(extraParams); err == nil {
			if search := q.Get("search"); search != "" {
				extra.Search = search
			}
			if skipStr := q.Get("skip"); skipStr != "" {
				if skip, err := strconv.Atoi(skipStr); err == nil {
					extra.Skip = skip
				}
			}
			if genre := q.Get("genre"); genre != "" {
				extra.Genre = genre
			}
		}
	}
	return extra
}

type CachedCatalogItem struct {
	stremio.MetaPreview
	hash string
}

var catalogCache = func() cache.Cache[[]CachedCatalogItem] {
	c := cache.NewCache[[]CachedCatalogItem](&cache.CacheConfig{
		Lifetime: 10 * time.Minute,
		Name:     "stremio:store:catalog",
	})
	return c
}()

func getCatalogCacheKey(idPrefix, storeToken string) string {
	return idPrefix + storeToken
}

func getMetaPreviewDescription(hash, name string) string {
	description := "[ 🧲 " + hash + " ]"

	r, err := util.ParseTorrentTitle(name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", name)
		return description
	}

	if r.Title != "" {
		description += " [ ✏️ " + r.Title + " ]"
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

func getCatalogItems(s store.Store, storeToken string, clientIp string, idPrefix string) []CachedCatalogItem {
	items := []CachedCatalogItem{}

	cacheKey := getCatalogCacheKey(idPrefix, storeToken)
	if !catalogCache.Get(cacheKey, &items) {
		tInfoItems := []torrent_info.TorrentInfoInsertData{}
		tInfoSource := torrent_info.TorrentInfoSource(s.GetName().Code())

		limit := 500
		offset := 0
		hasMore := true
		for hasMore && offset < 2000 {
			params := &store.ListMagnetsParams{
				Limit:    limit,
				Offset:   offset,
				ClientIP: clientIp,
			}
			params.APIKey = storeToken
			res, err := s.ListMagnets(params)
			if err != nil {
				break
			}

			for _, item := range res.Items {
				if item.Status == store.MagnetStatusDownloaded {
					items = append(items, CachedCatalogItem{stremio.MetaPreview{
						Id:          idPrefix + item.Id,
						Type:        ContentTypeOther,
						Name:        item.Name,
						Description: getMetaPreviewDescription(item.Hash, item.Name),
					}, item.Hash})
				}
				tInfoItems = append(tInfoItems, torrent_info.TorrentInfoInsertData{
					Hash:         item.Hash,
					TorrentTitle: item.Name,
					Size:         item.Size,
					Source:       tInfoSource,
				})
			}
			offset += limit
			hasMore = len(res.Items) == limit && offset < res.TotalItems
			time.Sleep(1 * time.Second)
		}
		catalogCache.Add(cacheKey, items)
		go torrent_info.Upsert(tInfoItems, "", s.GetName().Code() != store.StoreCodeRealDebrid)
	}

	return items
}

func getStoreActionMetaPreview(storeCode string) stremio.MetaPreview {
	meta := stremio.MetaPreview{
		Id:   getStoreActionId(storeCode),
		Type: ContentTypeOther,
		Name: "StremThru Store Actions",
	}
	return meta
}

var whitespacesRegex = regexp.MustCompile(`\s+`)

func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if _, err := getContentType(r); err != nil {
		err.Send(w, r)
		return
	}

	catalogId := getId(r)
	idr, err := parseId(catalogId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	if catalogId != getCatalogId(idr.getStoreCode()) {
		shared.ErrorBadRequest(r, "unsupported catalog id: "+catalogId).Send(w, r)
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

	extra := getExtra(r)

	res := stremio.CatalogHandlerResponse{
		Metas: []stremio.MetaPreview{},
	}

	if extra.Genre == CatalogGenreStremThru {
		res.Metas = append(res.Metas, getStoreActionMetaPreview(idr.getStoreCode()))
		SendResponse(w, r, 200, res)
		return
	}

	idPrefix := getIdPrefix(idr.getStoreCode())

	items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, idPrefix)

	if extra.Search != "" {
		query := strings.ToLower(extra.Search)
		parts := whitespacesRegex.Split(query, -1)
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		regex, err := regexp.Compile(strings.Join(parts, ".*"))
		if err != nil {
			SendError(w, r, err)
			return
		}
		filteredItems := []CachedCatalogItem{}
		for i := range items {
			item := &items[i]
			if regex.MatchString(strings.ToLower(item.Name)) {
				filteredItems = append(filteredItems, *item)
			}
		}
		items = filteredItems
	}

	limit := 100
	totalItems := len(items)
	items = items[min(extra.Skip, totalItems):min(extra.Skip+limit, totalItems)]

	hashes := make([]string, len(items))
	for i := range items {
		item := &items[i]
		hashes[i] = item.hash
	}

	res.Metas = make([]stremio.MetaPreview, len(hashes))

	stremIdByHash, err := torrent_stream.GetStremIdByHashes(hashes)
	if err != nil {
		log.Error("failed to get strem id by hashes", "error", err)
	}
	for i := range items {
		item := &items[i]
		if stremId := stremIdByHash.Get(item.hash); stremId != "" {
			stremId, _, _ = strings.Cut(stremId, ":")
			item.Poster = getPosterUrl(stremId)
		}
		res.Metas[i] = item.MetaPreview
	}

	SendResponse(w, r, 200, res)
}

func getStoreActionMeta(r *http.Request, storeCode string, eud string) stremio.Meta {
	released := time.Now().UTC()
	meta := stremio.Meta{
		Id:          getStoreActionId(storeCode),
		Type:        ContentTypeOther,
		Name:        "StremThru Store Actions",
		Description: "Actions for StremThru Store",
		Released:    released,
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

	idPrefix := getIdPrefix(idr.getStoreCode())

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

	if id == getStoreActionId(idr.getStoreCode()) {
		eud, err := ud.GetEncoded()
		if err != nil {
			SendError(w, r, err)
			return
		}

		res := stremio.MetaHandlerResponse{
			Meta: getStoreActionMeta(r, idr.getStoreCode(), eud),
		}

		SendResponse(w, r, 200, res)
		return
	}

	params := &store.GetMagnetParams{
		Id:       strings.TrimPrefix(id, idPrefix),
		ClientIP: ctx.ClientIP,
	}
	params.APIKey = ctx.StoreAuthToken
	magnet, err := ctx.Store.GetMagnet(params)
	if err != nil {
		SendError(w, r, err)
		return
	}

	meta := stremio.Meta{
		Id:          id,
		Type:        ContentTypeOther,
		Name:        magnet.Name,
		Description: getMetaPreviewDescription(magnet.Hash, magnet.Name),
		Released:    magnet.AddedAt,
		Videos:      []stremio.MetaVideo{},
	}

	sType, sId := "", ""
	if stremIdByHashes, err := torrent_stream.GetStremIdByHashes([]string{magnet.Hash}); err != nil {
		log.Error("failed to get strem id by hashes", "error", err)
	} else {
		if sid := stremIdByHashes.Get(magnet.Hash); sid != "" {
			sid, _, isSeries := strings.Cut(sid, ":")
			sId = sid
			if isSeries {
				sType = "series"
			} else {
				sType = "movie"
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
					key := strconv.Itoa(video.Season) + ":" + strconv.Itoa(video.Episode)
					metaVideoByKey[key] = video
				}
			}
		}
	}

	tInfo := torrent_info.TorrentInfoInsertData{
		Hash:         magnet.Hash,
		TorrentTitle: magnet.Name,
		Size:         magnet.Size,
		Source:       torrent_info.TorrentInfoSource(ctx.Store.GetName().Code()),
		Files:        []torrent_info.TorrentInfoInsertDataFile{},
	}

	tpttr, err := util.ParseTorrentTitle(magnet.Name)
	if err != nil {
		pttLog.Warn("failed to parse", "error", err, "title", magnet.Name)
	}

	for _, f := range magnet.Files {
		if !core.HasVideoExtension(f.Name) {
			continue
		}

		videoId := id + ":" + url.PathEscape(f.Link)
		video := stremio.MetaVideo{
			Id:        videoId,
			Title:     f.Name,
			Available: true,
			Released:  magnet.AddedAt,
		}

		season, episode := -1, -1
		pttr, err := util.ParseTorrentTitle(f.Name)
		if err != nil {
			pttLog.Warn("failed to parse", "error", err, "title", f.Name)
		} else {
			if len(pttr.Seasons) > 0 {
				season = pttr.Seasons[0]
				video.Season = season
			} else if len(tpttr.Seasons) == 1 {
				season = tpttr.Seasons[0]
				video.Season = season
			}
			if len(pttr.Episodes) > 0 {
				episode = pttr.Episodes[0]
				video.Episode = episode
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

	go torrent_info.Upsert([]torrent_info.TorrentInfoInsertData{tInfo}, "", ctx.Store.GetName().Code() != store.StoreCodeRealDebrid)

	res := stremio.MetaHandlerResponse{
		Meta: meta,
	}

	SendResponse(w, r, 200, res)
}

type StreamFileMatcher struct {
	MagnetId       string
	FileLink       string
	FileName       string
	UseLargestFile bool
	Episode        int
	Season         int

	IdPrefix   string
	Store      store.Store
	StoreCode  string
	StoreToken string
	ClientIP   string
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

	videoIdWithLink := getId(r)
	contentType := r.PathValue("contentType")
	isStremThruStoreId := isStoreId(videoIdWithLink)
	isImdbId := strings.HasPrefix(videoIdWithLink, "tt")
	if isStremThruStoreId {
		if contentType != ContentTypeOther {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else if isImdbId {
		if contentType != string(stremio.ContentTypeMovie) && contentType != string(stremio.ContentTypeSeries) {
			shared.ErrorBadRequest(r, "unsupported type: "+contentType).Send(w, r)
			return
		}
	} else {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	res := stremio.StreamHandlerResponse{
		Streams: []stremio.Stream{},
	}

	eud, err := ud.GetEncoded()
	if err != nil {
		SendError(w, r, err)
		return
	}

	var meta *stremio.Meta
	season, episode := -1, -1

	matchers := []StreamFileMatcher{}

	if isStremThruStoreId {
		idr, err := parseId(videoIdWithLink)
		if err != nil {
			SendError(w, r, err)
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

		idPrefix := getIdPrefix(idr.getStoreCode())
		videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
		videoId, escapedLink, _ := strings.Cut(videoId, ":")
		link, err := url.PathUnescape(escapedLink)
		if err != nil {
			LogError(r, "failed to parse link", err)
			SendError(w, r, err)
			return
		}

		matchers = append(matchers, StreamFileMatcher{
			MagnetId: videoId,
			FileLink: link,

			IdPrefix:   idPrefix,
			Store:      ctx.Store,
			StoreCode:  idr.getStoreCode(),
			StoreToken: ctx.StoreAuthToken,
			ClientIP:   ctx.ClientIP,
		})
	}

	if isImdbId {
		sType, sId := "", ""
		sType, sId, season, episode = parseStremId(videoIdWithLink)
		mres, err := fetchMeta(sType, sId, core.GetRequestIP(r))
		if err != nil {
			SendError(w, r, err)
			return
		}
		meta = &mres.Meta

		var wg sync.WaitGroup

		idPrefixes := ud.getIdPrefixes()
		errs := make([]error, len(idPrefixes))
		matcherResults := make([][]StreamFileMatcher, len(idPrefixes))

		for idx, idPrefix := range idPrefixes {
			wg.Add(1)
			go func() {
				defer wg.Done()

				idr, err := parseId(idPrefix)
				if err != nil {
					errs[idx] = err
					return
				}
				ctx, err := ud.GetRequestContext(r, idr)
				if err != nil || ctx.Store == nil {
					if err != nil {
						LogError(r, "failed to get request context", err)
					}
					errs[idx] = shared.ErrorBadRequest(r, "")
					return
				}

				items := getCatalogItems(ctx.Store, ctx.StoreAuthToken, ctx.ClientIP, idPrefix)
				if meta.Name != "" {
					query := strings.ToLower(meta.Name)
					filteredItems := []CachedCatalogItem{}
					for i := range items {
						item := &items[i]
						if fuzzy.TokenSetRatio(query, strings.ToLower(item.Name), false, true) > 90 {
							filteredItems = append(filteredItems, *item)
						}
					}
					items = filteredItems
				}

				for i := range items {
					item := &items[i]
					id := strings.TrimPrefix(item.Id, idPrefix)
					if sType == "series" {
						matcherResults[idx] = append(matcherResults[idx], StreamFileMatcher{
							MagnetId: id,
							Season:   season,
							Episode:  episode,

							IdPrefix:   idPrefix,
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
						})
					} else {
						matcherResults[idx] = append(matcherResults[idx], StreamFileMatcher{
							MagnetId:       id,
							UseLargestFile: true,

							IdPrefix:   idPrefix,
							Store:      ctx.Store,
							StoreCode:  idr.getStoreCode(),
							StoreToken: ctx.StoreAuthToken,
							ClientIP:   ctx.ClientIP,
						})
					}
				}
			}()
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				SendError(w, r, err)
				return
			}
		}
		for i := range matcherResults {
			matchers = append(matchers, matcherResults[i]...)
		}
	}

	streamBaseUrl := ExtractRequestBaseURL(r).JoinPath("/stremio/store/" + eud + "/_/strem/")
	for _, matcher := range matchers {
		params := &store.GetMagnetParams{
			Id:       matcher.MagnetId,
			ClientIP: matcher.ClientIP,
		}
		params.APIKey = matcher.StoreToken
		magnet, err := matcher.Store.GetMagnet(params)
		if err != nil {
			SendError(w, r, err)
			return
		}

		if meta == nil {
			stremIdByHash, err := torrent_stream.GetStremIdByHashes([]string{magnet.Hash})
			if err != nil {
				log.Error("failed to get strem id by hashes", "error", err)
			}
			if stremId := stremIdByHash.Get(magnet.Hash); stremId != "" {
				sType, sId := "", ""
				sType, sId, season, episode = parseStremId(stremId)
				if mRes, err := fetchMeta(sType, sId, core.GetRequestIP(r)); err == nil {
					meta = &mRes.Meta
				} else {
					log.Error("failed to fetch meta", "error", err)
				}
			}
		}

		tpttr, err := util.ParseTorrentTitle(magnet.Name)
		if err != nil {
			pttLog.Warn("failed to parse", "error", err, "title", magnet.Name)
		}
		tSeason := -1
		if len(tpttr.Seasons) == 1 {
			tSeason = tpttr.Seasons[0]
		}

		var pttr *ptt.Result
		var file *store.MagnetFile

		for i := range magnet.Files {
			f := &magnet.Files[i]
			if matcher.FileLink != "" && matcher.FileLink == f.Link {
				file = f
				break
			} else if matcher.FileName != "" && matcher.FileName == f.Name {
				file = f
				break
			} else if matcher.Episode > 0 {
				if r, err := util.ParseTorrentTitle(f.Name); err == nil {
					pttr = r
					season, episode := tSeason, -1
					if len(r.Seasons) > 0 {
						season = r.Seasons[0]
					}
					if len(r.Episodes) > 0 {
						episode = r.Episodes[0]
					}
					if season == matcher.Season && episode == matcher.Episode {
						file = f
						break
					}
				} else {
					pttLog.Warn("failed to parse", "error", err, "title", f.Name)
				}
			} else if matcher.UseLargestFile {
				if file == nil || file.Size < f.Size {
					file = f
				}
			}
		}

		if file == nil {
			continue
		}

		streamId := matcher.IdPrefix + matcher.MagnetId + ":" + file.Link
		stream := stremio.Stream{
			URL:  streamBaseUrl.JoinPath(url.PathEscape(streamId)).String(),
			Name: file.Name,
		}
		if pttr == nil {
			if r, err := util.ParseTorrentTitle(file.Name); err == nil {
				pttr = r
			} else {
				pttLog.Warn("failed to parse", "error", err, "title", file.Name)
			}
		}
		if pttr != nil {
			if tpttr.Error() == nil {
				if pttr.Resolution == "" {
					pttr.Resolution = tpttr.Resolution
				}
				if pttr.Quality == "" {
					pttr.Quality = tpttr.Quality
				}
				if pttr.Codec == "" {
					pttr.Codec = tpttr.Codec
				}
				if len(pttr.HDR) == 0 {
					pttr.HDR = tpttr.HDR
				}
				if len(pttr.Audio) == 0 {
					pttr.Audio = tpttr.Audio
				}
				if len(pttr.Channels) == 0 {
					pttr.Channels = tpttr.Channels
				}
				if pttr.Group == "" {
					pttr.Group = tpttr.Group
				}
			}
			pttr.Size = util.ToSize(file.Size)
			if meta != nil && season != -1 && episode != -1 {
				for i := range meta.Videos {
					video := &meta.Videos[i]
					if video.Season == season && video.Episode == episode {
						pttr.Title = video.Name
						break
					}
				}
			}
			if _, err := streamTemplate.Execute(&stream, &stremio_transformer.StreamTemplateData{
				Result:    pttr,
				StoreCode: strings.ToUpper(matcher.StoreCode),
			}); err != nil {
				log.Error("failed to execute stream template", "error", err)
			}
		}
		res.Streams = append(res.Streams, stream)
	}

	SendResponse(w, r, 200, res)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	actionId := r.PathValue("actionId")
	idr, err := parseId(actionId)
	if err != nil {
		SendError(w, r, err)
		return
	}

	storeActionIdPrefix := getStoreActionIdPrefix(idr.getStoreCode())
	if !strings.HasPrefix(actionId, storeActionIdPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+actionId).Send(w, r)
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		store_video.Redirect("500", w, r)
		return
	}

	idPrefix := getIdPrefix(idr.getStoreCode())
	switch strings.TrimPrefix(actionId, storeActionIdPrefix) {
	case "clear_cache":
		cacheKey := getCatalogCacheKey(idPrefix, ctx.StoreAuthToken)
		catalogCache.Remove(cacheKey)
	}

	store_video.Redirect("200", w, r)
}

func handleStrem(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ud, err := getUserData(r)
	if err != nil {
		SendError(w, r, err)
		return
	}

	videoIdWithLink := r.PathValue("videoId")
	idr, err := parseId(videoIdWithLink)
	if err != nil {
		SendError(w, r, err)
		return
	}

	idPrefix := getIdPrefix(idr.getStoreCode())
	if !strings.HasPrefix(videoIdWithLink, idPrefix) {
		shared.ErrorBadRequest(r, "unsupported id: "+videoIdWithLink).Send(w, r)
		return
	}

	ctx, err := ud.GetRequestContext(r, idr)
	if err != nil || ctx.Store == nil {
		if err != nil {
			LogError(r, "failed to get request context", err)
		}
		shared.ErrorBadRequest(r, "failed to get request context").Send(w, r)
		return
	}

	videoId := strings.TrimPrefix(videoIdWithLink, idPrefix)
	videoId, link, _ := strings.Cut(videoId, ":")

	url := link

	if url == "" {
		ctx.Log.Warn("no matching file found for (" + videoIdWithLink + ")")
		store_video.Redirect("no_matching_file", w, r)
		return
	}

	stLink, err := shared.GenerateStremThruLink(r, ctx, url)
	if err != nil {
		LogError(r, "failed to generate stremthru link", err)
		store_video.Redirect("500", w, r)
		return
	}

	http.Redirect(w, r, stLink.Link, http.StatusFound)
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := server.GetReqCtx(r)
		ctx.Log = log.With("request_id", ctx.RequestId)
		next.ServeHTTP(w, r)
		ctx.RedactURLPathValues(r, "userData")
	})
}

func AddStremioStoreEndpoints(mux *http.ServeMux) {
	withCors := shared.Middleware(shared.EnableCORS)

	router := http.NewServeMux()

	router.HandleFunc("/{$}", handleRoot)

	router.HandleFunc("/manifest.json", withCors(handleManifest))
	router.HandleFunc("/{userData}/manifest.json", withCors(handleManifest))

	router.HandleFunc("/configure", handleConfigure)
	router.HandleFunc("/{userData}/configure", handleConfigure)

	router.HandleFunc("/{userData}/catalog/{contentType}/{idJson}", withCors(handleCatalog))
	router.HandleFunc("/{userData}/catalog/{contentType}/{id}/{extraJson}", withCors(handleCatalog))

	router.HandleFunc("/{userData}/meta/{contentType}/{idJson}", withCors(handleMeta))

	router.HandleFunc("/{userData}/stream/{contentType}/{idJson}", withCors(handleStream))

	router.HandleFunc("/{userData}/_/action/{actionId}", withCors(handleAction))
	router.HandleFunc("/{userData}/_/strem/{videoId}", withCors(handleStrem))

	mux.Handle("/stremio/store/", http.StripPrefix("/stremio/store", commonMiddleware(router)))
}
