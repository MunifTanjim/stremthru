package offcloud

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type cachedAuth struct {
	cookie string
	apikey string
}

const SESSION_COOKIE_NAME = "connect.sid"

func parseCredential(token string) (email string, password string) {
	email, password, _ = strings.Cut(token, ":")
	return
}

func extractSessionCookieValue(setCookieHeader string) (string, bool) {
	cookies := strings.Split(setCookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, SESSION_COOKIE_NAME+"=") {
			return strings.TrimPrefix(cookie, SESSION_COOKIE_NAME+"="), true
		}
	}
	return "", false
}

func (c APIClient) getAuth(params request.Context) (*cachedAuth, error) {
	token := params.GetAPIKey(c.apiKey)
	auth := &cachedAuth{}
	if c.authCache.Get(token, auth) {
		return auth, nil
	}
	email, password := parseCredential(token)
	login_res, err := c.login(&loginParams{
		Username: email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	cookie, found := extractSessionCookieValue(login_res.Header.Get("Set-Cookie"))
	if !found {
		return nil, errors.New("failed to login")
	}
	auth.cookie = cookie
	apikey_res, err := c.getApiKey(&getAPIKeyParams{
		Cookie: cookie,
	})
	if err != nil {
		return nil, err
	}
	auth.apikey = apikey_res.Data.APIKey
	if err := c.authCache.Add(token, *auth); err != nil {
		return nil, err
	}
	return auth, nil
}

func (c APIClient) injectSessionCookie(ctx *Ctx) {
	if ctx.Headers == nil {
		ctx.Headers = &http.Header{}
	}
	if auth, err := c.getAuth(ctx); err == nil {
		ctx.Headers.Set("Cookie", SESSION_COOKIE_NAME+"="+auth.cookie+";")
	}
}

func (c APIClient) injectAPIKey(ctx *Ctx) {
	if ctx.Query == nil {
		ctx.Query = &url.Values{}
	}
	if auth, err := c.getAuth(ctx); err == nil {
		ctx.Query.Add("key", auth.apikey)
	}
}

type loginParams struct {
	Ctx
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginData struct {
	ResponseContainer
	Email  string `json:"email"`
	UserId string `json:"userId"`
}

func (c APIClient) login(params *loginParams) (APIResponse[loginData], error) {
	params.JSON = params
	response := &loginData{}
	res, err := c.Request("POST", "/api/login", params, response)
	return newAPIResponse(res, *response), err
}

type getAPIKeyParams struct {
	Ctx
	Cookie string
}

type getAPIKeyData struct {
	ResponseContainer
	Email  string `json:"email"`
	APIKey string `json:"apiKey"`
}

func (c APIClient) getApiKey(params *getAPIKeyParams) (APIResponse[getAPIKeyData], error) {
	params.Headers = &http.Header{}
	params.Headers.Set("Cookie", SESSION_COOKIE_NAME+"="+params.Cookie+";")
	response := &getAPIKeyData{}
	res, err := c.Request("POST", "/api/key", params, response)
	return newAPIResponse(res, *response), err
}
