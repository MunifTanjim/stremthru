package endpoint

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
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

func getStoreAuthToken(r *http.Request) string {
	authHeader := r.Header.Get("X-StremThru-Store-Authorization")
	if authHeader == "" {
		authHeader = r.Header.Get("Authorization")
	}
	if authHeader == "" {
		ctx := context.GetRequestContext(r)
		if ctx.IsProxyAuthorized && ctx.Store != nil {
			if token := config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(ctx.Store.GetName())); token != "" {
				return token
			}
		}
	}
	_, token, _ := strings.Cut(authHeader, " ")
	return strings.TrimSpace(token)
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
	if !IsMethod(r, http.MethodGet) {
		SendError(w, ErrorMethodNotAllowed(r))
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

func handleStoreMagnetsCheck(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	queryParams := r.URL.Query()
	magnet, ok := queryParams["magnet"]
	if !ok {
		SendError(w, ErrorBadRequest(r, "missing magnet"))
		return
	}

	magnets := []string{}
	for _, m := range magnet {
		magnets = append(magnets, strings.FieldsFunc(m, func(r rune) bool {
			return r == ','
		})...)
	}

	if len(magnets) == 0 {
		SendError(w, ErrorBadRequest(r, "missing magnet"))
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

func listMagnets(ctx *context.RequestContext) (*store.ListMagnetsData, error) {
	params := &store.ListMagnetsParams{}
	params.APIKey = ctx.StoreAuthToken
	data, err := ctx.Store.ListMagnets(params)
	if err == nil && data.Items == nil {
		data.Items = []store.ListMagnetsDataItem{}
	}
	return data, err
}

func handleStoreMagnetsList(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := listMagnets(ctx)
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
	return ctx.Store.AddMagnet(params)
}

func handleStoreMagnetAdd(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodPost) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	payload := &AddMagnetPayload{}
	err := ReadJSONPayload(r, payload)
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
	if IsMethod(r, http.MethodGet) {
		handleStoreMagnetsList(w, r)
		return
	}

	if IsMethod(r, http.MethodPost) {
		handleStoreMagnetAdd(w, r)
		return
	}

	SendError(w, ErrorMethodNotAllowed(r))
}

func getMagnet(ctx *context.RequestContext, magnetId string) (*store.GetMagnetData, error) {
	params := &store.GetMagnetParams{}
	params.APIKey = ctx.StoreAuthToken
	params.Id = magnetId
	return ctx.Store.GetMagnet(params)
}

func handleStoreMagnetGet(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	magnetId := r.PathValue("magnetId")
	if magnetId == "" {
		SendError(w, ErrorBadRequest(r, "missing magnetId"))
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
	if !IsMethod(r, http.MethodDelete) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	magnetId := r.PathValue("magnetId")
	if magnetId == "" {
		SendError(w, ErrorBadRequest(r, "missing magnetId"))
		return
	}

	ctx := context.GetRequestContext(r)
	data, err := removeMagnet(ctx, magnetId)
	SendResponse(w, 200, data, err)
}

func handleStoreMagnet(w http.ResponseWriter, r *http.Request) {
	if IsMethod(r, http.MethodGet) {
		handleStoreMagnetGet(w, r)
		return
	}

	if IsMethod(r, http.MethodDelete) {
		handleStoreMagnetRemove(w, r)
		return
	}

	SendError(w, ErrorMethodNotAllowed(r))
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
	if !IsMethod(r, http.MethodPost) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	payload := &GenerateLinkPayload{}
	err := ReadJSONPayload(r, payload)
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

var tokenLinkCache = func() core.Cache[string, string] {
	return core.NewCache[string, string](&core.CacheConfig[string]{
		Name:     "endpoint:store:tokenLink",
		HashKey:  core.CacheHashKeyString,
		Lifetime: 15 * time.Minute,
	})
}()

func handleStoreLinkAccess(w http.ResponseWriter, r *http.Request) {
	if !IsMethod(r, http.MethodGet) && !IsMethod(r, http.MethodHead) {
		SendError(w, ErrorMethodNotAllowed(r))
		return
	}

	encodedToken := r.PathValue("token")
	if encodedToken == "" {
		SendError(w, ErrorBadRequest(r, "missing token"))
		return
	}

	if link, ok := tokenLinkCache.Get(encodedToken); ok {
		ProxyToLink(w, r, link)
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

	link, err := core.Decrypt(secret, claims.Data.EncLink)
	if err != nil {
		SendError(w, err)
		return
	}

	tokenLinkCache.Add(encodedToken, link)

	ProxyToLink(w, r, link)
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
