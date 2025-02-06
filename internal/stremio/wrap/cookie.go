package stremio_wrap

import (
	"net/http"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type CookieValue struct {
	url.Values
	IsExpired bool
}

func (cv *CookieValue) User() string {
	return cv.Get("user")
}

func (cv *CookieValue) Pass() string {
	return cv.Get("pass")
}

const COOKIE_NAME = "stremio.wrap.auth"
const COOKIE_PATH = "/stremio/wrap/"

func setCookie(w http.ResponseWriter, user string, pass string) {
	value := &url.Values{
		"user": []string{user},
		"pass": []string{pass},
	}
	cookie := &http.Cookie{
		Name:     COOKIE_NAME,
		Value:    value.Encode(),
		HttpOnly: true,
		Path:     COOKIE_PATH,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

func unsetCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Expires: time.Unix(0, 0),
		Path:    COOKIE_PATH,
	})
}

func getCookieValue(w http.ResponseWriter, r *http.Request) (*CookieValue, error) {
	cookie, err := r.Cookie(COOKIE_NAME)
	value := &CookieValue{}
	if err != nil {
		if err != http.ErrNoCookie {
			return value, err
		}
		value.IsExpired = true
		return value, nil
	}

	v, err := url.ParseQuery(cookie.Value)
	if err != nil {
		core.LogError("[stremio/wrap] failed to parse cookie value", err)
		unsetCookie(w)
		value.IsExpired = true
		return value, nil
	}
	value.Values = v
	return value, nil
}
