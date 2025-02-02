package stremio

type Subtitle struct {
	Id   string `json:"id"`
	Url  string `json:"url"`
	Lang string `json:"lang"`

	SubEncoding string `json:"SubEncoding,omitempty"` // undocumented
	M           string `json:"m,omitempty"`           // undocumented
	G           string `json:"g,omitempty"`           // undocumented
}
