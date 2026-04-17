package stremio_transformer

import (
	"fmt"
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

type StreamExtractorResultIndexer struct {
	ID   string
	Host string
	Name string
}

type StreamExtractorResultKind = string

const (
	StreamExtractorResultKindTorz StreamExtractorResultKind = "torz"
	StreamExtractorResultKindNewz StreamExtractorResultKind = "newz"
)

type StreamExtractorResultBitRate int64

type StreamExtractorResult struct {
	*ptt.Result

	Addon     StreamExtractorResultAddon
	Category  string
	Date      time.Time
	Episode   int
	File      StreamExtractorResultFile
	Hash      string
	Indexer   StreamExtractorResultIndexer `expr:"-"`
	IsPrivate bool
	Kind      StreamExtractorResultKind
	Raw       StreamExtractorResultRaw
	Season    int
	Seeders   int
	Store     StreamExtractorResultStore
	TTitle    string `expr:"-"`

	BitRate   StreamExtractorResultBitRate
	Duration  time.Duration
	Subtitles []string
}

func (r *StreamExtractorResult) Age() string {
	if r.Date.IsZero() {
		return ""
	}
	return util.FormatDuration(time.Since(r.Date), 1)
}

func (br StreamExtractorResultBitRate) String() string {
	if br <= 0 {
		return ""
	}
	Bps := float64(br) / 8
	switch {
	case Bps >= 1_000_000:
		return fmt.Sprintf("%.1f MB/s", Bps/1_000_000)
	case Bps >= 1_000:
		return fmt.Sprintf("%.0f KB/s", Bps/1_000)
	default:
		return fmt.Sprintf("%d B/s", int64(Bps))
	}
}

var language_to_code = map[string]string{
	"dubbed":      "dub",
	"dub":         "dub",
	"dual audio":  "daud",
	"daud":        "daud",
	"multi audio": "maud",
	"maud":        "maud",
	"multi subs":  "msub",
	"msub":        "msub",

	"english":    "en",
	"eng":        "en",
	"en":         "en",
	"'eng'":      "en",
	"en-us":      "en",
	"🇬🇧":         "en",
	"🇺🇸":         "en",
	"japanese":   "ja",
	"jpn":        "ja",
	"ja":         "ja",
	"🇯🇵":         "ja",
	"russian":    "ru",
	"rus":        "ru",
	"ru":         "ru",
	"🇷🇺":         "ru",
	"italian":    "it",
	"ita":        "it",
	"it":         "it",
	"🇮🇹":         "it",
	"portuguese": "pt",
	"por":        "pt",
	"pt":         "pt",
	"🇵🇹":         "pt",
	"🇧🇷":         "pt",
	"spanish":    "es",
	"spa":        "es",
	"es":         "es",
	"spa(la)":    "es-419",
	"es-419":     "es-419",
	"spa(mx)":    "es-mx",
	"es-mx":      "es-mx",
	"🇪🇸":         "es",
	"latino":     "es-419",
	"🇲🇽":         "es-mx",
	"korean":     "ko",
	"kor":        "ko",
	"ko":         "ko",
	"🇰🇷":         "ko",
	"chinese":    "zh",
	"zho":        "zh",
	"zh":         "zh",
	"zh-hans":    "zh",
	"🇨🇳":         "zh",
	"taiwanese":  "zh-tw",
	"zh-tw":      "zh-tw",
	"🇹🇼":         "zh-tw",
	"french":     "fr",
	"fra":        "fr",
	"fr":         "fr",
	"🇫🇷":         "fr",
	"german":     "de",
	"deu":        "de",
	"de":         "de",
	"🇩🇪":         "de",
	"dutch":      "nl",
	"nld":        "nl",
	"nl":         "nl",
	"🇳🇱":         "nl",
	"hindi":      "hi",
	"hin":        "hi",
	"hi":         "hi",
	"🇮🇳":         "hi",
	"telugu":     "te",
	"tel":        "te",
	"te":         "te",
	"tamil":      "ta",
	"tam":        "ta",
	"ta":         "ta",
	"malayalam":  "ml",
	"mal":        "ml",
	"ml":         "ml",
	"kannada":    "kn",
	"kan":        "kn",
	"kn":         "kn",
	"marathi":    "mr",
	"mar":        "mr",
	"mr":         "mr",
	"gujarati":   "gu",
	"guj":        "gu",
	"gu":         "gu",
	"punjabi":    "pa",
	"pan":        "pa",
	"pa":         "pa",
	"bengali":    "bn",
	"ben":        "bn",
	"bn":         "bn",
	"🇧🇩":         "bn",
	"polish":     "pl",
	"pol":        "pl",
	"pl":         "pl",
	"pl-pl":      "pl",
	"🇵🇱":         "pl",
	"lithuanian": "lt",
	"lit":        "lt",
	"lt":         "lt",
	"🇱🇹":         "lt",
	"latvian":    "lv",
	"lav":        "lv",
	"lv":         "lv",
	"🇱🇻":         "lv",
	"estonian":   "et",
	"est":        "et",
	"et":         "et",
	"🇪🇪":         "et",
	"czech":      "cs",
	"ces":        "cs",
	"cs":         "cs",
	"🇨🇿":         "cs",
	"slovakian":  "sk",
	"slk":        "sk",
	"sk":         "sk",
	"🇸🇰":         "sk",
	"slovenian":  "sl",
	"slv":        "sl",
	"sl":         "sl",
	"🇸🇮":         "sl",
	"hungarian":  "hu",
	"hun":        "hu",
	"hu":         "hu",
	"🇭🇺":         "hu",
	"romanian":   "ro",
	"ron":        "ro",
	"ro":         "ro",
	"🇷🇴":         "ro",
	"bulgarian":  "bg",
	"bul":        "bg",
	"bg":         "bg",
	"🇧🇬":         "bg",
	"serbian":    "sr",
	"srp":        "sr",
	"sr":         "sr",
	"🇷🇸":         "sr",
	"croatian":   "hr",
	"hrv":        "hr",
	"hr":         "hr",
	"🇭🇷":         "hr",
	"ukrainian":  "uk",
	"ukr":        "uk",
	"uk":         "uk",
	"🇺🇦":         "uk",
	"greek":      "el",
	"ell":        "el",
	"el":         "el",
	"🇬🇷":         "el",
	"danish":     "da",
	"dan":        "da",
	"da":         "da",
	"🇩🇰":         "da",
	"finnish":    "fi",
	"fin":        "fi",
	"fi":         "fi",
	"🇫🇮":         "fi",
	"swedish":    "sv",
	"swe":        "sv",
	"sv":         "sv",
	"🇸🇪":         "sv",
	"norwegian":  "no",
	"nor":        "no",
	"no":         "no",
	"🇳🇴":         "no",
	"turkish":    "tr",
	"tur":        "tr",
	"tr":         "tr",
	"🇹🇷":         "tr",
	"arabic":     "ar",
	"ara":        "ar",
	"ar":         "ar",
	"🇸🇦":         "ar",
	"persian":    "fa",
	"fas":        "fa",
	"fa":         "fa",
	"🇮🇷":         "fa",
	"hebrew":     "he",
	"heb":        "he",
	"he":         "he",
	"🇮🇱":         "he",
	"vietnamese": "vi",
	"vie":        "vi",
	"vi":         "vi",
	"🇻🇳":         "vi",
	"indonesian": "id",
	"ind":        "id",
	"id":         "id",
	"🇮🇩":         "id",
	"malay":      "ms",
	"msa":        "ms",
	"ms":         "ms",
	"🇲🇾":         "ms",
	"thai":       "th",
	"tha":        "th",
	"th":         "th",
	"🇹🇭":         "th",
	"zxx":        "",
	"":           "",
}

func GetLangCode(lang string) string {
	if code, ok := language_to_code[strings.ToLower(lang)]; ok {
		return code
	}
	return ""
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
		case "DBD":
			r.Store.Code = "DR"
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
