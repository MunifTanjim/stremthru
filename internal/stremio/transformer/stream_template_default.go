package stremio_transformer

import "strings"

var StreamTemplateDefault = StreamTemplateBlob{
	Name: strings.TrimSpace(`
{{if .Store.IsProxied}}✨ {{end}}{{if ne .Store.Code ""}}{{if .Store.IsCached}}⚡️ {{end}}[{{.Store.Code}}]{{end}}
{{.Addon.Name}}
{{.Resolution}}
`),
	Description: strings.TrimSpace(`
{{if ne .Quality ""}}💿 {{.Quality}} {{end}}{{if ne .Codec ""}}🎞️ {{.Codec}}{{end}}
{{if ne (len .HDR) 0}}📺 {{str_join .HDR " "}} {{end -}}
{{- if or (gt (len .Audio) 0) (gt (len .Channels) 0)}}🎧 {{if gt (len .Audio) 0}}{{str_join .Audio  ", "}}{{if gt (len .Channels) 0}} | {{end}}{{end}}{{if gt (len .Channels) 0}}{{str_join .Channels ", "}}{{end}}{{end}}
{{if ne .Size ""}}📦 {{.Size}} {{end}}{{if ne .Group ""}} ⚙️ {{.Group}}{{end}}{{if ne .Site ""}}🔗 {{.Site}}{{end}}{{if ne (len .Languages) 0}}
🌐 {{lang_join .Languages " " "emoji"}}
{{- end}}{{if ne .File.Name ""}}
📄 {{.File.Name}}{{else if ne .TTitle ""}}
📁 {{.TTitle}}
{{end}}
`),
}.MustParse()
