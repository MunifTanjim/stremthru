package stremio_transformer

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MunifTanjim/go-ptt"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/stremio"
)

var nonAlphaRegex = regexp.MustCompile("(?i)[^a-zA-Z]")

type StreamExtractorField = string

const (
	StreamExtractorFieldAddonName     StreamExtractorField = "addon_name"
	StreamExtractorFieldBitDepth      StreamExtractorField = "bitdepth"
	StreamExtractorFieldChannel       StreamExtractorField = "channel"
	StreamExtractorFieldCodec         StreamExtractorField = "codec"
	StreamExtractorFieldEpisode       StreamExtractorField = "episode"
	StreamExtractorFieldFileIdx       StreamExtractorField = "file_idx"
	StreamExtractorFieldFileName      StreamExtractorField = "file_name"
	StreamExtractorFieldFileSize      StreamExtractorField = "file_size"
	StreamExtractorFieldHDR           StreamExtractorField = "hdr"
	StreamExtractorFieldHDRSep        StreamExtractorField = "hdr_sep"
	StreamExtractorFieldLanguage      StreamExtractorField = "language"
	StreamExtractorFieldLanguageSep   StreamExtractorField = "language_sep"
	StreamExtractorFieldHash          StreamExtractorField = "hash"
	StreamExtractorFieldQuality       StreamExtractorField = "quality"
	StreamExtractorFieldResolution    StreamExtractorField = "resolution"
	StreamExtractorFieldSeeders       StreamExtractorField = "seeders"
	StreamExtractorFieldSeason        StreamExtractorField = "season"
	StreamExtractorFieldSite          StreamExtractorField = "site"
	StreamExtractorFieldSize          StreamExtractorField = "size"
	StreamExtractorFieldStoreCode     StreamExtractorField = "store_code"
	StreamExtractorFieldStoreIsCached StreamExtractorField = "store_is_cached"
	StreamExtractorFieldStoreName     StreamExtractorField = "store_name"
	StreamExtractorFieldTTitle        StreamExtractorField = "t_title"
)

type StreamExtractorBlob string

type StreamExtractorPattern struct {
	Field string
	Regex *regexp.Regexp
}

type StreamExtractor struct {
	Blob  StreamExtractorBlob
	items []StreamExtractorPattern
}

func (seb StreamExtractorBlob) Parse() (StreamExtractor, error) {
	se := StreamExtractor{
		Blob: seb,
	}
	if seb == "" {
		return se, nil
	}

	parts := strings.Split(string(seb), "\n")

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
				log.Error("failed to compile regex", "regex", part, "error", err)
				return se, err
			}
			pattern := StreamExtractorPattern{Regex: re}
			if field != lastField {
				pattern.Field = field
				lastField = field
			}
			se.items = append(se.items, pattern)
		}
	}

	return se, nil
}

func (seb StreamExtractorBlob) MustParse() StreamExtractor {
	se, err := seb.Parse()
	if err != nil {
		panic(err)
	}
	return se
}

type StreamExtractorResultFile struct {
	Idx  int
	Name string
	Size string
}

type StreamExtractorResultStore struct {
	Name      string
	Code      string
	IsCached  bool
	IsProxied bool
}

type StreamExtractorResultAddon struct {
	Name string
}

type StreamExtractorResultRaw struct {
	Name        string
	Description string
}

type StreamExtractorResult struct {
	*ptt.Result

	Addon     StreamExtractorResultAddon
	Age       time.Duration
	Category  string
	Episode   int
	File      StreamExtractorResultFile
	Hash      string
	IsPrivate bool
	Raw       StreamExtractorResultRaw
	Season    int
	Seeders   int
	Store     StreamExtractorResultStore
	TTitle    string `expr:"-"`
	Indexer   string `expr:"-"`
	Emoji     string `expr:"-"`
}

var language_to_code = map[string]string{
	"dubbed":      "dub",
	"dual audio":  "daud",
	"multi audio": "maud",
	"multi subs":  "msub",

	"english":    "en",
	"🇬🇧":         "en",
	"🇺🇸":         "en",
	"japanese":   "ja",
	"🇯🇵":         "ja",
	"russian":    "ru",
	"🇷🇺":         "ru",
	"italian":    "it",
	"🇮🇹":         "it",
	"portuguese": "pt",
	"🇵🇹":         "pt",
	"🇧🇷":         "pt",
	"spanish":    "es",
	"🇪🇸":         "es",
	"latino":     "es-419",
	"🇲🇽":         "es-mx",
	"korean":     "ko",
	"🇰🇷":         "ko",
	"chinese":    "zh",
	"🇨🇳":         "zh",
	"taiwanese":  "zh-tw",
	"🇹🇼":         "zh-tw",
	"french":     "fr",
	"🇫🇷":         "fr",
	"german":     "de",
	"🇩🇪":         "de",
	"dutch":      "nl",
	"🇳🇱":         "nl",
	"hindi":      "hi",
	"🇮🇳":         "hi",
	"telugu":     "te",
	"tamil":      "ta",
	"malayalam":  "ml",
	"kannada":    "kn",
	"marathi":    "mr",
	"gujarati":   "gu",
	"punjabi":    "pa",
	"bengali":    "bn",
	"🇧🇩":         "bn",
	"polish":     "pl",
	"🇵🇱":         "pl",
	"lithuanian": "lt",
	"🇱🇹":         "lt",
	"latvian":    "lv",
	"🇱🇻":         "lv",
	"estonian":   "et",
	"🇪🇪":         "et",
	"czech":      "cs",
	"🇨🇿":         "cs",
	"slovakian":  "sk",
	"🇸🇰":         "sk",
	"slovenian":  "sl",
	"🇸🇮":         "sl",
	"hungarian":  "hu",
	"🇭🇺":         "hu",
	"romanian":   "ro",
	"🇷🇴":         "ro",
	"bulgarian":  "bg",
	"🇧🇬":         "bg",
	"serbian":    "sr",
	"🇷🇸":         "sr",
	"croatian":   "hr",
	"🇭🇷":         "hr",
	"ukrainian":  "uk",
	"🇺🇦":         "uk",
	"greek":      "el",
	"🇬🇷":         "el",
	"danish":     "da",
	"🇩🇰":         "da",
	"finnish":    "fi",
	"🇫🇮":         "fi",
	"swedish":    "sv",
	"🇸🇪":         "sv",
	"norwegian":  "no",
	"🇳🇴":         "no",
	"turkish":    "tr",
	"🇹🇷":         "tr",
	"arabic":     "ar",
	"🇸🇦":         "ar",
	"persian":    "fa",
	"🇮🇷":         "fa",
	"hebrew":     "he",
	"🇮🇱":         "he",
	"vietnamese": "vi",
	"🇻🇳":         "vi",
	"indonesian": "id",
	"🇮🇩":         "id",
	"malay":      "ms",
	"🇲🇾":         "ms",
	"thai":       "th",
	"🇹🇭":         "th",
}

func (se StreamExtractor) Parse(stream *stremio.Stream, sType string) *StreamExtractorResult {
	r := &StreamExtractorResult{
		Result: &ptt.Result{},
		File: StreamExtractorResultFile{
			Idx: -1,
		},
		Season:  -1,
		Episode: -1,
		Raw: StreamExtractorResultRaw{
			Name:        stream.Name,
			Description: stream.Description,
		},
		Category: sType,
	}
	if stream.Description == "" {
		r.Raw.Description = stream.Title
	}

	var hdr, hdr_sep string
	var language, language_sep string

	lastField := ""
	for _, pattern := range se.items {
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
		case "filename":
			if stream.BehaviorHints != nil {
				fieldValue = stream.BehaviorHints.Filename
			}
		case "url":
			fieldValue = stream.URL
		}
		if fieldValue == "" {
			continue
		}

		for _, match := range pattern.Regex.FindAllStringSubmatch(fieldValue, -1) {
			for i, name := range pattern.Regex.SubexpNames() {
				value := match[i]
				if i != 0 && name != "" && value != "" {
					switch name {
					case "addon", StreamExtractorFieldAddonName:
						r.Addon.Name = value
					case StreamExtractorFieldBitDepth:
						r.BitDepth = value
					case "cached", StreamExtractorFieldStoreIsCached:
						r.Store.IsCached = true
					case StreamExtractorFieldChannel:
						r.Channels = []string{value}
					case StreamExtractorFieldCodec:
						if r.Codec == "" {
							r.Codec = value
						}
					case "debrid", StreamExtractorFieldStoreCode:
						r.Store.Code = value
					case StreamExtractorFieldStoreName:
						r.Store.Name = value
					case StreamExtractorFieldEpisode:
						if ep, err := strconv.Atoi(value); err == nil {
							r.Episode = ep
							if len(r.Episodes) == 0 {
								r.Episodes = []int{ep}
							}
						}
					case "fileidx", StreamExtractorFieldFileIdx:
						if fileIdx, err := strconv.Atoi(value); err == nil {
							r.File.Idx = fileIdx
						}
					case "filename", StreamExtractorFieldFileName:
						if field == "url" {
							if name, err := url.PathUnescape(value); err == nil {
								value = name
							}
						}
						r.File.Name = value
					case StreamExtractorFieldFileSize:
						r.Size = value
					case StreamExtractorFieldHash:
						r.Hash = value
					case StreamExtractorFieldHDR:
						hdr = value
					case StreamExtractorFieldHDRSep:
						hdr_sep = value
					case StreamExtractorFieldLanguage:
						language = value
					case StreamExtractorFieldLanguageSep:
						language_sep = value
					case StreamExtractorFieldQuality:
						if r.Quality == "" {
							r.Quality = value
						}
					case StreamExtractorFieldResolution:
						if r.Resolution == "" {
							r.Resolution = value
						}
					case StreamExtractorFieldSeeders:
						if seeders, err := strconv.Atoi(value); err == nil {
							r.Seeders = seeders
						}
					case StreamExtractorFieldSeason:
						if season, err := strconv.Atoi(value); err == nil {
							r.Season = season
							if len(r.Seasons) == 0 {
								r.Seasons = []int{season}
							}
						}
					case StreamExtractorFieldSite:
						r.Site = value
					case StreamExtractorFieldSize:
						r.Size = value
					case "title", StreamExtractorFieldTTitle:
						r.TTitle = value
					}
				}
			}
		}
	}

	if hdr != "" {
		if hdr_sep != "" {
			r.HDR = strings.Split(hdr, hdr_sep)
		} else {
			r.HDR = []string{hdr}
		}
	}

	if language != "" {
		if language_sep != "" {
			for lang := range strings.SplitSeq(language, language_sep) {
				lang = strings.TrimSpace(lang)
				if code, ok := language_to_code[strings.ToLower(lang)]; ok {
					lang = code
				}
				r.Languages = append(r.Languages, lang)
			}
		} else if code, ok := language_to_code[strings.ToLower(language)]; ok {
			r.Languages = []string{code}
		} else {
			r.Languages = []string{language}
		}
	}

	if stream.InfoHash != "" {
		r.Hash = stream.InfoHash
		r.File.Idx = stream.FileIndex
	}

	if stream.BehaviorHints != nil {
		if stream.BehaviorHints.Filename != "" {
			r.File.Name = stream.BehaviorHints.Filename
		}
		if stream.BehaviorHints.VideoSize != 0 {
			if r.File.Size == "" {
				r.File.Size = util.ToSize(stream.BehaviorHints.VideoSize)
			}
			if r.Size == "" {
				r.Size = r.File.Size
			}
		}
	}

	if r.File.Name != "" {
		r.File.Name = filepath.Base(strings.TrimSpace(r.File.Name))
	}

	if len(se.items) == 0 {
		r = fallbackStreamExtractor(r)
	}

	if r.Quality != "" {
		r.Quality = strings.Trim(r.Quality, " .-")
	}

	if r.Store.Code == "" && r.Store.Name != "" {
		r.Store.Code = strings.ToUpper(string(store.StoreName(strings.ToLower(nonAlphaRegex.ReplaceAllLiteralString(r.Store.Name, ""))).Code()))
	}
	if r.Store.Code != "" {
		r.Store.Code = strings.ToUpper(r.Store.Code)
		switch r.Store.Code {
		case "PKP":
			r.Store.Code = "PP"
		case "TRB":
			r.Store.Code = "TB"
		}

		r.Store.Name = string(store.StoreCode(strings.ToLower(r.Store.Code)).Name())
	}

	r.Result = r.Normalize()

	if r.Episode == -1 && len(r.Episodes) > 0 {
		r.Episode = r.Episodes[0]
	}

	if r.Season == -1 && len(r.Seasons) > 0 {
		r.Season = r.Seasons[0]
	}

	return r
}
