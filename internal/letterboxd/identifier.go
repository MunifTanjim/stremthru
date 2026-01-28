package letterboxd

import (
	"errors"
	"fmt"
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
	ctx := request.Ctx{}
	req, err := ctx.NewRequest(SITE_BASE_URL_PARSED, "HEAD", urlPath, nil, nil)
	if err != nil {
		return "", err
	}
	res, err := ctx.DoRequest(config.DefaultHTTPClient, req)
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("status code %d", res.StatusCode)
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

func FetchLetterboxdUserIdentifier(userName string) (lid string, err error) {
	urlPath := "/" + userName + "/"

	if letterboxdIdentifierCache.Get(urlPath, &lid) {
		return lid, nil
	}

	if id, err := GetUserIdByName(userName); err != nil {
		log.Warn("failed to get user id from database", "error", err, "user_name", userName)
	} else if id != "" {
		letterboxdIdentifierCache.Add(urlPath, lid)
		return id, nil
	}
	return fetchLetterboxdIdentifier(urlPath)
}

func FetchLetterboxdListIdentifier(userName, listSlug string) (lid string, err error) {
	urlPath := "/" + userName + "/list/" + listSlug + "/"

	if letterboxdIdentifierCache.Get(urlPath, &lid) {
		return lid, nil
	}

	if id, err := GetListIdByUserNameAndSlug(userName, listSlug); err != nil {
		log.Warn("failed to get list id from database", "error", err, "user_name", userName, "slug", listSlug)
	} else if id != "" {
		letterboxdIdentifierCache.Add(urlPath, lid)
		return id, nil
	}
	return fetchLetterboxdIdentifier(urlPath)
}
