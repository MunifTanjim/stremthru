package stremio_transformer

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"

	"github.com/MunifTanjim/stremthru/stremio"
)

type StreamTemplateBlob struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (blob StreamTemplateBlob) IsEmpty() bool {
	return blob.Name == "" && blob.Description == ""
}

type StreamTemplate struct {
	Name        *template.Template
	Description *template.Template
}

func (blob StreamTemplateBlob) Parse() (*StreamTemplate, error) {
	if blob.IsEmpty() {
		return nil, nil
	}
	t := &StreamTemplate{}
	var err error
	t.Name, err = template.New("name").Funcs(funcMap).Parse(blob.Name)
	if err != nil {
		log.Error("failed to parse name template", "error", err)
		return t, err
	}
	t.Description, err = template.New("description").Funcs(funcMap).Parse(blob.Description)
	if err != nil {
		log.Error("failed to parse description template", "error", err)
		return t, err
	}
	return t, nil
}

type StreamTemplateDataRaw struct {
	Name        string
	Description string
}

var newlinesRegex = regexp.MustCompile("\n\n+")

func (t StreamTemplate) Execute(stream *stremio.Stream, data *StreamExtractorResult) (*stremio.Stream, error) {
	var name bytes.Buffer
	if err := t.Name.Execute(&name, data); err != nil {
		return stream, err
	}
	stream.Name = strings.TrimSpace(name.String())

	var description bytes.Buffer
	if err := t.Description.Execute(&description, data); err != nil {
		return stream, err
	}
	stream.Description = newlinesRegex.ReplaceAllLiteralString(strings.TrimSpace(description.String()), "\n")
	if stream.Title != "" {
		stream.Title = ""
	}

	return stream, nil
}
