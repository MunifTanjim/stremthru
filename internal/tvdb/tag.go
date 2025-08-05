package tvdb

type TagOption struct {
	Id       int    `json:"id"`
	Tag      int    `json:"tag"`
	TagName  string `json:"tagName"`
	Name     string `json:"name"`
	HelpText string `json:"helpText"`
}
