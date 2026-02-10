package shared

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	storecontext "github.com/MunifTanjim/stremthru/internal/store/context"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/alldebrid"
	"github.com/MunifTanjim/stremthru/store/debrider"
	"github.com/MunifTanjim/stremthru/store/debridlink"
	"github.com/MunifTanjim/stremthru/store/easydebrid"
	"github.com/MunifTanjim/stremthru/store/offcloud"
	"github.com/MunifTanjim/stremthru/store/pikpak"
	"github.com/MunifTanjim/stremthru/store/premiumize"
	"github.com/MunifTanjim/stremthru/store/qbittorrent"
	"github.com/MunifTanjim/stremthru/store/realdebrid"
	"github.com/MunifTanjim/stremthru/store/stremthru"
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
var stStore = stremthru.NewStoreClient(&stremthru.StoreClientConfig{})
var tbStore = torbox.NewStoreClient(&torbox.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("torbox")),
	UserAgent:  config.StoreClientUserAgent,
})
var qbStore = qbittorrent.NewStoreClient(&qbittorrent.StoreClientConfig{
	HTTPClient: config.GetHTTPClient(config.StoreTunnel.GetTypeForAPI("qbittorrent")),
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
	case store.StoreNameStremThru:
		return stStore
	case store.StoreNameTorBox:
		return tbStore
	case store.StoreNameQBittorrent:
		return qbStore
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
	case store.StoreCodeStremThru:
		return stStore
	case store.StoreCodeTorBox:
		return tbStore
	case store.StoreCodeQBittorrent:
		return qbStore
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

// ProxyLinkInfo holds all data extracted from a proxy link token.
type ProxyLinkInfo struct {
	User       string
	Link       string
	Headers    map[string]string
	TunnelType config.TunnelType
}

func CreateProxyLink(r *http.Request, link string, headers map[string]string, tunnelType config.TunnelType, expiresIn time.Duration, user, password string, shouldEncrypt bool, filename string) (string, error) {
	var encodedToken string

	if !shouldEncrypt && expiresIn == 0 {
		pld := proxyLinkData{
			User:    user + ":" + password,
			Value:   link,
			Headers: headers,
			TunT:    tunnelType,
		}
		blob, err := json.Marshal(pld)
		if err != nil {
			return "", err
		}
		encodedToken = "base64." + util.Base64EncodeByte(blob)
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
			encLink = util.Base64Encode(linkBlob)
			encFormat = "base64"
		}

		tokenData := &proxyLinkTokenData{
			EncLink:    encLink,
			EncFormat:  encFormat,
			TunnelType: tunnelType,
		}

		claims := core.JWTClaims[proxyLinkTokenData]{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:  "stremthru",
				Subject: user,
			},
			Data: tokenData,
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

func ProxyWrapLink(r *http.Request, ctx *storecontext.Context, link string, filename string) (string, error) {
	storeName := string(ctx.Store.GetName())
	if config.StoreContentProxy.IsEnabled(storeName) && ctx.StoreAuthToken == config.StoreAuthToken.GetToken(ctx.ProxyAuthUser, storeName) {
		if ctx.IsProxyAuthorized {
			tunnelType := config.StoreTunnel.GetTypeForStream(string(ctx.Store.GetName()))
			proxyLink, err := CreateProxyLink(r, link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, filename)
			if err != nil {
				return link, err
			}

			return proxyLink, nil
		}
	}
	return link, nil
}

func GenerateStremThruLink(r *http.Request, ctx *storecontext.Context, link string, filename string) (*store.GenerateLinkData, error) {
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

			proxyLink, err := CreateProxyLink(r, data.Link, nil, tunnelType, 12*time.Hour, ctx.ProxyAuthUser, ctx.ProxyAuthPassword, true, filename)
			if err != nil {
				return nil, err
			}

			// For qBittorrent, append torrent hash and file index as query
			// params so the proxy can check download progress.
			if ctx.Store.GetName() == store.StoreNameQBittorrent {
				if hash, fileIdx, err := qbittorrent.ParseLockedFileLink(link); err == nil {
					sep := "?"
					if strings.Contains(proxyLink, "?") {
						sep = "&"
					}
					proxyLink += sep + "qbit_hash=" + hash + "&qbit_fidx=" + strconv.Itoa(fileIdx)
				}
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
	password = config.Auth.GetPassword(user)
	return user, password, nil
}

func UnwrapProxyLinkToken(encodedToken string) (*ProxyLinkInfo, error) {
	proxyLink := &proxyLinkData{}
	if found := proxyLinkTokenCache.Get(encodedToken, proxyLink); found {
		return proxyLink.toInfo(), nil
	}

	if encodedBlob, ok := strings.CutPrefix(encodedToken, "base64."); ok {
		blob, err := util.Base64DecodeToByte(encodedBlob)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(blob, proxyLink); err != nil {
			return nil, err
		}
		user, pass, _ := strings.Cut(proxyLink.User, ":")
		if pass != config.Auth.GetPassword(user) {
			err := core.NewAPIError("unauthorized")
			err.StatusCode = http.StatusUnauthorized
			return nil, err
		}
		proxyLink.User = user
	} else {
		claims := &core.JWTClaims[proxyLinkTokenData]{}
		password := ""
		var user string
		var err error
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

			return nil, err
		}

		var linkBlob string
		if claims.Data.EncFormat == "base64" {
			blob, err := util.Base64Decode(claims.Data.EncLink)
			if err != nil {
				return nil, err
			}
			linkBlob = blob
		} else {
			blob, err := core.Decrypt(password, claims.Data.EncLink)
			if err != nil {
				return nil, err
			}
			linkBlob = blob
		}

		link, headersBlob, hasHeaders := strings.Cut(linkBlob, "\n")

		proxyLink.User = user
		proxyLink.TunT = claims.Data.TunnelType
		proxyLink.Value = link

		if hasHeaders {
			proxyLink.Headers = map[string]string{}
			for header := range strings.SplitSeq(headersBlob, "\n") {
				if k, v, ok := strings.Cut(header, ": "); ok {
					proxyLink.Headers[k] = v
				}
			}
		}
	}

	proxyLinkTokenCache.Add(encodedToken, *proxyLink)

	return proxyLink.toInfo(), nil
}

func (p *proxyLinkData) toInfo() *ProxyLinkInfo {
	return &ProxyLinkInfo{
		User:       p.User,
		Link:       p.Value,
		Headers:    p.Headers,
		TunnelType: p.TunT,
	}
}

// GetQbitFileProgress returns the download progress for a qBittorrent file.
// Uses the user's configured store auth token to access the qBittorrent API.
func GetQbitFileProgress(user string, hash string, fileIndex int) (*qbittorrent.FileProgressInfo, error) {
	apiKey := config.StoreAuthToken.GetToken(user, "qbittorrent")
	if apiKey == "" {
		return nil, errors.New("no qBittorrent API key for user")
	}
	return qbStore.GetFileProgress(apiKey, hash, fileIndex)
}

// GetQbitSafeBytes returns the contiguous-from-start safe byte boundary for
// a qBittorrent file. Unlike GetQbitFileProgress (total progress), this tracks
// the sequential download frontier for accurate streaming pacing.
func GetQbitSafeBytes(user, hash string, fileIndex int) (safeBytes int64, fileSize int64, done bool, err error) {
	apiKey := config.StoreAuthToken.GetToken(user, "qbittorrent")
	if apiKey == "" {
		return 0, 0, false, errors.New("no qBittorrent API key for user")
	}
	return qbStore.GetSafeBytes(apiKey, hash, fileIndex)
}

// IsQbitFileRangeAvailable checks whether the byte range [start, end] within
// a qBittorrent file is fully downloaded at the piece level.
func IsQbitFileRangeAvailable(user, hash string, fileIndex int, start, end int64) (bool, error) {
	apiKey := config.StoreAuthToken.GetToken(user, "qbittorrent")
	if apiKey == "" {
		return false, errors.New("no qBittorrent API key for user")
	}
	return qbStore.IsFileRangeAvailable(apiKey, hash, fileIndex, start, end)
}
