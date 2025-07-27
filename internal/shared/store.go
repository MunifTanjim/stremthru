package shared

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/alldebrid"
	"github.com/MunifTanjim/stremthru/store/debrider"
	"github.com/MunifTanjim/stremthru/store/debridlink"
	"github.com/MunifTanjim/stremthru/store/easydebrid"
	"github.com/MunifTanjim/stremthru/store/offcloud"
	"github.com/MunifTanjim/stremthru/store/pikpak"
	"github.com/MunifTanjim/stremthru/store/premiumize"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
	"github.com/MunifTanjim/stremthru/store/torbox"
	"github.com/golang-jwt/jwt/v5"
)

var adStore = alldebrid.NewStoreClient(&alldebrid.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("alldebrid")),
	UserAgent:  config.StoreClientUserAgent,
})
var drStore = debrider.NewStoreClient(&debrider.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("debrider")),
	UserAgent:  config.StoreClientUserAgent,
})
var dlStore = debridlink.NewStoreClient(&debridlink.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("debridlink")),
	UserAgent:  config.StoreClientUserAgent,
})
var edStore = easydebrid.NewStoreClient(&easydebrid.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("easydebrid")),
	UserAgent:  config.StoreClientUserAgent,
})
var pmStore = premiumize.NewStoreClient(&premiumize.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("premiumize")),
	UserAgent:  config.StoreClientUserAgent,
})
var ppStore = pikpak.NewStoreClient(&pikpak.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("pikpak")),
	UserAgent:  config.StoreClientUserAgent,
})
var ocStore = offcloud.NewStoreClient(&offcloud.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("offcloud")),
	UserAgent:  config.StoreClientUserAgent,
})
var rdStore = realdebrid.NewStoreClient(&realdebrid.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("realdebrid")),
	UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
})
var tbStore = torbox.NewStoreClient(&torbox.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("torbox")),
	UserAgent:  config.StoreClientUserAgent,
})

func GetStore(name string) store.Store {
	switch store.StoreName(name) {
	case store.StoreNameAlldebrid:
		return adStore
	case store.StoreNameDebrider:
		return drStore
	case store.StoreNameDebridLink:
		return dlStore
	case store.StoreNameEasyDebrid:
		return edStore
	case store.StoreNameOffcloud:
		return ocStore
	case store.StoreNamePikPak:
		return ppStore
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

func GetStoreByCode(code string) store.Store {
	switch store.StoreCode(code) {
	case store.StoreCodeAllDebrid:
		return adStore
	case store.StoreCodeDebrider:
		return drStore
	case store.StoreCodeDebridLink:
		return dlStore
	case store.StoreCodeEasyDebrid:
		return edStore
	case store.StoreCodeOffcloud:
		return ocStore
	case store.StoreCodePikPak:
		return ppStore
	case store.StoreCodePremiumize:
		return pmStore
	case store.StoreCodeRealDebrid:
		return rdStore
	case store.StoreCodeTorBox:
		return tbStore
	default:
		return nil
	}
}

type proxyLinkTokenData struct {
	EncLink    string            `json:"enc_link"`
	EncFormat  string            `json:"enc_format"`
	TunnelType config.TunnelType `json:"tunt,omitempty"`
}

type proxyLinkData struct {
	User    string            `json:"u"`
	Value   string            `json:"v"`
	Headers map[string]string `json:"reqh,omitempty"`
	TunT    config.TunnelType `json:"tunt,omitempty"`
}

func CreateProxyLink(r *http.Request, link string, headers map[string]string, tunnelType config.TunnelType, expiresIn time.Duration, user, password string, shouldEncrypt bool, filename string) (string, error) {
	var encodedToken string

	if !shouldEncrypt && expiresIn == 0 {
		blob, err := json.Marshal(proxyLinkData{
			User:    user + ":" + password,
			Value:   link,
			Headers: headers,
			TunT:    tunnelType,
		})
		if err != nil {
			return "", err
		}
		encodedToken = "base64." + core.Base64EncodeByte(blob)
	} else {
		linkBlob := link
		if headers != nil {
			for k, v := range headers {
				linkBlob += "\n" + k + ": " + v
			}
		}

		var encLink string
		var encFormat string

		if shouldEncrypt {
			encryptedLink, err := core.Encrypt(password, linkBlob)
			if err != nil {
				return "", err
			}
			encLink = encryptedLink
			encFormat = core.EncryptionFormat
		} else {
			encLink = core.Base64Encode(linkBlob)
			encFormat = "base64"
		}

		claims := core.JWTClaims[proxyLinkTokenData]{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:  "stremthru",
				Subject: user,
			},
			Data: &proxyLinkTokenData{
				EncLink:    encLink,
				EncFormat:  encFormat,
				TunnelType: tunnelType,
			},
		}
		if expiresIn != 0 {
			claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(expiresIn))
		}
		token, err := core.CreateJWT(password, claims)
		if err != nil {
			return "", err
		}
		encodedToken = token
	}

	pLink := ExtractRequestBaseURL(r).JoinPath("/v0/proxy", encodedToken)

	if filename == "" {
		filename, _, _ = strings.Cut(filepath.Base(link), "?")
	}
	if filename != "" {
		pLink = pLink.JoinPath(filename)
	}

	return pLink.String(), nil
}

func GenerateStremThruLink(r *http.Request, ctx *context.StoreContext, link string) (*store.GenerateLinkData, error) {
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

	storeName := string(ctx.Store.GetName())
	if config.StoreContentProxy.IsEnabled(storeName) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, storeName) {
		if ctx.IsProxyAuthorized {
			tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
			proxyLink, err := CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, "")
			if err != nil {
				return nil, err
			}

			data.Link = proxyLink
		}
	}

	return data, nil
}

var proxyLinkTokenCache = func() cache.Cache[proxyLinkData] {
	return cache.NewCache[proxyLinkData](&cache.CacheConfig{
		Name:     "store:proxyLinkToken",
		Lifetime: 30 * time.Minute,
	})
}()

func getUserCredsFromJWT(t *jwt.Token) (user, password string, err error) {
	user, err = t.Claims.GetSubject()
	if err != nil {
		return "", "", err
	}
	password = config.ProxyAuthPassword.GetPassword(user)
	return user, password, nil
}

func UnwrapProxyLinkToken(encodedToken string) (user string, link string, headers map[string]string, tunnelType config.TunnelType, err error) {
	proxyLink := &proxyLinkData{}
	if found := proxyLinkTokenCache.Get(encodedToken, proxyLink); found {
		return proxyLink.User, proxyLink.Value, proxyLink.Headers, proxyLink.TunT, nil
	}

	if strings.HasPrefix(encodedToken, "base64.") {
		blob, err := core.Base64DecodeToByte(strings.TrimPrefix(encodedToken, "base64."))
		if err != nil {
			return "", "", nil, "", err
		}
		if err := json.Unmarshal(blob, proxyLink); err != nil {
			return "", "", nil, "", err
		}
		user, pass, _ := strings.Cut(proxyLink.User, ":")
		if pass != config.ProxyAuthPassword.GetPassword(user) {
			err := core.NewAPIError("unauthorized")
			err.StatusCode = http.StatusUnauthorized
			return "", "", nil, "", err
		}
		proxyLink.User = user
	} else {
		claims := &core.JWTClaims[proxyLinkTokenData]{}
		password := ""
		_, err = core.ParseJWT(func(t *jwt.Token) (any, error) {
			user, password, err = getUserCredsFromJWT(t)
			return []byte(password), err
		}, encodedToken, claims)

		if err != nil {
			if errors.Is(err, jwt.ErrTokenInvalidClaims) {
				rerr := core.NewAPIError("unauthorized")
				rerr.StatusCode = http.StatusUnauthorized
				rerr.Cause = err
				err = rerr
			}

			return "", "", nil, "", err
		}

		var linkBlob string
		if claims.Data.EncFormat == "base64" {
			blob, err := core.Base64Decode(claims.Data.EncLink)
			if err != nil {
				return "", "", nil, "", err
			}
			linkBlob = blob
		} else {
			blob, err := core.Decrypt(password, claims.Data.EncLink)
			if err != nil {
				return "", "", nil, "", err
			}
			linkBlob = blob
		}

		link, headersBlob, hasHeaders := strings.Cut(linkBlob, "\n")

		proxyLink.User = user
		proxyLink.TunT = claims.Data.TunnelType
		proxyLink.Value = link

		if hasHeaders {
			proxyLink.Headers = map[string]string{}
			for _, header := range strings.Split(headersBlob, "\n") {
				if k, v, ok := strings.Cut(header, ": "); ok {
					proxyLink.Headers[k] = v
				}
			}
		}
	}

	proxyLinkTokenCache.Add(encodedToken, *proxyLink)

	return proxyLink.User, proxyLink.Value, proxyLink.Headers, proxyLink.TunT, nil
}
