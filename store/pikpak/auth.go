package pikpak

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/kv"
)

type CaptchaToken struct {
	Token     string
	ExpiresAt int64
}

func (ct *CaptchaToken) inject(c APIClient, ctx *Ctx, method, endpoint string) error {
	if method == "" {
		method = "POST"
	}
	if endpoint == "" {
		endpoint = "/config/v1/basic"
	}
	action := getCaptchaAction(method, endpoint)

	deviceId := ctx.GetDeviceId()
	cacheKey := deviceId + ":" + action
	if captchaCache.Get(cacheKey, ct) {
		if !ct.IsExpiring() {
			ctx.auth.CaptchaToken = ct.Token
			return nil
		}
	}
	log.Printf("[pikpak] refreshing captcha")
	params := c.getInitCaptchaParams(action)
	params.DeviceId = deviceId
	params.Meta.Timestamp = getTimestamp()
	params.Meta.CaptchaSign = calculateCaptchaSign(params.DeviceId, params.Meta.Timestamp)
	params.Meta.ClientVersion = clientVersion
	params.Meta.PackageName = packageName
	params.Meta.UserId = ctx.auth.UserId
	res, err := c.initCaptcha(params)
	if err != nil {
		log.Printf("[pikpak] failed to refresh captcha: %v\n", err)
		return err
	}
	ct.Token = res.Data.CaptchaToken
	ct.ExpiresAt = time.Now().Unix() + res.Data.ExpiresIn
	if err := captchaCache.Add(cacheKey, *ct); err != nil {
		log.Printf("[pikpak] failed to cache captcha: %v\n", err)
	}
	ctx.auth.CaptchaToken = ct.Token
	return nil
}

func (ct CaptchaToken) IsExpiring() bool {
	return ct.ExpiresAt <= time.Now().Add(30*time.Second).Unix()
}

var captchaCache = func() cache.Cache[CaptchaToken] {
	return cache.NewCache[CaptchaToken](&cache.CacheConfig{
		Lifetime: 300 * time.Second,
		Name:     "store:pikpak:captcha",
	})
}()

var pikpakAuthState = func() kv.KVStore[AuthState] {
	return kv.NewKVStore[AuthState](&kv.KVStoreConfig{
		GetKey: func(key string) string {
			return "pikpak:auth:" + key
		},
	})
}()

type AuthState struct {
	AccessToken  string `json:"atok"`
	RefreshToken string `json:"rtok"`
	UserId       string `json:"uid"`
	ExpiresAt    int64  `json:"exp"`
	CaptchaToken string `json:"-"`
}

func (as *AuthState) load(deviceId string) error {
	return pikpakAuthState.Get(deviceId, as)
}

func (as AuthState) save(deviceId string) error {
	return pikpakAuthState.Set(deviceId, as)
}

func (as AuthState) IsAuthed() bool {
	return as.RefreshToken != "" && as.UserId != ""
}

func (as AuthState) IsExpired() bool {
	return as.ExpiresAt <= time.Now().Unix()
}

func (as AuthState) IsExpiring() bool {
	return as.ExpiresAt <= time.Now().Add(10*time.Minute).Unix()
}

var WebAlgorithms = []string{
	"fyZ4+p77W1U4zcWBUwefAIFhFxvADWtT1wzolCxhg9q7etmGUjXr",
	"uSUX02HYJ1IkyLdhINEFcCf7l2",
	"iWt97bqD/qvjIaPXB2Ja5rsBWtQtBZZmaHH2rMR41",
	"3binT1s/5a1pu3fGsN",
	"8YCCU+AIr7pg+yd7CkQEY16lDMwi8Rh4WNp5",
	"DYS3StqnAEKdGddRP8CJrxUSFh",
	"crquW+4",
	"ryKqvW9B9hly+JAymXCIfag5Z",
	"Hr08T/NDTX1oSJfHk90c",
	"i",
}

type loginInitCaptchaMeta struct {
	Username    string `json:"username"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
}

type initCaptchaParamsMeta struct {
	CaptchaSign   string `json:"captcha_sign,omitempty"`
	ClientVersion string `json:"client_version,omitempty"`
	PackageName   string `json:"package_name,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	UserId        string `json:"user_id,omitempty"`
	UserName      string `json:"username,omitempty"`
	Email         string `json:"email,omitempty"`
	PhoneNumber   string `json:"phone_number,omitempty"`
}

type initCaptchaParams struct {
	Ctx
	Action       string                 `json:"action,omitempty"`
	CaptchaToken string                 `json:"captcha_token,omitempty"`
	ClientId     string                 `json:"client_id,omitempty"`
	DeviceId     string                 `json:"device_id,omitempty"`
	Meta         *initCaptchaParamsMeta `json:"meta,omitempty"`
}

func getDeviceId(credential string) string {
	// Hash the input string using SHA-256
	hash := sha256.Sum256([]byte(credential))

	// Use the first 16 bytes of the hash as the UUID
	uuid := make([]byte, 16)
	copy(uuid, hash[:16])

	// Set the UUID version to 4
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4 (0100xxxx)

	// Set the variant to RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant (10xxxxxx)

	// Format the UUID as a string without dashes
	return fmt.Sprintf("%08x%04x%04x%04x%012x",
		uuid[0:4],  // First 4 bytes
		uuid[4:6],  // Next 2 bytes
		uuid[6:8],  // Next 2 bytes (with version bits)
		uuid[8:10], // Next 2 bytes (with variant bits)
		uuid[10:])  // Last 6 bytes
}

func getCaptchaAction(method string, endpoint string) string {
	return method + ":" + endpoint
}

func (c APIClient) getInitCaptchaParams(action string) *initCaptchaParams {
	return &initCaptchaParams{
		Action:   action,
		ClientId: clientId,
		Meta:     &initCaptchaParamsMeta{},
	}
}

var md5Salts = []string{
	"Gez0T9ijiI9WCeTsKSg3SMlx",
	"zQdbalsolyb1R/",
	"ftOjr52zt51JD68C3s",
	"yeOBMH0JkbQdEFNNwQ0RI9T3wU/v",
	"BRJrQZiTQ65WtMvwO",
	"je8fqxKPdQVJiy1DM6Bc9Nb1",
	"niV",
	"9hFCW2R1",
	"sHKHpe2i96",
	"p7c5E6AcXQ/IJUuAEC9W6",
	"",
	"aRv9hjc9P+Pbn+u3krN6",
	"BzStcgE8qVdqjEH16l4",
	"SqgeZvL5j9zoHP95xWHt",
	"zVof5yaJkPe3VFpadPof",
}

func md5sum(v string) string {
	sum := md5.Sum([]byte(v))
	return hex.EncodeToString(sum[:])
}

func sha1sum(v string) string {
	sum := sha1.Sum([]byte(v))
	return hex.EncodeToString(sum[:])
}

func calculateCaptchaSign(deviceId string, timestamp string) (captchaSign string) {
	captchaSign = clientId + clientVersion + packageName + deviceId + timestamp
	for _, salt := range md5Salts {
		captchaSign = md5sum(captchaSign + salt)
	}
	captchaSign = "1." + captchaSign
	return captchaSign
}

func generateDeviceSign(deviceId string) string {
	return "div101." + deviceId + md5sum(sha1sum(deviceId+packageName+"1appkey"))
}

func buildUserAgent(deviceId string, userId string) string {
	return strings.Join(
		[]string{
			"ANDROID-" + packageName + "/" + clientVersion,
			"protocolVersion/200",
			"accesstype/",
			"clientid/" + clientId,
			"clientversion/" + clientVersion,
			"action_type/",
			"networktype/WIFI",
			"sessionid/",
			"deviceid/" + deviceId,
			"providername/NONE",
			"devicesign/" + generateDeviceSign(deviceId),
			"refresh_token/",
			"sdkversion/" + sdkVersion,
			"datetime/" + getTimestamp(),
			"usrno/" + userId,
			"appname/" + packageName,
			"session_origin/",
			"grant_type/",
			"appid/",
			"clientip/",
			"devicename/Xiaomi_M2004j7ac",
			"osversion/13",
			"platformversion/10",
			"accessmode/",
			"devicemodel/M2004J7AC",
		}, " ")
}

type initCaptchaData struct {
	ResponseContainer
	CaptchaToken string `json:"captcha_token"`
	ExpiresIn    int64  `json:"expires_in"`
	URL          string `json:"url,omitempty"`
}

func (c APIClient) initCaptcha(params *initCaptchaParams) (APIResponse[initCaptchaData], error) {
	params.JSON = params
	response := &initCaptchaData{}
	res, err := c.UserRequest("POST", "/v1/shield/captcha/init", params, response)
	return newAPIResponse(res, *response), err

}

type LoginParams struct {
	Ctx
}

type signinData struct {
	ResponseContainer
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Sub          string `json:"sub"`
	TokenType    string `json:"token_type"`
	IdToken      string `json:"id_token"`
	Scope        string `json:"scope"`
	UserId       string `json:"user_id"`
}

type signinParams struct {
	Ctx
}

func (c APIClient) getCaptchaTokenForLogin(username string, deviceId string) (APIResponse[initCaptchaData], error) {
	captchaParams := c.getInitCaptchaParams(getCaptchaAction("POST", UserAPIBaseURL.JoinPath("/v1/auth/signin").String()))
	if ok, _ := regexp.MatchString(`\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`, username); ok {
		captchaParams.Meta.Email = username
	} else if ok, _ := regexp.MatchString(`\d{11,18}`, username); ok {
		captchaParams.Meta.PhoneNumber = username
	} else {
		captchaParams.Meta.UserName = username
	}
	captchaParams.DeviceId = deviceId
	return c.initCaptcha(captchaParams)
}

func getTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

func (c APIClient) withCaptchaToken(ctx *Ctx, method, endpoint string) error {
	captcha := &CaptchaToken{}
	return captcha.inject(c, ctx, method, endpoint)
}

func (c APIClient) withAccessToken(ctx *Ctx) error {
	deviceId := ctx.GetDeviceId()

	if ctx.auth == nil {
		ctx.auth = &AuthState{}
	}
	err := ctx.auth.load(deviceId)
	if err != nil {
		return err
	}

	if ctx.auth.IsAuthed() {
		if ctx.auth.IsExpiring() {
			if err := c.refreshAuthToken(ctx); err != nil {
				log.Printf("[pikpak] failed to refresh access token: %v", deviceId)
			} else {
				log.Printf("[pikpak] refreshed access token: %v", deviceId)
			}
		}
		return nil
	}

	user := ctx.GetUser()
	username, password := user.Username, user.Password

	sResponse := &signinData{}

	captchaRes, err := c.getCaptchaTokenForLogin(username, deviceId)
	if err != nil {
		return err
	}

	signinParams := &signinParams{Ctx: *ctx}
	signinParams.Form = &url.Values{
		"captcha_token": []string{captchaRes.Data.CaptchaToken},
		"client_id":     []string{clientId},
		"client_secret": []string{clientSecret},
		"password":      []string{password},
		"username":      []string{username},
	}
	_, err = c.UserRequest("POST", "/v1/auth/signin", signinParams, sResponse)
	if err != nil {
		log.Printf("[pikpak] failed to signin: %v\n", err)
		return err
	}

	ctx.auth.AccessToken = sResponse.AccessToken
	ctx.auth.RefreshToken = sResponse.RefreshToken
	ctx.auth.UserId = sResponse.Sub
	ctx.auth.ExpiresAt = time.Now().Unix() + sResponse.ExpiresIn

	err = ctx.auth.save(deviceId)
	if err != nil {
		log.Printf("[pikpak] failed to store auth state: %v\n", err)
		return err
	}
	return nil
}

type refreshAuthTokenParams struct {
	Ctx
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
	GrantType    string `json:"grant_type"`
}

type refreshAuthTokenData struct {
	ResponseContainer
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Sub          string `json:"sub"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (c APIClient) refreshAuthToken(ctx *Ctx) error {
	params := &refreshAuthTokenParams{
		Ctx:          *ctx,
		RefreshToken: ctx.auth.RefreshToken,
		ClientId:     clientId,
		GrantType:    "refresh_token",
	}
	params.JSON = params
	response := &refreshAuthTokenData{}
	_, err := c.UserRequest("POST", "/v1/auth/token", params, response)
	if err != nil {
		return err
	}
	ctx.auth.AccessToken = response.AccessToken
	ctx.auth.RefreshToken = response.RefreshToken
	ctx.auth.UserId = response.Sub
	ctx.auth.ExpiresAt = time.Now().Unix() + response.ExpiresIn
	ctx.auth.save(ctx.GetDeviceId())
	return nil
}

type GetUserParams struct {
	Ctx
}

type GetUserDataProvider struct {
	Id             string `json:"id"`
	ProviderUserId string `json:"provider_user_id"`
	Name           string `json:"name"`
}

type GetUserData struct {
	ResponseContainer
	CreatedAt         time.Time             `json:"created_at"`
	Email             string                `json:"email"`
	Name              string                `json:"name"`
	PasswordUpdatedAt time.Time             `json:"password_updated_at"`
	Sub               string                `json:"sub"`
	Providers         []GetUserDataProvider `json:"providers"`
}

func (c APIClient) GetUser(params *GetUserParams) (APIResponse[GetUserData], error) {
	response := &GetUserData{}

	err := c.withAccessToken(&params.Ctx)
	if err != nil {
		return newAPIResponse(nil, *response), err
	}

	res, err := c.UserRequest("GET", "/v1/user/me", params, response)
	return newAPIResponse(res, *response), err
}
