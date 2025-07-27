package debrider

type CheckLinkAvailabilityDataItemFile struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	DownloadLink string `json:"download_link"`
}

type CheckLinkAvailabilityDataItem struct {
	Cached bool                                `json:"cached"`
	Files  []CheckLinkAvailabilityDataItemFile `json:"files"`
}

type CheckLinkAvailabilityData struct {
	ResponseContainer
	Result []CheckLinkAvailabilityDataItem `json:"result"`
}

type CheckLinkAvailabilityParams struct {
	Ctx
	Data []string `json:"data"` // links
}

func (c APIClient) CheckLinkAvailability(params *CheckLinkAvailabilityParams) (APIResponse[CheckLinkAvailabilityData], error) {
	params.JSON = params
	response := &CheckLinkAvailabilityData{}
	res, err := c.Request("POST", "/v1/link/lookup", params, response)
	return newAPIResponse(res, *response), err
}
