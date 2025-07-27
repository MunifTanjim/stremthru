package debrider

type GetAccountData struct {
	ResponseContainer
	Id               string `json:"id"`
	Email            string `json:"email"`
	EmailConfirmedAt string `json:"email_confirmed_at"`
	Role             string `json:"role"` // user, admin
	Subscription     struct {
		Id                     string `json:"id"`
		UserId                 string `json:"user_id"`
		PlanId                 string `json:"plan_id"`
		Status                 string `json:"status"` // active, trialing, canceled
		StartDate              string `json:"start_date"`
		EndDate                string `json:"end_date,omitempty"`
		CanceledAt             string `json:"canceled_at,omitempty"`
		CancelAtPeriodEnd      bool   `json:"cancel_at_period_end"`
		TrialStart             string `json:"trial_start,omitempty"`
		TrialEnd               string `json:"trial_end,omitempty"`
		Metadata               any    `json:"metadata,omitempty"`
		CreatedAt              string `json:"created_at"`
		UpdatedAt              string `json:"updated_at"`
		RazorpaySubscriptionId string `json:"razorpay_subscription_id,omitempty"`
		ParadiseSubId          string `json:"paradise_sub_id,omitempty"`
		Plan                   struct {
			Id       string `json:"id"`
			Url      string `json:"url,omitempty"`
			Name     string `json:"name"`
			Price    int    `json:"price"`
			IsFree   bool   `json:"is_free"`
			Currency string `json:"currency"` // USD
			Interval string `json:"interval"` // year
			Metadata struct {
				ApiAccess                bool `json:"api_access"`
				ConcurrentSlots          int  `json:"concurrent_slots"`
				DailyNzbDownloads        int  `json:"daily_nzb_downloads"` // -1
				DownloadSpeedGbps        int  `json:"download_speed_gbps"`
				UnlimitedDownloads       bool `json:"unlimited_downloads"`
				MaxDownloadSizeGb        int  `json:"max_download_size_gb"`
				DailyHosterDownloads     int  `json:"daily_hoster_downloads"`  // -1
				DailyTorrentDownloads    int  `json:"daily_torrent_downloads"` // -1
				DownloadRetentionDays    int  `json:"download_retention_days"`
				UnlimitedHosterDownloads bool `json:"unlimited_hoster_downloads"`
			} `json:"metadata"`
			IsActive       bool   `json:"is_active"`
			CreatedAt      string `json:"created_at"`
			UpdatedAt      string `json:"updated_at"`
			Description    string `json:"description"`
			TotalCount     int    `json:"total_count"`
			IntervalCount  int    `json:"interval_count"`
			RazorpayPlanId string `json:"razorpay_plan_id,omitempty"`
		} `json:"plan"`
	} `json:"subscription"`
}

type GetAccountParams struct {
	Ctx
}

func (c APIClient) GetAccount(params *GetAccountParams) (APIResponse[GetAccountData], error) {
	response := GetAccountData{}
	res, err := c.Request("GET", "/v1/account", params, &response)
	return newAPIResponse(res, response), err
}
