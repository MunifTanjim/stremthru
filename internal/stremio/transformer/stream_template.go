package stremio_transformer

import (
	"bytes"
	"regexp"
	"strings"
	"text/template"

	"github.com/MunifTanjim/stremthru/internal/util"
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
	Blob        StreamTemplateBlob
	Name        *template.Template
	Description *template.Template
}

func (t StreamTemplate) IsEmpty() bool {
	return t.Name == nil && t.Description == nil
}

func (blob StreamTemplateBlob) Parse() (*StreamTemplate, error) {
	t := &StreamTemplate{
		Blob: blob,
	}
	if blob.IsEmpty() {
		return t, nil
	}
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

func (blob StreamTemplateBlob) MustParse() *StreamTemplate {
	st, err := blob.Parse()
	if err != nil {
		panic(err)
	}
	return st
}

type StreamTemplateDataRaw struct {
	Name        string
	Description string
}

var newlinesRegex = regexp.MustCompile("\n\n+")

func (t StreamTemplate) Execute(stream *stremio.Stream, data *StreamExtractorResult) (s *stremio.Stream, err error) {
	defer func() {
		if perr, stack := util.HandlePanic(recover(), true); perr != nil {
			err = perr
			log.Error("StreamTemplate panic", "error", err, "stack", stack)
		}
	}()

	s = stream
	if t.Name != nil {
		var name bytes.Buffer
		if err := t.Name.Execute(&name, data); err != nil {
			return stream, err
		}
		stream.Name = strings.TrimSpace(name.String())
	}

	if t.Description != nil {
		var description bytes.Buffer
		if err := t.Description.Execute(&description, data); err != nil {
			return stream, err
		}
		stream.Description = newlinesRegex.ReplaceAllLiteralString(strings.TrimSpace(description.String()), "\n")
		if stream.Title != "" {
			stream.Title = ""
		}
	}

	return stream, nil
}
