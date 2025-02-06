package stremio_wrap

import (
	"bytes"
	"log"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/MunifTanjim/stremthru/internal/kv"
	"github.com/MunifTanjim/stremthru/stremio"
)

type StreamTransformerField string

const (
	StreamTransformerFieldAddon      StreamTransformerField = "addon"
	StreamTransformerFieldBitDepth   StreamTransformerField = "bitdepth"
	StreamTransformerFieldCached     StreamTransformerField = "cached"
	StreamTransformerFieldCodec      StreamTransformerField = "codec"
	StreamTransformerFieldDebrid     StreamTransformerField = "debrid"
	StreamTransformerFieldFileIdx    StreamTransformerField = "fileidx"
	StreamTransformerFieldFilename   StreamTransformerField = "filename"
	StreamTransformerFieldHash       StreamTransformerField = "hash"
	StreamTransformerFieldHDR        StreamTransformerField = "hdr"
	StreamTransformerFieldQuality    StreamTransformerField = "quality"
	StreamTransformerFieldResolution StreamTransformerField = "resolution"
	StreamTransformerFieldSite       StreamTransformerField = "site"
	StreamTransformerFieldSize       StreamTransformerField = "size"
	StreamTransformerFieldTitle      StreamTransformerField = "title"
)

var extractorStore = func() kv.KVStore[StreamTransformerExtractorBlob] {
	return kv.NewKVStore[StreamTransformerExtractorBlob](&kv.KVStoreConfig{
		Type: "st:wrap:transformer:extractor",
		GetKey: func(key string) string {
			return key
		},
	})
}()

type StreamTransformerExtractorBlob string

type StreamTransformerPattern struct {
	Field string
	Regex *regexp.Regexp
}

type StreamTransformerExtractor []StreamTransformerPattern

func (steb StreamTransformerExtractorBlob) Parse() (StreamTransformerExtractor, error) {
	ste := StreamTransformerExtractor{}
	parts := strings.Split(string(steb), "\n")

	field := ""
	lastField := ""
	lastPart := ""
	for _, part := range parts {
		if part == "" && lastPart == "" {
			field = ""
			lastField = ""
			continue
		}
		if field == "" {
			field = part
		} else {
			re, err := regexp.Compile(part)
			if err != nil {
				log.Printf("[stremio/wrap] failed to compile regex %s: %v", part, err)
				return nil, err
			}
			pattern := StreamTransformerPattern{Regex: re}
			if field != lastField {
				pattern.Field = field
				lastField = field
			}
			ste = append(ste, pattern)
		}
	}

	return ste, nil
}

var templateStore = func() kv.KVStore[StreamTransformerTemplateBlob] {
	return kv.NewKVStore[StreamTransformerTemplateBlob](&kv.KVStoreConfig{
		Type: "st:wrap:transformer:template",
		GetKey: func(key string) string {
			return key
		},
	})
}()

type StreamTransformerTemplateBlob struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type StreamTransformerTemplate struct {
	Name        *template.Template
	Description *template.Template
}

func (sttb StreamTransformerTemplateBlob) IsEmpty() bool {
	return sttb.Name == "" && sttb.Description == ""
}

func (sttb StreamTransformerTemplateBlob) Parse() (*StreamTransformerTemplate, error) {
	if sttb.Name == "" && sttb.Description == "" {
		return nil, nil
	}
	stt := &StreamTransformerTemplate{}
	var err error
	stt.Name, err = template.New("name").Parse(sttb.Name)
	if err != nil {
		log.Printf("[stremio/wrap] failed to parse name template: %v", err)
		return stt, err
	}
	stt.Description, err = template.New("description").Parse(sttb.Description)
	if err != nil {
		log.Printf("[stremio/wrap] failed to parse description template: %v", err)
		return stt, err
	}
	return stt, nil
}

type StreamTransformer struct {
	Extractor StreamTransformerExtractor
	Template  StreamTransformerTemplate
}

type StreamTransformerResult struct {
	Addon      string
	BitDepth   string
	Codec      string
	Debrid     string
	FileIdx    int
	Filename   string
	HDR        string
	Hash       string
	IsCached   bool
	Quality    string
	Resolution string
	Site       string
	Size       string
	Title      string
}

func (st StreamTransformer) parse(stream *stremio.Stream) StreamTransformerResult {
	result := StreamTransformerResult{}
	lastField := ""
	for _, pattern := range st.Extractor {
		field := pattern.Field
		if field == "" {
			field = lastField
		}
		if field == "" {
			continue
		} else {
			lastField = field
		}

		fieldValue := ""
		switch field {
		case "name":
			fieldValue = stream.Name
		case "description":
			fieldValue = stream.Description
			if fieldValue == "" {
				fieldValue = stream.Title
			}
		case "bingeGroup":
			if stream.BehaviorHints != nil {
				fieldValue = stream.BehaviorHints.BingeGroup
			}
		case "url":
			fieldValue = stream.URL
		}
		if fieldValue == "" {
			continue
		}

		for _, match := range pattern.Regex.FindAllStringSubmatch(fieldValue, -1) {
			for i, name := range pattern.Regex.SubexpNames() {
				if i != 0 && name != "" {
					switch name {
					case "addon":
						result.Addon = match[i]
					case "bitdepth":
						result.BitDepth = match[i]
					case "cached":
						result.IsCached = match[i] != ""
					case "codec":
						result.Codec = match[i]
					case "debrid":
						result.Debrid = match[i]
					case "fileidx":
						if fileIdx, err := strconv.Atoi(match[i]); err == nil {
							result.FileIdx = fileIdx
						}
					case "filename":
						result.Filename = match[i]
					case "hash":
						result.Hash = match[i]
					case "hdr":
						result.HDR = match[i]
					case "quality":
						result.Quality = match[i]
					case "resolution":
						result.Resolution = match[i]
					case "site":
						result.Site = match[i]
					case "size":
						result.Size = match[i]
					case "title":
						result.Title = match[i]
					}
				}
			}
		}

	}
	return result
}

func (st StreamTransformer) Do(stream *stremio.Stream) error {
	data := st.parse(stream)
	if stream.InfoHash != "" {
		data.Hash = stream.InfoHash
		data.FileIdx = stream.FileIndex
	}
	if stream.BehaviorHints != nil {
		if stream.BehaviorHints.Filename != "" {
			data.Filename = stream.BehaviorHints.Filename
		}
	}

	var name bytes.Buffer
	err := st.Template.Name.Execute(&name, data)
	if err != nil {
		return err
	}
	stream.Name = name.String()
	var description bytes.Buffer
	err = st.Template.Description.Execute(&description, data)
	if err != nil {
		return err
	}
	stream.Description = description.String()
	if stream.Title != "" {
		stream.Title = ""
	}

	return nil
}
