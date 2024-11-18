package realdebrid

type UserType string

const (
	UserTypeFree    = "free"
	UserTypePremium = "premium"
)

type GetUserData struct {
	*ResponseError
	Id         int      `json:"id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Points     int      `json:"points"`     // Fidelity points
	Locale     string   `json:"locale"`     // User language
	Avatar     string   `json:"avatar"`     // URL
	Type       UserType `json:"type"`       // "premium" or "free"
	Premium    int      `json:"premium"`    // seconds left as a Premium user
	Expiration string   `json:"expiration"` // jsonDate
}

type GetUserParams struct {
	Ctx
}

func (c APIClient) GetUser(params *GetUserParams) (APIResponse[GetUserData], error) {
	response := &GetUserData{}
	res, err := c.Request("GET", "/rest/1.0/user", params, response)
	return newAPIResponse(res, *response), err
}
