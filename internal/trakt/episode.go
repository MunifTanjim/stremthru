package trakt

type MinimalItemEpisode struct {
	Season int         `json:"season"`
	Number int         `json:"number"`
	Title  string      `json:"title"`
	Ids    ListItemIds `json:"ids"`
}
