package letterboxd

import (
	"errors"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
)

var letterboxdIdentifierCache = cache.NewCache[string](&cache.CacheConfig{
	Lifetime: 2 * time.Hour,
	Name:     "letterboxd:identifier",
})

func fetchLetterboxdIdentifier(urlPath string) (lid string, err error) {
	if letterboxdIdentifierCache.Get(urlPath, &lid) {
		return lid, nil
	}

	ctx := request.Ctx{}
	req, err := ctx.NewRequest(SITE_BASE_URL_PARSED, "HEAD", urlPath, nil, nil)
	if err != nil {
		return "", err
	}
	res, err := ctx.DoRequest(config.DefaultHTTPClient, req)
	if err != nil {
		return "", err
	}
	lid = res.Header.Get("X-Letterboxd-Identifier")
	if lid == "" {
		return "", errors.New("not found")
	}
	if err := letterboxdIdentifierCache.Add(urlPath, lid); err != nil {
		return "", err
	}
	return lid, nil
}

func FetchLetterboxdUserIdentifier(userName string) (string, error) {
	return fetchLetterboxdIdentifier("/" + userName + "/")
}

func FetchLetterboxdListIdentifier(userName, listSlug string) (string, error) {
	return fetchLetterboxdIdentifier("/" + userName + "/list/" + listSlug + "/")
}
