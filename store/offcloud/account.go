package offcloud

import (
	"encoding/json"
	"errors"
	"time"
)

type GetAccountInfoParams struct {
	Ctx
}

type GetAccountInfoData struct {
	ResponseContainer
	UserID         string `json:"user_id"`
	IsPremium      bool   `json:"is_premium"`
	ExpirationDate string `json:"expiration_date,omitempty"` // 2026-12-31
	CanDownload    bool   `json:"can_download"`
}

func (c APIClient) GetAccountInfo(params *GetAccountInfoParams) (APIResponse[GetAccountInfoData], error) {
	response := GetAccountInfoData{}
	res, err := c.Request("GET", "/api/account/info", params, &response)
	return newAPIResponse(res, response), err
}

type GetAccountParams struct {
	Ctx
}

type GetAccountData struct {
	ResponseContainer
	APIKey             string `json:"apiKey"`
	Email              string `json:"email"`
	EmailNotifications []any  `json:"emailNotifications"`
	ReferralCode       string `json:"referralCode"`
}

func (c APIClient) GetAccount(params *GetAccountParams) (APIResponse[GetAccountData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetAccountData{}
	res, err := c.Request("POST", "/account/get", params, response)
	return newAPIResponse(res, *response), err
}

type GetAccountStatsParams struct {
	Ctx
}

type GetAccountStatsDataAdditionalTags struct {
	Mega bool `json:"mega"`
}

type MembershipType string

const (
	MembershipTypeLifetime MembershipType = "Lifetime"
	MembershipTypeNone     MembershipType = ""
)

type dateOrInt struct{ time.Time }

func (doi *dateOrInt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		t, err := time.Parse("02-01-2006", str)
		if err != nil {
			return err
		}
		doi.Time = t
		return nil
	}
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		doi.Time = time.Unix(int64(i), 0)
		return nil
	}
	return errors.New("[offcloud] failed to parse: " + string(data))
}

type GetAccountStatsData struct {
	ResponseContainer
	AdditionalTags GetAccountStatsDataAdditionalTags `json:"additionalTags"`
	ExpirationDate dateOrInt                         `json:"expirationDate"`
	MembershipType MembershipType                    `json:"membershipType"`
	Revenue        int                               `json:"revenue"`
}

func (c APIClient) GetAccountStats(params *GetAccountStatsParams) (APIResponse[GetAccountStatsData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetAccountStatsData{}
	res, err := c.Request("POST", "/account/stats", params, response)
	return newAPIResponse(res, *response), err
}

type GetUserEmailParams struct {
	Ctx
}

type GetUserEmailData struct {
	ResponseContainer
	CreatedOn  time.Time `json:"createdOn"`
	Email      string    `json:"email"`
	ProfilePic string    `json:"profilePic"`
	UserId     string    `json:"userId"`
}

func (c APIClient) GetUserEmail(params *GetUserEmailParams) (APIResponse[GetUserEmailData], error) {
	c.injectSessionCookie(&params.Ctx)
	response := &GetUserEmailData{}
	res, err := c.Request("POST", "/auth/user/email", params, response)
	return newAPIResponse(res, *response), err
}
