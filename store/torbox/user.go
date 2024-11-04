package torbox

import (
	"net/url"
	"strconv"
)

type Plan int

const (
	PlanFree = iota
	PlanEssential
	PlanPro
	PlanStandard
)

type GetUserData struct {
	Id               int             `json:"id"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	Email            string          `json:"email"`
	Plan             Plan            `json:"plan"`
	TotalDownloaded  int             `json:"total_downloaded"`
	Customer         string          `json:"customer"`
	Server           int             `json:"server"`
	IsSubscribed     bool            `json:"is_subscribed"`
	PremiumExpiresAt string          `json:"premium_expires_at"`
	CooldownUntil    string          `json:"cooldown_until"`
	AuthId           string          `json:"auth_id"`
	UserReferral     string          `json:"user_referral"`
	BaseEmail        string          `json:"base_email"`
	Settings         *map[string]any `json:"settings,omitempty"`
}

type GetUserParams struct {
	Ctx
	Settings bool // Allows you to retrieve user settings.
}

func (c APIClient) GetUser(params *GetUserParams) (APIResponse[GetUserData], error) {
	form := &url.Values{}
	form.Add("settings", strconv.FormatBool(params.Settings))
	params.Form = form
	response := &Response[GetUserData]{}
	res, err := c.Request("GET", "/v1/api/user/me", params, response)
	return newAPIResponse(res, response.Data, response.Detail), err
}
