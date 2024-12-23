package shared

import (
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/alldebrid"
	"github.com/MunifTanjim/stremthru/store/debridlink"
	"github.com/MunifTanjim/stremthru/store/offcloud"
	"github.com/MunifTanjim/stremthru/store/premiumize"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
	"github.com/MunifTanjim/stremthru/store/torbox"
	"github.com/golang-jwt/jwt/v5"
)

var adStore = alldebrid.NewStore()
var dlStore = debridlink.NewStoreClient()
var pmStore = premiumize.NewStoreClient(&premiumize.StoreClientConfig{})
var ocStore = offcloud.NewStoreClient()
var rdStore = realdebrid.NewStoreClient()
var tbStore = torbox.NewStoreClient()

func GetStore(name string) store.Store {
	switch store.StoreName(name) {
	case store.StoreNameAlldebrid:
		return adStore
	case store.StoreNameDebridLink:
		return dlStore
	case store.StoreNameOffcloud:
		return ocStore
	case store.StoreNamePremiumize:
		return pmStore
	case store.StoreNameRealDebrid:
		return rdStore
	case store.StoreNameTorBox:
		return tbStore
	default:
		return nil
	}
}

type proxyLinkTokenData struct {
	EncLink   string `json:"enc_link"`
	EncFormat string `json:"enc_format"`
}

func CreateProxyLink(r *http.Request, ctx *context.RequestContext, link string) (string, error) {
	if !ctx.IsProxyAuthorized || ctx.StoreAuthToken != config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, string(ctx.Store.GetName())) {
		return link, nil
	}

	encryptedLink, err := core.Encrypt(ctx.ProxyAuthPassword, link)
	if err != nil {
		return "", err
	}

	proxyLink := ExtractRequestBaseURL(r).JoinPath("/v0/store/link/access")

	token, err := core.CreateJWT(ctx.ProxyAuthPassword, core.JWTClaims[proxyLinkTokenData]{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "stremthru",
			Subject:   ctx.ProxyAuthUser,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(6 * time.Hour)),
		},
		Data: &proxyLinkTokenData{
			EncLink:   encryptedLink,
			EncFormat: core.EncryptionFormat,
		},
	})

	if err != nil {
		return "", err
	}

	proxyLink = proxyLink.JoinPath(token)

	return proxyLink.String(), nil
}

func GenerateStremThruLink(r *http.Request, ctx *context.RequestContext, link string) (*store.GenerateLinkData, error) {
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

	proxyLink, err := CreateProxyLink(r, ctx, data.Link)
	if err != nil {
		return nil, err
	}

	data.Link = proxyLink

	return data, nil
}

var proxyLinkTokenCache = func() cache.Cache[string] {
	return cache.NewCache[string](&cache.CacheConfig{
		Name:     "store:proxyLinkToken",
		Lifetime: 30 * time.Minute,
	})
}()

func getUserSecretFromJWT(t *jwt.Token) (string, []byte, error) {
	username, err := t.Claims.GetSubject()
	if err != nil {
		return "", nil, err
	}
	password := config.ProxyAuthPassword.GetPassword(username)
	return password, []byte(password), nil
}

func UnwrapProxyLinkToken(encodedToken string) (string, error) {
	link := ""
	if found := proxyLinkTokenCache.Get(encodedToken, &link); found {
		return link, nil
	}

	claims := &core.JWTClaims[proxyLinkTokenData]{}
	token, err := core.ParseJWT(func(t *jwt.Token) (interface{}, error) {
		_, secret, err := getUserSecretFromJWT(t)
		return secret, err
	}, encodedToken, claims)

	if err != nil {
		return "", err
	}

	secret, _, err := getUserSecretFromJWT(token)
	if err != nil {
		return "", err
	}

	link, err = core.Decrypt(secret, claims.Data.EncLink)
	if err != nil {
		return "", err
	}

	proxyLinkTokenCache.Add(encodedToken, link)

	return link, nil
}
