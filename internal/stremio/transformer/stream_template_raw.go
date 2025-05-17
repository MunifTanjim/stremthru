package stremio_transformer

var StreamTemplateRaw = StreamTemplateBlob{
	Name:        `{{.Raw.Name}}`,
	Description: `{{.Raw.Description}}`,
}.MustParse()
