package dash_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/shared"
	"github.com/google/uuid"
)

type Session struct {
	Id   string
	User string
}

var sessionStorage = cache.NewCache[Session](&cache.CacheConfig{
	Lifetime:      7 * 24 * time.Hour,
	Name:          "dash:session",
	LocalCapacity: 8,
})

const SESSION_COOKIE_NAME = "stremthru.dash.session"
const SESSION_COOKIE_PATH = "/dash/"

func (s Session) Save(w http.ResponseWriter, r *http.Request) error {
	if s.Id == "" {
		s.Id = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	if err := sessionStorage.Add(s.Id, s); err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     SESSION_COOKIE_NAME,
		Value:    s.Id,
		HttpOnly: true,
		Path:     SESSION_COOKIE_PATH,
		Secure:   strings.HasPrefix(r.Referer(), "https:"),
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

func (s Session) Destroy(w http.ResponseWriter) {
	sessionStorage.Remove(s.Id)
	http.SetCookie(w, &http.Cookie{
		Name:    SESSION_COOKIE_NAME,
		Expires: time.Unix(0, 0),
		Path:    SESSION_COOKIE_PATH,
	})
}

func getSession(w http.ResponseWriter, r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(SESSION_COOKIE_NAME)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}

	session := &Session{}
	if !sessionStorage.Get(cookie.Value, session) {
		session.Id = cookie.Value
		session.Destroy(w)
		return nil, nil
	}
	return session, nil
}

type GetUserResponse struct {
	Id string `json:"id"`
}

func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := GetReqCtx(r)
	SendData(w, r, 200, GetUserResponse{
		Id: ctx.Session.User,
	})
}

type SignInRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

func HandleSignIn(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	request := &SignInRequest{}
	if err := ReadRequestBodyJSON(r, request); err != nil {
		SendError(w, r, err)
		return
	}

	if password := config.AdminPassword.GetPassword(request.User); password == "" || password != request.Password {
		ErrorUnauthorized(r, "Invalid Credentials").Send(w, r)
		return
	}

	ctx := GetReqCtx(r)

	if ctx.Session == nil {
		ctx.Session = &Session{}
	}
	ctx.Session.User = request.User
	if err := ctx.Session.Save(w, r); err != nil {
		SendError(w, r, err)
		return
	}

	SendData(w, r, 200, GetUserResponse{
		Id: ctx.Session.User,
	})
}

func HandleSignOut(w http.ResponseWriter, r *http.Request) {
	if !shared.IsMethod(r, http.MethodPost) {
		ErrorMethodNotAllowed(r).Send(w, r)
		return
	}

	ctx := GetReqCtx(r)

	if ctx.Session != nil {
		ctx.Session.Destroy(w)
	}

	SendData(w, r, 204, nil)
}
