package endpoint

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/peer_token"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/alldebrid"
	"github.com/MunifTanjim/stremthru/store/debridlink"
	"github.com/MunifTanjim/stremthru/store/premiumize"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
	"github.com/MunifTanjim/stremthru/store/torbox"
	"github.com/golang-jwt/jwt/v5"
)

func getStoreName(r *http.Request) (store.StoreName, *core.StoreError) {
	name := r.Header.Get("X-StremThru-Store-Name")
	if name == "" {
		ctx := context.GetRequestContext(r)
		if ctx.IsProxyAuthorized {
			name = config.StoreAuthToken.GetPreferredStore(ctx.ProxyAuthUser)
			r.Header.Set("X-StremThru-Store-Name", name)
		}
	}
	if name == "" {
		return "", nil
	}
	return store.StoreName(name).Validate()
}

var adStore = alldebrid.NewStore()
var dlStore = debridlink.NewStoreClient()
var pmStore = premiumize.NewStoreClient(&premiumize.StoreClientConfig{})
var rdStore = realdebrid.NewStoreClient()
var tbStore = torbox.NewStoreClient()

func getStore(r *http.Request) (store.Store, error) {
	name, err := getStoreName(r)
	if err != nil {
		err.InjectReq(r)
		err.StatusCode = http.StatusBadRequest
		return nil, err
	}
	switch name {
	case store.StoreNameAlldebrid:
		return adStore, nil
	case store.StoreNameDebridLink:
		return dlStore, nil
	case store.StoreNamePremiumize:
		return pmStore, nil
	case store.StoreNameRealDebrid:
		return rdStore, nil
	case store.StoreNameTorBox:
		return tbStore, nil
	default:
		return nil, nil
	}
}

func getUser(ctx *context.RequestContext) (*store.User, error) {
	params := &store.GetUserParams{}
	params.APIKey = ctx.StoreAuthToken
	return ctx.Store.GetUser(params)
}

func handleStoreUser(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ctx := context.GetRequestContext(r)
	user, err := getUser(ctx)
	SendResponse(w, 200, user, err)
}

type AddMagnetPayload struct {
	Magnet string `json:"magnet"`
}

func checkMagnet(ctx *context.RequestContext, magnets []string) (*store.CheckMagnetData, error) {
	params := &store.CheckMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Magnets = magnets
	data, err := ctx.Store.CheckMagnet(params)
	if err == nil && data.Items == nil {
		data.Items = []store.CheckMagnetDataItem{}
	}
	return data, err
}

type TrackMagnetPayload struct {
	Hash   string             `json:"hash"`
	Files  []store.MagnetFile `json:"files"`
	IsMiss bool               `json:"is_miss"`
}

type TrackMagnetData struct {
}

func hadleStoreMagnetsTrack(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ctx := context.GetRequestContext(r)

	isValidToken, err := peer_token.IsValid(ctx.PeerToken)
	if err != nil {
		SendError(w, err)
		return
	}
	if !isValidToken {
		shared.ErrorUnauthorized(r).Send(w)
		return
	}

	payload := &TrackMagnetPayload{}
	if err := shared.ReadRequestBodyJSON(r, payload); err != nil {
		SendError(w, err)
		return
	}

	buddy.TrackMagnet(ctx.Store, payload.Hash, payload.Files, payload.IsMiss, ctx.StoreAuthToken)

	SendResponse(w, 202, &TrackMagnetData{}, nil)
}

func handleStoreMagnetsCheck(w http.ResponseWriter, r *http.Request) {
	if shared.IsMethod(r, http.MethodPost) {
		hadleStoreMagnetsTrack(w, r)
		return
	}

	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	queryParams := r.URL.Query()
	magnet, ok := queryParams["magnet"]
	if !ok {
		shared.ErrorBadRequest(r, "missing magnet").Send(w)
		return
	}

	magnets := []string{}
	for _, m := range magnet {
		magnets = append(magnets, strings.FieldsFunc(m, func(r rune) bool {
			return r == ','
		})...)
	}

	if len(magnets) == 0 {
		shared.ErrorBadRequest(r, "missing magnet").Send(w)
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := checkMagnet(ctx, magnets)
	if err == nil && data != nil {
		for _, item := range data.Items {
			item.Hash = strings.ToLower(item.Hash)
		}
	}
	SendResponse(w, 200, data, err)
}

func listMagnets(ctx *context.RequestContext, r *http.Request) (*store.ListMagnetsData, error) {
	queryParams := r.URL.Query()
	limit, err := GetQueryInt(queryParams, "limit", 100)
	if err != nil {
		return nil, shared.ErrorBadRequest(r, err.Error())
	}
	if limit > 500 {
		limit = 500
	}
	offset, err := GetQueryInt(queryParams, "offset", 0)
	if err != nil {
		return nil, shared.ErrorBadRequest(r, err.Error())
	}

	params := &store.ListMagnetsParams{
		Limit:  limit,
		Offset: offset,
	}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.ListMagnets(params)

	if err == nil && data.Items == nil {
		data.Items = []store.ListMagnetsDataItem{}
	}
	return data, err
}

func handleStoreMagnetsList(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := listMagnets(ctx, r)
	if err == nil && data != nil {
		for _, item := range data.Items {
			item.Hash = strings.ToLower(item.Hash)
		}
	}
	SendResponse(w, 200, data, err)
}

func addMagnet(ctx *context.RequestContext, magnet string) (*store.AddMagnetData, error) {
	params := &store.AddMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Magnet = magnet
	if ctx.ClientIP != "" {
		params.ClientIP = ctx.ClientIP
	}
	data, err := ctx.Store.AddMagnet(params)
	if err == nil {
		buddy.TrackMagnet(ctx.Store, data.Hash, data.Files, data.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)
	}
	return data, err
}

func handleStoreMagnetAdd(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	payload := &AddMagnetPayload{}
	err := shared.ReadRequestBodyJSON(r, payload)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := addMagnet(ctx, payload.Magnet)
	if err == nil && data != nil {
		data.Hash = strings.ToLower(data.Hash)
	}
	SendResponse(w, 201, data, err)
}

func handleStoreMagnets(w http.ResponseWriter, r *http.Request) {
	if shared.IsMethod(r, http.MethodGet) {
		handleStoreMagnetsList(w, r)
		return
	}

	if shared.IsMethod(r, http.MethodPost) {
		handleStoreMagnetAdd(w, r)
		return
	}

	shared.ErrorMethodNotAllowed(r).Send(w)
}

func getMagnet(ctx *context.RequestContext, magnetId string) (*store.GetMagnetData, error) {
	params := &store.GetMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Id = magnetId
	data, err := ctx.Store.GetMagnet(params)
	if err == nil {
		buddy.TrackMagnet(ctx.Store, data.Hash, data.Files, data.Status != store.MagnetStatusDownloaded, ctx.StoreAuthToken)
	}
	return data, err
}

func handleStoreMagnetGet(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	magnetId := r.PathValue("magnetId")
	if magnetId == "" {
		shared.ErrorBadRequest(r, "missing magnetId").Send(w)
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := getMagnet(ctx, magnetId)
	if err == nil && data != nil {
		data.Hash = strings.ToLower(data.Hash)
	}
	SendResponse(w, 200, data, err)
}

func removeMagnet(ctx *context.RequestContext, magnetId string) (*store.RemoveMagnetData, error) {
	params := &store.RemoveMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Id = magnetId
	return ctx.Store.RemoveMagnet(params)
}

func handleStoreMagnetRemove(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodDelete) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	magnetId := r.PathValue("magnetId")
	if magnetId == "" {
		shared.ErrorBadRequest(r, "missing magnetId").Send(w)
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := removeMagnet(ctx, magnetId)
	SendResponse(w, 200, data, err)
}

func handleStoreMagnet(w http.ResponseWriter, r *http.Request) {
	if shared.IsMethod(r, http.MethodGet) {
		handleStoreMagnetGet(w, r)
		return
	}

	if shared.IsMethod(r, http.MethodDelete) {
		handleStoreMagnetRemove(w, r)
		return
	}

	shared.ErrorMethodNotAllowed(r).Send(w)
}

type GenerateLinkPayload struct {
	Link string `json:"link"`
}

type GenerateLinkJWTData struct {
	EncLink   string `json:"enc_link"`
	EncFormat string `json:"enc_format"`
}

func extractReqScheme(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")

	if scheme == "" {
		scheme = r.URL.Scheme
	}

	if scheme == "" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}

	return scheme
}

func extractReqHost(r *http.Request) string {
	host := r.Header.Get("X-Forwarded-Host")

	if host == "" {
		host = r.Host
	}

	return host
}

func createSecureLink(r *http.Request, ctx *context.RequestContext, link string) (string, error) {
	encryptedLink, err := core.Encrypt(ctx.ProxyAuthPassword, link)
	if err != nil {
		return "", err
	}

	secureLink := (&url.URL{
		Scheme: extractReqScheme(r),
		Host:   extractReqHost(r),
	}).JoinPath("/v0/store/link/access")

	token, err := core.CreateJWT(ctx.ProxyAuthPassword, core.JWTClaims[GenerateLinkJWTData]{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "stremthru",
			Subject:   ctx.ProxyAuthUser,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		Data: &GenerateLinkJWTData{
			EncLink:   encryptedLink,
			EncFormat: core.EncryptionFormat,
		},
	})

	if err != nil {
		return "", err
	}

	secureLink = secureLink.JoinPath(token)

	return secureLink.String(), nil

}

func generateLink(r *http.Request, ctx *context.RequestContext, link string) (*store.GenerateLinkData, error) {
	params := &store.GenerateLinkParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Link = link
	if ctx.ClientIP != "" {
		params.ClientIP = ctx.ClientIP
	}

	data, err := ctx.Store.GenerateLink(params)
	if err != nil {
		return nil, err
	}

	if ctx.IsProxyAuthorized && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(ctx.Store.GetName())) {
		secureLink, err := createSecureLink(r, ctx, data.Link)
		if err != nil {
			return nil, err
		}

		data.Link = secureLink
	}

	return data, nil
}

func handleStoreLinkGenerate(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	payload := &GenerateLinkPayload{}
	err := shared.ReadRequestBodyJSON(r, payload)
	if err != nil {
		SendError(w, err)
		return
	}

	ctx := context.GetRequestContext(r)
	link, err := generateLink(r, ctx, payload.Link)
	SendResponse(w, 200, link, err)
}

func getUserSecretFromJWT(t *jwt.Token) (string, []byte, error) {
	username, err := t.Claims.GetSubject()
	if err != nil {
		return "", nil, err
	}
	password := config.ProxyAuthPassword.GetPassword(username)
	return password, []byte(password), nil
}

var tokenLinkCache = func() cache.Cache[string] {
	return cache.NewCache[string](&cache.CacheConfig{
		Name:     "endpoint:store:tokenLink",
		Lifetime: 15 * time.Minute,
	})
}()

func handleStoreLinkAccess(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodGet) && !shared.IsMethod(r, http.MethodHead) {
		shared.ErrorMethodNotAllowed(r).Send(w)
		return
	}

	encodedToken := r.PathValue("token")
	if encodedToken == "" {
		shared.ErrorBadRequest(r, "missing token").Send(w)
		return
	}

	link := ""
	if ok := tokenLinkCache.Get(encodedToken, &link); ok {
		shared.ProxyResponse(w, r, link)
		return
	}

	claims := &core.JWTClaims[GenerateLinkJWTData]{}
	token, err := core.ParseJWT(func(t *jwt.Token) (interface{}, error) {
		_, secret, err := getUserSecretFromJWT(t)
		return secret, err
	}, encodedToken, claims)

	if err != nil {
		SendError(w, err)
		return
	}

	secret, _, err := getUserSecretFromJWT(token)
	if err != nil {
		SendError(w, err)
		return
	}

	link, err = core.Decrypt(secret, claims.Data.EncLink)
	if err != nil {
		SendError(w, err)
		return
	}

	tokenLinkCache.Add(encodedToken, link)

	shared.ProxyResponse(w, r, link)
}

func AddStoreEndpoints(mux *http.ServeMux) {
	withContext := Middleware(ProxyAuthContext)
	withStore := Middleware(ProxyAuthContext, StoreContext, StoreRequired)

	mux.HandleFunc("/v0/store/user", withStore(handleStoreUser))
	mux.HandleFunc("/v0/store/magnets", withStore(handleStoreMagnets))
	mux.HandleFunc("/v0/store/magnets/check", withStore(handleStoreMagnetsCheck))
	mux.HandleFunc("/v0/store/magnets/{magnetId}", withStore(handleStoreMagnet))
	mux.HandleFunc("/v0/store/link/generate", withStore(handleStoreLinkGenerate))
	mux.HandleFunc("/v0/store/link/access/{token}", withContext(handleStoreLinkAccess))
}
