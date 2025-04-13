package stremio_transformer

import (
	"html/template"
	"strconv"
	"strings"
)

var funcMap = template.FuncMap{
	"str_join":   strings.Join,
	"int_to_str": strconv.Itoa,
}
