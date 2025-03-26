package ptt

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type hProcessor func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta
type hTransformer func(title string, m *parseMeta, result map[string]*parseMeta)
type hMatchValidator func(input string, idxs []int) bool

type handler struct {
	Field         string
	Pattern       *regexp.Regexp
	ValidateMatch hMatchValidator
	Transform     hTransformer
	Process       hProcessor

	Remove        bool // remove
	KeepMatching  bool // !skipIfAlreadyFound
	SkipIfFirst   bool // skipIfFirst
	SkipFromTitle bool // skipFromTitle

	MatchGroup int // capture group to use as match
	ValueGroup int // capture group to use as value
}

func validate_and(validators ...hMatchValidator) hMatchValidator {
	return func(input string, idxs []int) bool {
		for _, validator := range validators {
			if !validator(input, idxs) {
				return false
			}
		}
		return true
	}
}

func validate_not_at_start() hMatchValidator {
	return func(input string, match []int) bool {
		return match[0] != 0
	}
}
func validate_not_at_end() hMatchValidator {
	return func(input string, match []int) bool {
		return match[1] != len(input)
	}
}

func validate_not_match(re *regexp.Regexp) hMatchValidator {
	return func(input string, match []int) bool {
		rv := input[match[0]:match[1]]
		return !re.MatchString(rv)
	}
}

func validate_match(re *regexp.Regexp) hMatchValidator {
	return func(input string, match []int) bool {
		rv := input[match[0]:match[1]]
		return re.MatchString(rv)
	}
}

func validate_matched_groups_are_same(indices ...int) hMatchValidator {
	return func(input string, match []int) bool {
		first := input[match[indices[0]*2]:match[indices[0]*2+1]]
		for _, index := range indices[1:] {
			other := input[match[index*2]:match[index*2+1]]
			if other != first {
				return false
			}
		}
		return true
	}
}

func to_value(value string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = value
	}
}

func to_lowercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToLower(m.value.(string))
	}
}

func to_uppercase() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.ToUpper(m.value.(string))
	}
}

func to_trimmed() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = strings.TrimSpace(m.value.(string))
	}
}

func to_clean_date() hTransformer {
	re := regexp.MustCompile(`(\d+)(?:st|nd|rd|th)`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = re.ReplaceAllString(v, "$1")
		}
	}
}

func to_clean_month() hTransformer {
	re := regexp.MustCompile(`(?i)(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = re.ReplaceAllStringFunc(v, func(str string) string {
				return str[0:3]
			})
		}
	}
}

func to_date(format string) hTransformer {
	seperatorRe := regexp.MustCompile(`[.\-/\\]`)
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if t, err := time.Parse(format, seperatorRe.ReplaceAllString(v, " ")); err == nil {
				m.value = t.Format("2006-01-02")
				return
			}
		}
		m.value = ""
	}
}

func to_year() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		vstr, ok := m.value.(string)
		if !ok {
			m.value = ""
			return
		}
		parts := non_digits_regex.Split(vstr, -1)
		if len(parts) == 1 {
			m.value = parts[0]
			return
		}
		start, end := parts[0], parts[1]
		endYear, err := strconv.Atoi(end)
		if err != nil {
			m.value = start
			return
		}
		startYear, err := strconv.Atoi(start)
		if err != nil {
			m.value = ""
			return
		}
		if endYear < 100 {
			endYear = endYear + startYear - startYear%100
		}
		if endYear <= startYear {
			m.value = ""
			return
		}
		m.value = strconv.Itoa(startYear) + "-" + strconv.Itoa(endYear)
	}
}

func to_int_range() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		v, ok := m.value.(string)
		if !ok {
			m.value = nil
			return
		}
		parts := strings.Split(strings.Trim(non_digits_regex.ReplaceAllString(v, " "), " "), " ")
		nums := make([]int, len(parts))
		for i, part := range parts {
			if num, err := strconv.Atoi(part); err == nil {
				nums[i] = num
			}
		}
		if len(nums) == 2 && nums[0] < nums[1] {
			seq := make([]int, nums[1]-nums[0]+1)
			for i := range seq {
				seq[i] = nums[0] + i
			}
			nums = seq
		}
		for i, num := range nums {
			if i != len(nums)-1 && num+1 != nums[i+1] {
				m.value = nil // not in sequence and ascending order
				return
			}
		}
		m.value = nums
	}
}

func to_with_suffix(suffix string) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			m.value = v + suffix
		} else {
			m.value = ""
		}
	}
}

func to_boolean() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		m.value = true
	}
}

type value_set[T comparable] struct {
	existMap map[T]struct{}
	values   []T
}

func (vs *value_set[any]) append(v any) *value_set[any] {
	if _, found := vs.existMap[v]; !found {
		vs.existMap[v] = struct{}{}
		vs.values = append(vs.values, v)
	}
	return vs
}

func (vs *value_set[any]) exists(v any) bool {
	_, found := vs.existMap[v]
	return found
}

func to_value_set(v any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*value_set[any]); ok {
			m.value = val.append(v)
		}
	}
}

func to_value_set_with_transform(to_v func(v string) any) hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if val, ok := m.value.(*value_set[any]); ok {
			m.value = val.append(to_v(m.mValue))
		}
	}
}

func to_int_array() hTransformer {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) {
		if v, ok := m.value.(string); ok {
			if num, err := strconv.Atoi(v); err == nil {
				m.value = []int{num}
				return
			}
		}
		m.value = []int{}
		return
	}
}

func remove_from_value(re *regexp.Regexp) hProcessor {
	return func(title string, m *parseMeta, _ map[string]*parseMeta) *parseMeta {
		if v, ok := m.value.(string); ok && v != "" {
			m.value = re.ReplaceAllString(v, "")
		}
		return m
	}
}

var handlers = []handler{
	// parser.add_handler("title", regex.compile(r"360.Degrees.of.Vision.The.Byakugan'?s.Blind.Spot", regex.IGNORECASE), none, {"remove": True}) # episode title
	// parser.add_handler("title", regex.compile(r"\b(?:INTEGRALE?|INTÉGRALE?|INTERNAL|HFR)\b", regex.IGNORECASE), none, {"remove": True})
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)360.Degrees.of.Vision.The.Byakugan'?s.Blind.Spot`),
		Remove:  true,
	},
	{
		Field:   "title",
		Pattern: regexp.MustCompile(`(?i)\b(?:INTEGRALE?|INTÉGRALE?|INTERNAL|HFR)\b`),
		Remove:  true,
	},

	// parser.add_handler("ppv", regex.compile(r"\bPPV\b", regex.IGNORECASE), boolean, {"skipFromTitle": True, "remove": True})
	// parser.add_handler("ppv", regex.compile(r"\b\W?Fight.?Nights?\W?\b", regex.IGNORECASE), boolean, {"skipFromTitle": True, "remove": False})
	{
		Field:         "ppv",
		Pattern:       regexp.MustCompile(`(?i)\bPPV\b`),
		Remove:        true,
		SkipFromTitle: true,
	},
	{
		Field:         "ppv",
		Pattern:       regexp.MustCompile(`(?i)\b\W?Fight.?Nights?\W?\b`),
		SkipFromTitle: true,
	},

	// parser.add_handler("site", regex.compile(r"^(www?[\.,][\w-]+\.[\w-]+(?:\.[\w-]+)?)\s+-\s*", regex.IGNORECASE), options={"skipFromTitle": True, "remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("site", regex.compile(r"^((?:www?[\.,])?[\w-]+\.[\w-]+(?:\.[\w-]+)*?)\s+-\s*", regex.IGNORECASE), options={"skipIfAlreadyFound": False})
	// parser.add_handler("site", regex.compile(r"\bwww.+rodeo\b", regex.IGNORECASE), lowercase, {"remove": True})
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)^(www?[\.,][\w-]+\.[\w-]+(?:\.[\w-]+)?)\s+-\s*`),
		KeepMatching:  true,
		SkipFromTitle: true,
		Remove:        true,
	},
	{
		Field:        "site",
		Pattern:      regexp.MustCompile(`(?i)^((?:www?[\.,])?[\w-]+\.[\w-]+(?:\.[\w-]+)*?)\s+-\s*`),
		KeepMatching: true,
	},
	{
		Field:     "site",
		Pattern:   regexp.MustCompile(`(?i)\bwww.+rodeo\b`),
		Transform: to_lowercase(),
		Remove:    true,
	},

	// parser.addHandler("episodeCode", /[[(]([a-z0-9]{8}|[A-Z0-9]{8})[\])](?=\.[a-zA-Z0-9]{1,5}$|$)/, uppercase, { remove: true });
	// parser.addHandler("episodeCode", /\[(?=[A-Z]+\d|\d+[A-Z])([A-Z0-9]{8})]/, uppercase, { remove: true });
	{
		Field:      "episodeCode",
		Pattern:    regexp.MustCompile(`([[(]([a-z0-9]{8}|[A-Z0-9]{8})[\])])(?:\.[a-zA-Z0-9]{1,5}$|$)`),
		Transform:  to_uppercase(),
		Remove:     true,
		MatchGroup: 1,
		ValueGroup: 2,
	},
	{
		Field:         "episodeCode",
		Pattern:       regexp.MustCompile(`\[([A-Z0-9]{8})]`),
		ValidateMatch: validate_match(regexp.MustCompile(`(?:[A-Z]+\d|\d+[A-Z])`)),
		Transform:     to_uppercase(),
		Remove:        true,
	},

	// parser.add_handler("resolution", regex.compile(r"\b(?:4k|2160p|1080p|720p|480p)(?!.*\b(?:4k|2160p|1080p|720p|480p)\b)", regex.IGNORECASE), transform_resolution, {"remove": True})
	{
		Field:      "resolution",
		Pattern:    regexp.MustCompile(`(?i)\b(?:4k|2160p|1080p|720p|480p)\b.+\b(4k|2160p|1080p|720p|480p)\b`),
		Transform:  to_lowercase(),
		Remove:     true,
		MatchGroup: 1,
	},
	// parser.addHandler("resolution", /\b[([]?4k[)\]]?\b/i, value("4k"), { remove: true });
	// parser.addHandler("resolution", /21600?[pi]/i, value("4k"), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("resolution", /[([]?3840x\d{4}[)\]]?/i, value("4k"), { remove: true });
	// parser.addHandler("resolution", /[([]?1920x\d{3,4}[)\]]?/i, value("1080p"), { remove: true });
	// parser.addHandler("resolution", /[([]?1280x\d{3}[)\]]?/i, value("720p"), { remove: true });
	// parser.addHandler("resolution", /[([]?\d{3,4}x(\d{3,4})[)\]]?/i, value("$1p"), { remove: true });
	// parser.addHandler("resolution", /(480|720|1080)0[pi]/i, value("$1p"), { remove: true });
	// parser.addHandler("resolution", /(?:BD|HD|M)(720|1080|2160)/, value("$1p"), { remove: true });
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)\b[([]?4k[)\]]?\b`),
		Transform: to_value("4k"),
		Remove:    true,
	},
	{
		Field:        "resolution",
		Pattern:      regexp.MustCompile(`(?i)21600?[pi]`),
		Transform:    to_value("4k"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)[([]?3840x\d{4}[)\]]?`),
		Transform: to_value("4k"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)[([]?1920x\d{3,4}[)\]]?`),
		Transform: to_value("1080p"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)[([]?1280x\d{3}[)\]]?`),
		Transform: to_value("720p"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)[([]?\d{3,4}x(\d{3,4})[)\]]?`),
		Transform: to_with_suffix("p"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(480|720|1080)0[pi]`),
		Transform: to_with_suffix("p"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|HD|M)(720|1080|2160)`),
		Transform: to_with_suffix("p"),
		Remove:    true,
	},
	// parser.addHandler("resolution", /(480|576|720|1080|2160)[pi]/i, value("$1p"), { remove: true });
	// parser.addHandler("resolution", /(?:^|\D)(\d{3,4})[pi]/i, value("$1p"), { remove: true });
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(480|576|720|1080|2160)[pi]`),
		Transform: to_with_suffix("p"),
		Remove:    true,
	},
	{
		Field:     "resolution",
		Pattern:   regexp.MustCompile(`(?i)(?:^|\D)(\d{3,4})[pi]`),
		Transform: to_with_suffix("p"),
		Remove:    true,
	},

	// parser.addHandler("date", /(?<=\W|^)([([]?(?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])\2(?:0[1-9]|[12][0-9]|3[01])[)\]]?)(?=\W|$)/, date("YYYY MM DD"), { remove: true });
	// parser.addHandler("date", /(?<=\W|^)([([]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:19[6-9]|20[012])[0-9][)\]]?)(?=\W|$)/, date("DD MM YYYY"), { remove: true });
	// parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])\2(?:19[6-9]|20[012])[0-9][)\]]?)(?=\W|$)/, date("MM DD YYYY"), { remove: true });
	// parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])\2(?:[0][1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/, date("MM DD YY"), { remove: true });
	// parser.addHandler("date", /(?<=\W)([([]?(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])\2(?:[0][1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/, date("DD MM YY"), { remove: true });
	// parser.addHandler("date", /(?<=\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\2(?:19[7-9]|20[012])[0-9][)\]]?)(?=\W|$)/i, date("DD MMM YYYY"), { remove: true });
	// parser.addHandler("date", /(?<=\W|^)([([]?(?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\2(?:0[1-9]|[0126789][0-9])[)\]]?)(?=\W|$)/i, date("DD MMM YY"), { remove: true });
	// parser.addHandler("date", /(?<=\W|^)([([]?20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01])[)\]]?)(?=\W|$)/, date("YYYYMMDD"), { remove: true });
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)([([]?((?:19[6-9]|20[012])[0-9]([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01]))[)\]]?)(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(3, 4),
		Transform:     to_date("2006 01 02"),
		Remove:        true,
		ValueGroup:    2,
		MatchGroup:    1,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W|^)[([]?((?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:19[6-9]|20[012])[0-9])[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_date("02 01 2006"),
		Remove:        true,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)[([]?((?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:19[6-9]|20[012])[0-9])[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_date("01 02 2006"),
		Remove:        true,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)[([]?((?:0[1-9]|1[012])([. \-/\\])(?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:[0][1-9]|[0126789][0-9]))[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_date("01 02 06"),
		Remove:        true,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?:\W)[([]?((?:0[1-9]|[12][0-9]|3[01])([. \-/\\])(?:0[1-9]|1[012])([. \-/\\])(?:[0][1-9]|[0126789][0-9]))[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform:     to_date("02 01 06"),
		Remove:        true,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)[([]?((?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:19[7-9]|20[012])[0-9])[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform: func() hTransformer {
			cd := to_clean_date()
			cm := to_clean_month()
			td := to_date("_2 Jan 2006")
			return func(title string, m *parseMeta, result map[string]*parseMeta) {
				cd(title, m, result)
				cm(title, m, result)
				td(title, m, result)
			}
		}(),
		Remove: true,
	},
	{
		Field:         "date",
		Pattern:       regexp.MustCompile(`(?i)(?:\W|^)[([]?((?:0?[1-9]|[12][0-9]|3[01])[. ]?(?:st|nd|rd|th)?([. \-/\\])(?:feb(?:ruary)?|jan(?:uary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sept?(?:ember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)([. \-/\\])(?:0[1-9]|[0126789][0-9]))[)\]]?(?:\W|$)`),
		ValidateMatch: validate_matched_groups_are_same(2, 3),
		Transform: func() hTransformer {
			cd := to_clean_date()
			cm := to_clean_month()
			td := to_date("_2 Jan 06")
			return func(title string, m *parseMeta, result map[string]*parseMeta) {
				cd(title, m, result)
				cm(title, m, result)
				td(title, m, result)
			}
		}(),
		Remove: true,
	},
	{
		Field:     "date",
		Pattern:   regexp.MustCompile(`(?:\W|^)[([]?(20[012][0-9](?:0[1-9]|1[012])(?:0[1-9]|[12][0-9]|3[01]))[)\]]?(?:\W|$)`),
		Transform: to_date("20060102"),
		Remove:    true,
	},

	// parser.addHandler("year", /[([*]?[ .]?((?:19\d|20[012])\d[ .]?-[ .]?(?:19\d|20[012])\d)(?:\s?[*)\]])?/, yearRange, { remove: true });
	// parser.addHandler("year", /[([*][ .]?((?:19\d|20[012])\d[ .]?-[ .]?\d{2})(?:\s?[*)\]])?/, yearRange, { remove: true });
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`[([*]?[ .]?((?:19\d|20[012])\d[ .]?-[ .]?(?:19\d|20[012])\d)(?:\s?[*)\]])?`),
		Transform: func() hTransformer {
			ty := to_year()
			return func(title string, m *parseMeta, result map[string]*parseMeta) {
				ty(title, m, result)
				if _, ok := result["complete"]; !ok && strings.Contains(m.value.(string), "-") {
					cm := *m
					cm.value = true
					result["complete"] = &cm
				}
			}
		}(),
		Remove: true,
	},
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`[([*][ .]?((?:19\d|20[012])\d[ .]?-[ .]?\d{2})(?:\s?[*)\]])?`),
		Transform: func() hTransformer {
			ty := to_year()
			return func(title string, m *parseMeta, result map[string]*parseMeta) {
				ty(title, m, result)
				if _, ok := result["complete"]; !ok && strings.Contains(m.value.(string), "-") {
					cm := *m
					cm.value = true
					result["complete"] = &cm
				}
			}
		}(),
		Remove: true,
	},
	// parser.add_handler("year", regex.compile(r"\b(20[0-9]{2}|2100)(?!\D*\d{4}\b)"), integer, {"remove": True})
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`[([*]?\b(20[0-9]{2}|2100)[*)\]]?`),
		ValidateMatch: func() hMatchValidator {
			re := regexp.MustCompile(`(?:\D*\d{4}\b)`)
			return func(input string, match []int) bool {
				return !re.MatchString(input[match[3]:])
			}
		}(),
		Transform: to_year(),
		Remove:    true,
	},
	// parser.addHandler("year", /[([*]?(?!^)(?<!\d|Cap[. ]?)((?:19\d|20[012])\d)(?!\d|kbps)[*)\]]?/i, integer, { remove: true });
	// parser.addHandler("year", /^[([]?((?:19\d|20[012])\d)(?!\d|kbps)[)\]]?/i, integer, { remove: true });
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`(?:[([*]|.)((?:\d|Cap[. ]?)?(?:19\d|20[012])\d(?:\d|kbps)?)[*)\]]?`),
		ValidateMatch: func(input string, match []int) bool {
			if match[0] < 2 {
				return false
			}
			return len(input[match[2]:match[3]]) == 4
		},
		Transform: to_year(),
		Remove:    true,
	},
	{
		Field:   "year",
		Pattern: regexp.MustCompile(`^[([]?((?:19\d|20[012])\d)(?:\d|kbps)?[)\]]?`),
		ValidateMatch: func(input string, match []int) bool {
			mValue := input[match[0]:match[1]]
			if len(mValue) == 4 {
				return match[0] != 0
			}
			return len(strings.Trim(mValue, "()[]")) == 4
		},
		Transform: to_year(),
		Remove:    true,
	},

	// parser.addHandler("extended", /EXTENDED/, boolean);
	// parser.addHandler("extended", /- Extended/i, boolean);
	{
		Field:     "extended",
		Pattern:   regexp.MustCompile(`EXTENDED`),
		Transform: to_boolean(),
	},
	{
		Field:     "extended",
		Pattern:   regexp.MustCompile(`(?i)- Extended`),
		Transform: to_boolean(),
	},

	// parser.add_handler("edition", regex.compile(r"\b\d{2,3}(th)?[\.\s\-\+_\/(),]Anniversary[\.\s\-\+_\/(),](Edition|Ed)?\b", regex.IGNORECASE), value("Anniversary Edition"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bUltimate[\.\s\-\+_\/(),]Edition\b", regex.IGNORECASE), value("Ultimate Edition"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bExtended[\.\s\-\+_\/(),]Director(\')?s\b", regex.IGNORECASE), value("Directors Cut"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\b(custom.?)?Extended\b", regex.IGNORECASE), value("Extended Edition"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bDirector(\')?s.?Cut\b", regex.IGNORECASE), value("Directors Cut"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bCollector(\')?s\b", regex.IGNORECASE), value("Collectors Edition"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bTheatrical\b", regex.IGNORECASE), value("Theatrical"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\buncut(?!.gems)\b", regex.IGNORECASE), value("Uncut"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bIMAX\b", regex.IGNORECASE), value("IMAX"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\b\.Diamond\.\b", regex.IGNORECASE), value("Diamond Edition"), {"remove": True})
	// parser.add_handler("edition", regex.compile(r"\bRemaster(?:ed)?\b", regex.IGNORECASE), value("Remastered"), {"remove": True, "skipIfAlreadyFound": True})
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b\d{2,3}(th)?[\.\s\-\+_\/(),]Anniversary[\.\s\-\+_\/(),](Edition|Ed)?`),
		Transform: to_value("Anniversary Edition"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bUltimate[\.\s\-\+_\/(),]Edition`),
		Transform: to_value("Ultimate Edition"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bExtended[\.\s\-\+_\/(),]Director(?:\')?s`),
		Transform: to_value("Directors Cut"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b(?:custom.?)?Extended`),
		Transform: to_value("Extended Edition"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bDirector(?:\')?s.?Cut`),
		Transform: to_value("Directors Cut"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bCollector(\')?s`),
		Transform: to_value("Collectors Edition"),
		Remove:    true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\bTheatrical`),
		Transform: to_value("Theatrical"),
		Remove:    true,
	},
	{
		Field:         "edition",
		Pattern:       regexp.MustCompile(`(?i)\buncut(?:.gems)?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:.gems)`)),
		Transform:     to_value("Uncut"),
		Remove:        true,
	},
	{
		Field:         "edition",
		Pattern:       regexp.MustCompile(`(?i)\bIMAX`),
		Transform:     to_value("IMAX"),
		Remove:        true,
		SkipFromTitle: true,
	},
	{
		Field:     "edition",
		Pattern:   regexp.MustCompile(`(?i)\b\.Diamond\.\b`),
		Transform: to_value("Diamond Edition"),
		Remove:    true,
	},
	{
		Field:        "edition",
		Pattern:      regexp.MustCompile(`(?i)\bRemaster(?:ed)?`),
		Transform:    to_value("Remastered"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field: "edition",
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			if m.value == nil {
				return m
			}
			switch m.value {
			case "Remastered":
				if _, ok := result["remastered"]; !ok {
					result["remastered"] = &parseMeta{mIndex: m.mIndex, mValue: m.mValue, value: true}
				}
			}
			return m
		},
	},

	// parser.add_handler("upscaled", regex.compile(r"\b(?:AI.?)?(Upscal(ed?|ing)|Enhanced?)\b", regex.IGNORECASE), boolean)
	// parser.add_handler("upscaled", regex.compile(r"\b(?:iris2|regrade|ups(uhd|fhd|hd|4k))\b", regex.IGNORECASE), boolean)
	// parser.add_handler("upscaled", regex.compile(r"\b\.AI\.\b", regex.IGNORECASE), boolean)
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b(?:AI.?)?(Upscal(ed?|ing)|Enhanced?)\b`),
		Transform: to_boolean(),
	},
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b(?:iris2|regrade|ups(uhd|fhd|hd|4k))\b`),
		Transform: to_boolean(),
	},
	{
		Field:     "upscaled",
		Pattern:   regexp.MustCompile(`(?i)\b\.AI\.\b`),
		Transform: to_boolean(),
	},

	// parser.addHandler("convert", /CONVERT/, boolean);
	{
		Field:     "convert",
		Pattern:   regexp.MustCompile(`CONVERT`),
		Transform: to_boolean(),
	},

	// parser.addHandler("hardcoded", /HC|HARDCODED/, boolean);
	{
		Field:     "hardcoded",
		Pattern:   regexp.MustCompile(`HC|HARDCODED`),
		Transform: to_boolean(),
	},

	// parser.addHandler("proper", /(?:REAL.)?PROPER/, boolean);
	{
		Field:     "proper",
		Pattern:   regexp.MustCompile(`(?:REAL.)?PROPER`),
		Transform: to_boolean(),
	},

	// parser.addHandler("repack", /REPACK|RERIP/, boolean);
	{
		Field:     "repack",
		Pattern:   regexp.MustCompile(`REPACK|RERIP`),
		Transform: to_boolean(),
	},

	// parser.addHandler("retail", /\bRetail\b/i, boolean);
	{
		Field:     "retail",
		Pattern:   regexp.MustCompile(`(?i)\bRetail\b`),
		Transform: to_boolean(),
	},

	// x parser.addHandler("remastered", /\bRemaster(?:ed)?\b/i, boolean);
	// parser.addHandler("remastered", /\b[[(]?REKONSTRUKCJA[\])]?\b/i, boolean);
	// {
	// 	Field:     "remastered",
	// 	Pattern:   regexp.MustCompile(`(?i)\bRemaster(?:ed)?\b`),
	// 	Transform: to_boolean(),
	// },
	{
		Field:     "remastered",
		Pattern:   regexp.MustCompile(`(?i)\b[[(]?REKONSTRUKCJA[\])]?\b`),
		Transform: to_boolean(),
	},

	// parser.add_handler("documentary", regex.compile(r"\bDOCU(?:menta?ry)?\b", regex.IGNORECASE), boolean, {"skipFromTitle": True})
	{
		Field:         "documentary",
		Pattern:       regexp.MustCompile(`(?i)\bDOCU(?:menta?ry)?\b`),
		Transform:     to_boolean(),
		SkipFromTitle: true,
	},

	// parser.addHandler("unrated", /\bunrated|uncensored\b/i, boolean);
	{
		Field:     "unrated",
		Pattern:   regexp.MustCompile(`(?i)\bunrated|uncensored\b`),
		Transform: to_boolean(),
	},

	// x parser.addHandler("region", /R\d\b/, none, { skipIfFirst: true });
	// {
	// 	Field:       "region",
	// 	Pattern:     regexp.MustCompile(`R\d\b`),
	// 	SkipIfFirst: true,
	// },
	// parser.add_handler("region", regex.compile(r"R\dJ?\b"), uppercase, {"remove": True})
	// parser.add_handler("region", regex.compile(r"\b(PAL|NTSC|SECAM)\b", regex.IGNORECASE), uppercase, {"remove": True})
	{
		Field:       "region",
		Pattern:     regexp.MustCompile(`R\dJ?\b`),
		Remove:      true,
		SkipIfFirst: true,
	},
	{
		Field:     "region",
		Pattern:   regexp.MustCompile(`\b(PAL|NTSC|SECAM)\b`),
		Transform: to_uppercase(),
		Remove:    true,
	},

	// parser.addHandler("source", /\b(?:H[DQ][ .-]*)?CAM(?:H[DQ])?(?:[ .-]*Rip)?\b/i, value("CAM"), { remove: true });
	// parser.addHandler("source", /\b(?:H[DQ][ .-]*)?S[ .-]+print/i, value("CAM"), { remove: true });
	// parser.addHandler("source", /\b(?:HD[ .-]*)?T(?:ELE)?S(?:YNC)?(?:Rip)?\b/i, value("TeleSync"), { remove: true });
	// parser.addHandler("source", /\b(?:HD[ .-]*)?T(?:ELE)?C(?:INE)?(?:Rip)?\b/, value("TeleCine"), { remove: true });
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?CAM(?:H[DQ])?(?:[ .-]*Rip)?\b`),
		Transform: to_value("CAM"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:H[DQ][ .-]*)?S[ .-]+print`),
		Transform: to_value("CAM"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:HD[ .-]*)?T(?:ELE)?S(?:YNC)?(?:Rip)?\b`),
		Transform: to_value("TeleSync"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`\b(?:HD[ .-]*)?T(?:ELE)?C(?:INE)?(?:Rip)?\b`),
		Transform: to_value("TeleCine"),
		Remove:    true,
	},
	// parser.add_handler("quality", regex.compile(r"\b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b", regex.IGNORECASE), value("SCR"), {"remove": True})
	// parser.add_handler("quality", regex.compile(r"\bP(?:RE)?-?(HD|DVD)(?:Rip)?\b", regex.IGNORECASE), value("SCR"), {"remove": True})
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:DVD?|BD|BR|HD)?[ .-]*Scr(?:eener)?\b`),
		Transform: to_value("SCR"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bP(?:RE)?-?(HD|DVD)(?:Rip)?\b`),
		Transform: to_value("SCR"),
		Remove:    true,
	},
	// x parser.addHandler("source", /\b(?:DVD?|BD|BR)?[ .-]*Scr(?:eener)?\b/i, value("SCR"), { remove: true });
	// x parser.addHandler("source", /\bP(?:re)?DVD(?:Rip)?\b/i, value("SCR"), { remove: true });
	// {
	// 	Field:     "quality",
	// 	Pattern:   regexp.MustCompile(`(?i)\b(?:DVD?|BD|BR)?[ .-]*Scr(?:eener)?\b`),
	// 	Transform: to_value("SCR"),
	// 	Remove:    true,
	// },
	// {
	// 	Field:     "quality",
	// 	Pattern:   regexp.MustCompile(`(?i)\bP(?:re)?DVD(?:Rip)?\b`),
	// 	Transform: to_value("SCR"),
	// 	Remove:    true,
	// },
	// parser.addHandler("source", /\bBlu[ .-]*Ray\b(?=.*remux)/i, value("BluRay REMUX"), { remove: true });
	{
		Field:      "quality",
		Pattern:    regexp.MustCompile(`(?i)\b(Blu[ .-]*Ray)\b(?:.*remux)`),
		Transform:  to_value("BluRay REMUX"),
		Remove:     true,
		MatchGroup: 1,
	},
	// parser.addHandler("source", /(?:BD|BR|UHD)[- ]?remux/i, value("BluRay REMUX"), { remove: true });
	// parser.addHandler("source", /(?<=remux.*)\bBlu[ .-]*Ray\b/i, value("BluRay REMUX"), { remove: true });
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)(?:BD|BR|UHD)[- ]?remux`),
		Transform: to_value("BluRay REMUX"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)(?:remux.*)\bBlu[ .-]*Ray\b`),
		Transform: to_value("BluRay REMUX"),
		Remove:    true,
	},
	// parser.add_handler("quality", regex.compile(r"\bremux\b", regex.IGNORECASE), value("REMUX"), {"remove": True})
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bremux\b`),
		Transform: to_value("REMUX"),
		Remove:    true,
	},
	// parser.addHandler("source", /\bBlu[ .-]*Ray\b(?![ .-]*Rip)/i, value("BluRay"), { remove: true });
	// parser.addHandler("source", /\bUHD[ .-]*Rip\b/i, value("UHDRip"), { remove: true });
	// parser.addHandler("source", /\bHD[ .-]*Rip\b/i, value("HDRip"), { remove: true });
	// parser.addHandler("source", /\bMicro[ .-]*HD\b/i, value("HDRip"), { remove: true });
	// parser.addHandler("source", /\b(?:BR|Blu[ .-]*Ray)[ .-]*Rip\b/i, value("BRRip"), { remove: true });
	// parser.addHandler("source", /\bBD[ .-]*Rip\b|\bBDR\b|\bBD-RM\b|[[(]BD[\]) .,-]/i, value("BDRip"), { remove: true });
	// parser.addHandler("source", /\b(?:HD[ .-]*)?DVD[ .-]*Rip\b/i, value("DVDRip"), { remove: true });
	// parser.addHandler("source", /\bVHS[ .-]*Rip\b/i, value("DVDRip"), { remove: true });
	{
		Field:   "quality",
		Pattern: regexp.MustCompile(`(?i)\bBlu[ .-]*Ray\b(?:[ .-]*Rip)?`),
		ValidateMatch: func(input string, match []int) bool {
			return !strings.HasSuffix(strings.ToLower(input[match[0]:match[1]]), "rip")
		},
		Transform: to_value("BluRay"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bUHD[ .-]*Rip\b`),
		Transform: to_value("UHDRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD[ .-]*Rip\b`),
		Transform: to_value("HDRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bMicro[ .-]*HD\b`),
		Transform: to_value("HDRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:BR|Blu[ .-]*Ray)[ .-]*Rip\b`),
		Transform: to_value("BRRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bBD[ .-]*Rip\b|\bBDR\b|\bBD-RM\b|[[(]BD[\]) .,-]`),
		Transform: to_value("BDRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\b(?:HD[ .-]*)?DVD[ .-]*Rip\b`),
		Transform: to_value("DVDRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bVHS[ .-]*Rip\b`),
		Transform: to_value("DVDRip"),
		Remove:    true,
	},
	// parser.addHandler("source", /\bDVD(?:R\d?)?\b/i, value("DVD"), { remove: true });
	// parser.addHandler("source", /\bVHS\b/i, value("DVD"), { remove: true, skipIfFirst: true });
	// parser.addHandler("source", /\bPPVRip\b/i, value("PPVRip"), { remove: true });
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bDVD(?:R\d?)?\b`),
		Transform: to_value("DVD"),
		Remove:    true,
	},
	{
		Field:       "quality",
		Pattern:     regexp.MustCompile(`(?i)\bVHS\b`),
		Transform:   to_value("DVD"),
		Remove:      true,
		SkipIfFirst: true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bPPVRip\b`),
		Transform: to_value("PPVRip"),
		Remove:    true,
	},
	// parser.add_handler("quality", regex.compile(r"\bHD.?TV.?Rip\b", regex.IGNORECASE), value("HDTVRip"), {"remove": True})
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD.?TV.?Rip\b`),
		Transform: to_value("HDTVRip"),
		Remove:    true,
	},
	// x parser.addHandler("source", /\bHD[ .-]*TV(?:Rip)?\b/i, value("HDTV"), { remove: true });
	// {
	// 	Field:     "quality",
	// 	Pattern:   regexp.MustCompile(`(?i)\bHD[ .-]*TV(?:Rip)?\b`),
	// 	Transform: to_value("HDTV"),
	// 	Remove:    true,
	// },
	// parser.addHandler("source", /\bDVB[ .-]*(?:Rip)?\b/i, value("HDTV"), { remove: true });
	// parser.addHandler("source", /\bSAT[ .-]*Rips?\b/i, value("SATRip"), { remove: true });
	// parser.addHandler("source", /\bTVRips?\b/i, value("TVRip"), { remove: true });
	// parser.addHandler("source", /\bR5\b/i, value("R5"), { remove: true });
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bDVB[ .-]*(?:Rip)?\b`),
		Transform: to_value("HDTV"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bSAT[ .-]*Rips?\b`),
		Transform: to_value("SATRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bTVRips?\b`),
		Transform: to_value("TVRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bR5\b`),
		Transform: to_value("R5"),
		Remove:    true,
	},
	// parser.add_handler("quality", regex.compile(r"\bWEB[ .-]*Rip\b", regex.IGNORECASE), value("WEBRip"), {"remove": True})
	// parser.add_handler("quality", regex.compile(r"\bWEB[ .-]?DL[ .-]?Rip\b", regex.IGNORECASE), value("WEB-DLRip"), {"remove": True})
	// parser.add_handler("quality", regex.compile(r"\bWEB[ .-]*(DL|.BDrip|.DLRIP)\b", regex.IGNORECASE), value("WEB-DL"), {"remove": True})
	// parser.add_handler("quality", regex.compile(r"\b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b", regex.IGNORECASE), value("WEB"), {"remove": True, "skipFromTitle": True})  #
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*Rip\b`),
		Transform: to_value("WEBRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]?DL[ .-]?Rip\b`),
		Transform: to_value("WEB-DLRip"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*(DL|.BDrip|.DLRIP)\b`),
		Transform: to_value("WEB-DL"),
		Remove:    true,
	},
	// {
	// 	Field:         "quality",
	// 	Pattern:       regexp.MustCompile(`(?i)\b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b`),
	// 	ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b`)),
	// 	Transform:     to_value("WEB"),
	// 	Remove:        true,
	// 	SkipFromTitle: true,
	// },
	// x parser.addHandler("source", /\bWEB[ .-]*DL(?:Rip)?\b/i, value("WEB-DL"), { remove: true });
	// x parser.addHandler("source", /\bWEB[ .-]*Rip\b/i, value("WEBRip"), { remove: true });
	// parser.addHandler("source", /\b(?:DL|WEB|BD|BR)MUX\b/i, { remove: true });
	// {
	// 	Field:     "quality",
	// 	Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*DL(?:Rip)?\b`),
	// 	Transform: to_value("WEB-DL"),
	// 	Remove:    true,
	// },
	// {
	// 	Field:     "quality",
	// 	Pattern:   regexp.MustCompile(`(?i)\bWEB[ .-]*Rip\b`),
	// 	Transform: to_value("WEBRip"),
	// 	Remove:    true,
	// },
	{
		Field:   "quality",
		Pattern: regexp.MustCompile(`(?i)\b(?:DL|WEB|BD|BR)MUX\b`),
		Remove:  true,
	},
	// parser.addHandler("source", /\b(DivX|XviD)\b/, { remove: true });
	{
		Field:   "quality",
		Pattern: regexp.MustCompile(`\b(DivX|XviD)\b`),
		// Remove: true,
	},
	// parser.add_handler("quality", regex.compile(r"\b(?<!\w.)WEB\b|\bWEB(?!([ \.\-\(\],]+\d))\b", regex.IGNORECASE), value("WEB"), {"remove": True, "skipFromTitle": True})
	{
		Field:         "quality",
		Pattern:       regexp.MustCompile(`(?i)\b(?:\w.)?WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:\w.)WEB\b|\bWEB(?:(?:[ \.\-\(\],]+\d))\b`)),
		Transform:     to_value("WEB"),
		Remove:        true,
		SkipFromTitle: true,
	},
	// parser.add_handler("quality", regex.compile(r"\bPDTV\b", regex.IGNORECASE), value("PDTV"), {"remove": True})
	// parser.add_handler("quality", regex.compile(r"\bHD(.?TV)?\b", regex.IGNORECASE), value("HDTV"), {"remove": True})
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bPDTV\b`),
		Transform: to_value("PDTV"),
		Remove:    true,
	},
	{
		Field:     "quality",
		Pattern:   regexp.MustCompile(`(?i)\bHD(?:.?TV)?\b`),
		Transform: to_value("HDTV"),
		Remove:    true,
	},

	// parser.addHandler("bitDepth", /(?:8|10|12)[- ]?bit/i, lowercase, { remove: true });
	// parser.addHandler("bitDepth", /\bhevc\s?10\b/i, value("10bit"));
	// parser.addHandler("bitDepth", /\bhdr10\b/i, value("10bit"));
	// parser.addHandler("bitDepth", /\bhi10\b/i, value("10bit"));
	// parser.addHandler("bitDepth", ({ result }) => {
	//     if (result.bitDepth) {
	//         result.bitDepth = result.bitDepth.replace(/[ -]/, "");
	//     }
	// });
	{
		Field:       "bitDepth",
		Pattern:     regexp.MustCompile(`(?i)(?:8|10|12)[- ]?bit`),
		Transform:   to_lowercase(),
		Remove:      true,
		SkipIfFirst: true,
	},
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhevc\s?10\b`),
		Transform: to_value("10bit"),
	},
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhdr10(?:\+|plus)?\b`),
		Transform: to_value("10bit"),
	},
	{
		Field:     "bitDepth",
		Pattern:   regexp.MustCompile(`(?i)\bhi10\b`),
		Transform: to_value("10bit"),
	},
	{
		Field:   "bitDepth",
		Process: remove_from_value(regexp.MustCompile(`[ -]`)),
	},

	// parser.addHandler("hdr", /\bDV\b|dolby.?vision|\bDoVi\b/i, uniqConcat(value("DV")), { remove: true, skipIfAlreadyFound: false });
	// parser.addHandler("hdr", /HDR10(?:\+|plus)/i, uniqConcat(value("HDR10+")), { remove: true, skipIfAlreadyFound: false });
	// parser.addHandler("hdr", /\bHDR(?:10)?\b/i, uniqConcat(value("HDR")), { remove: true, skipIfAlreadyFound: false });
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bDV\b|dolby.?vision|\bDoVi\b`),
		Transform:    to_value_set("DV"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)HDR10(?:\+|plus)`),
		Transform:    to_value_set("HDR10+"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bHDR(?:10)?\b`),
		Transform:    to_value_set("HDR"),
		Remove:       true,
		KeepMatching: true,
	},
	// parser.add_handler("hdr", regex.compile(r"\bSDR\b", regex.IGNORECASE), uniq_concat(value("SDR")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:        "hdr",
		Pattern:      regexp.MustCompile(`(?i)\bSDR\b`),
		Transform:    to_value_set("SDR"),
		Remove:       true,
		KeepMatching: true,
	},

	// parser.addHandler("threeD", /\b(3D)\b.*\b(Half-?SBS|H[-\\/]?SBS)\b/i, value("3D HSBS"));
	// parser.addHandler("threeD", /\bHalf.Side.?By.?Side\b/i, value("3D HSBS"));
	// parser.addHandler("threeD", /\b(3D)\b.*\b(Full-?SBS|SBS)\b/i, value("3D SBS"));
	// parser.addHandler("threeD", /\bSide.?By.?Side\b/i, value("3D SBS"));
	// parser.addHandler("threeD", /\b(3D)\b.*\b(Half-?OU|H[-\\/]?OU)\b/i, value("3D HOU"));
	// parser.addHandler("threeD", /\bHalf.?Over.?Under\b/i, value("3D HOU"));
	// parser.addHandler("threeD", /\b(3D)\b.*\b(OU)\b/i, value("3D OU"));
	// parser.addHandler("threeD", /\bOver.?Under\b/i, value("3D OU"));
	// parser.addHandler("threeD", /\b((?:BD)?3D)\b/i, value("3D"), { skipIfFirst: true });
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\b(3D)\b.*\b(Half-?SBS|H[-\\/]?SBS)\b`),
		Transform: to_value("3D HSBS"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\bHalf.Side.?By.?Side\b`),
		Transform: to_value("3D HSBS"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\b(3D)\b.*\b(Full-?SBS|SBS)\b`),
		Transform: to_value("3D SBS"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\bSide.?By.?Side\b`),
		Transform: to_value("3D SBS"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\b(3D)\b.*\b(Half-?OU|H[-\\/]?OU)\b`),
		Transform: to_value("3D HOU"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\bHalf.?Over.?Under\b`),
		Transform: to_value("3D HOU"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\b(3D)\b.*\b(OU)\b`),
		Transform: to_value("3D OU"),
	},
	{
		Field:     "threeD",
		Pattern:   regexp.MustCompile(`(?i)\bOver.?Under\b`),
		Transform: to_value("3D OU"),
	},
	{
		Field:       "threeD",
		Pattern:     regexp.MustCompile(`(?i)\b((?:BD)?3D)\b`),
		Transform:   to_value("3D"),
		SkipIfFirst: true,
	},

	// parser.addHandler("codec", /\b[xh][-. ]?26[45]/i, lowercase, { remove: true });
	// parser.addHandler("codec", /\bhevc(?:\s?10)?\b/i, value("hevc"), { remove: true, skipIfAlreadyFound: false });
	// parser.addHandler("codec", /\b(?:dvix|mpeg2|divx|xvid|avc)\b/i, lowercase, { remove: true, skipIfAlreadyFound: false });
	// parser.addHandler("codec", ({ result }) => {
	//     if (result.codec) {
	//         result.codec = result.codec.replace(/[ .-]/, "");
	//     }
	// });
	{
		Field:     "codec",
		Pattern:   regexp.MustCompile(`(?i)\b[xh][-. ]?26[45]`),
		Transform: to_lowercase(),
		Remove:    true,
	},
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\bhevc(?:\s?10)?\b`),
		Transform:    to_value("hevc"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:        "codec",
		Pattern:      regexp.MustCompile(`(?i)\b(?:dvix|mpeg2|divx|xvid|avc)\b`),
		Transform:    to_lowercase(),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:   "codec",
		Process: remove_from_value(regexp.MustCompile(`[ .-]`)),
	},

	// parser.add_handler("channels", regex.compile(r"\bDDP?5[ \.\_]1\b", regex.IGNORECASE), uniq_concat(value("5.1")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\b5\.1(ch)?\b", regex.IGNORECASE), uniq_concat(value("5.1")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\b7[\.\- ]1(.?ch(annel)?)?\b", regex.IGNORECASE), uniq_concat(value("7.1")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\b2\.0\b", regex.IGNORECASE), uniq_concat(value("2.0")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\bstereo\b", regex.IGNORECASE), uniq_concat(value("stereo")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\bmono\b", regex.IGNORECASE), uniq_concat(value("mono")), {"remove": False})
	// parser.add_handler("channels", regex.compile(r"\b(?:x[2-4]|5[\W]1(?:x[2-4])?)\b", regex.IGNORECASE), uniq_concat(value("5.1")), {"remove": True})
	// parser.add_handler("channels", regex.compile(r"\b2\.0(?:x[2-4])\b", regex.IGNORECASE), uniq_concat(value("2.0")), {"remove": True})
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\bDDP?5[ ._]1\b`),
		Transform: to_value_set("5.1"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\b5\.1(?:ch)?\b`),
		Transform: to_value_set("5.1"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\b7[.\- ]1(?:.?ch(?:annel)?)?\b`),
		Transform: to_value_set("7.1"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\b2\.0\b`),
		Transform: to_value_set("2.0"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\bstereo\b`),
		Transform: to_value_set("stereo"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\bmono\b`),
		Transform: to_value_set("mono"),
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\b(?:x[2-4]|5[\W]1(?:x[2-4])?)\b`),
		Transform: to_value_set("5.1"),
		Remove:    true,
	},
	{
		Field:     "channels",
		Pattern:   regexp.MustCompile(`(?i)\b2\.0(?:x[2-4])\b`),
		Transform: to_value_set("2.0"),
		Remove:    true,
	},

	// parser.add_handler("audio", regex.compile(r"\bDDP5[ \.\_]1\b", regex.IGNORECASE), uniq_concat(value("Dolby Digital Plus")), {"remove": True, "skipIfFirst": True})
	// parser.add_handler("audio", regex.compile(r"\b(?!.+HR)(DTS.?HD.?Ma(ster)?|DTS.?X)\b", regex.IGNORECASE), uniq_concat(value("DTS Lossless")), {"remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("audio", regex.compile(r"\bDTS(?!(.?HD.?Ma(ster)?|.X)).?(HD.?HR|HD)?\b", regex.IGNORECASE), uniq_concat(value("DTS Lossy")), {"remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("audio", regex.compile(r"\b(Dolby.?)?Atmos\b", regex.IGNORECASE), uniq_concat(value("Atmos")), {"remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("audio", regex.compile(r"\b(TrueHD|\.True\.)\b", regex.IGNORECASE), uniq_concat(value("TrueHD")), {"remove": True, "skipIfAlreadyFound": False, "skipFromTitle": True})
	// parser.add_handler("audio", regex.compile(r"\bTRUE\b"), uniq_concat(value("TrueHD")), {"remove": True, "skipIfAlreadyFound": False, "skipFromTitle": True})
	{
		Field:       "audio",
		Pattern:     regexp.MustCompile(`(?i)\bDDP5[ ._]1\b`),
		Transform:   to_value_set("DDP"),
		Remove:      true,
		SkipIfFirst: true,
	},
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\b(?:.+HR)?(?:DTS.?HD.?Ma(?:ster)?|DTS.?X)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:.+HR)`)),
		Transform:     to_value_set("DTS Lossless"),
		Remove:        true,
		KeepMatching:  true,
	},
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\bDTS(?:(?:.?HD.?Ma(?:ster)?|.X))?.?(?:HD.?HR|HD)?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)DTS(?:.?HD.?Ma(?:ster)?|.X)`)),
		Transform:     to_value_set("DTS Lossy"),
		Remove:        true,
		KeepMatching:  true,
	},
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(?:Dolby.?)?Atmos\b`),
		Transform:    to_value_set("Atmos"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\b(?:TrueHD|\.True\.)\b`),
		Transform:     to_value_set("TrueHD"),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`\bTRUE\b`),
		Transform:     to_value_set("TrueHD"),
		Remove:        true,
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// x parser.addHandler("audio", /7\.1[. ]?Atmos\b/i, value("7.1 Atmos"), { remove: true });
	// x parser.addHandler("audio", /\b(?:mp3|Atmos|DTS(?:-HD)?|TrueHD)\b/i, lowercase);
	// parser.addHandler("audio", /\bFLAC(?:\+?2\.0)?(?:x[2-4])?\b/i, value("flac"), { remove: true });
	// parser.addHandler("audio", /\bEAC-?3(?:[. -]?[256]\.[01])?/i, value("eac3"), { remove: true, skipIfAlreadyFound: false });
	// {
	// 	Field:     "audio",
	// 	Pattern:   regexp.MustCompile(`(?i)7\.1[. ]?Atmos\b`),
	// 	Transform: to_value_set("7.1 Atmos"),
	// 	Remove:    true,
	// },
	// {
	// 	Field:     "audio",
	// 	Pattern:   regexp.MustCompile(`(?i)\b(?:mp3|Atmos|DTS(?:-HD)?|TrueHD)\b`),
	// 	Transform: to_lowercase(),
	// },
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bFLAC(?:\+?2\.0)?(?:x[2-4])?\b`),
		Transform:    to_value_set("FLAC"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bEAC-?3(?:[. -]?[256]\.[01])?`),
		Transform:    to_value_set("EAC3"),
		Remove:       true,
		KeepMatching: true,
	},
	// x parser.addHandler("audio", /\bAC-?3(?:[.-]5\.1|x2\.?0?)?\b/i, value("ac3"), { remove: true, skipIfAlreadyFound: false });
	// {
	// 	Field:        "audio",
	// 	Pattern:      regexp.MustCompile(`(?i)\bAC-?3(?:[.-]5\.1|x2\.?0?)?\b`),
	// 	Transform:    to_value_set("AC3"),
	// 	Remove:       true,
	// 	KeepMatching: true,
	// },
	// parser.add_handler("audio", regex.compile(r"\bAC-?3(x2)?(?:[ .-](5\.1)?[x+]2\.?0?x?3?)?\b", regex.IGNORECASE), uniq_concat(value("AC3")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bAC-?3(?:x2)?(?:[ .-](?:5\.1)?[x+]2\.?0?x?3?)?\b`),
		Transform:    to_value_set("AC3"),
		Remove:       true,
		KeepMatching: true,
	},
	// x parser.addHandler("audio", /\b5\.1(?:x[2-4]+)?\+2\.0(?:x[2-4])?\b/i, value("2.0"), { remove: true, skipIfAlreadyFound: false });
	// x parser.addHandler("audio", /\b2\.0(?:x[2-4]|\+5\.1(?:x[2-4])?)\b/i, value("2.0"), { remove: true, skipIfAlreadyFound: false });
	// parser.addHandler("audio", /\b5\.1ch\b/i, value("ac3"), { remove: true, skipIfAlreadyFound: false });
	// {
	// 	Field:        "audio",
	// 	Pattern:      regexp.MustCompile(`(?i)\b5\.1(?:x[2-4]+)?\+2\.0(?:x[2-4])?\b`),
	// 	Transform:    to_value_set("2.0"),
	// 	Remove:       true,
	// 	KeepMatching: true,
	// },
	// {
	// 	Field:        "audio",
	// 	Pattern:      regexp.MustCompile(`(?i)\b2\.0(?:x[2-4]|\+5\.1(?:x[2-4])?)\b`),
	// 	Transform:    to_value_set("2.0"),
	// 	Remove:       true,
	// 	KeepMatching: true,
	// },
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b5\.1ch\b`),
		Transform:    to_value_set("AC3"),
		Remove:       true,
		KeepMatching: true,
	},
	// parser.add_handler("audio", regex.compile(r"\b(DD2?[\+p]2?(.?5.1)?|DD Plus|Dolby Digital Plus)\b", regex.IGNORECASE), uniq_concat(value("Dolby Digital Plus")), {"remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("audio", regex.compile(r"\b(DD|Dolby.?Digital.?)2?(5.?1)?(?!.?(Plus|P|\+))\b", regex.IGNORECASE), uniq_concat(value("Dolby Digital")), {"remove": True, "skipIfAlreadyFound": False})
	// parser.add_handler("audio", regex.compile(r"\bDolbyD\b", regex.IGNORECASE), uniq_concat(value("Dolby Digital")), {"remove": True, "skipIfFirst": True})
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(DD2?[\+p]2?(?:.?5.1)?|DD Plus|Dolby Digital Plus)\b`),
		Transform:    to_value_set("DDP"),
		Remove:       true,
		KeepMatching: true,
	},
	{
		Field:         "audio",
		Pattern:       regexp.MustCompile(`(?i)\b(DD|Dolby.?Digital.?)2?(?:5.?1)?(?:.?(?:Plus|P|\+))?\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:Plus|P|\+)`)),
		Transform:     to_value_set("DD"),
		Remove:        true,
		KeepMatching:  true,
	},
	{
		Field:       "audio",
		Pattern:     regexp.MustCompile(`(?i)\bDolbyD\b`),
		Transform:   to_value_set("DD"),
		Remove:      true,
		SkipIfFirst: true,
	},
	// x parser.addHandler("audio", /\bDD5[. ]?1\b/i, value("dd5.1"), { remove: true });
	// parser.addHandler("audio", /\bQ?AAC(?:[. ]?2[. ]0|x2)?\b/, value("aac"), { remove: true });
	// {
	// 	Field:     "audio",
	// 	Pattern:   regexp.MustCompile(`(?i)\bDD5[. ]?1\b`),
	// 	Transform: to_value_set("dd5.1"),
	// 	Remove:    true,
	// },
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`\bQ?AAC(?:[. ]?2[. ]0|x2)?\b`),
		Transform:    to_value_set("AAC"),
		Remove:       true,
		KeepMatching: true,
	},
	// parser.add_handler("audio", regex.compile(r"\b(H[DQ])?.?(Clean.?Aud(io)?)\b", regex.IGNORECASE), uniq_concat(value("HQ Clean Audio")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\b(?:H[DQ])?.?(?:Clean.?Aud(?:io)?)\b`),
		Transform:    to_value_set("HQ"),
		Remove:       true,
		KeepMatching: true,
	},
	// parser.addHandler("audioChannels", /\[[257][.-][01]]/, lowercase, { remove: true });
	{
		Field:   "channels",
		Pattern: regexp.MustCompile(`\[([257][.-][01])]`),
		Transform: to_value_set_with_transform(func(v string) any {
			return strings.ToLower(v)
		}),
		Remove:       true,
		KeepMatching: true,
	},

	// parser.addHandler("group", /- ?(?!\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?(?=\.\w{2,4}$|$)/i, { remove: true });
	{
		Field:         "group",
		Pattern:       regexp.MustCompile(`(?i)- ?([^\-. []+[^\-. [)\]\d][^\-. [)\]]*)(?:\[[\w.-]+])?(?:\.\w{2,4}$|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)- ?(?:\d+$|S\d+|\d+x|ep?\d+|[^[]+]$)`)),
		// Remove:        true,
	},

	// parser.addHandler("container", /\.?[[(]?\b(MKV|AVI|MP4|WMV|MPG|MPEG)\b[\])]?/i, lowercase);
	{
		Field:     "container",
		Pattern:   regexp.MustCompile(`(?i)\.?[[(]?\b(MKV|AVI|MP4|WMV|MPG|MPEG)\b[\])]?`),
		Transform: to_lowercase(),
	},

	// parser.addHandler("volumes", /\bvol(?:s|umes?)?[. -]*(?:\d{1,2}[., +/\\&-]+)+\d{1,2}\b/i, range, { remove: true });
	// parser.addHandler("volumes", ({ title, result, matched }) => {
	//     const startIndex = matched.year && matched.year.matchIndex || 0;
	//     const match = title.slice(startIndex).match(/\bvol(?:ume)?[. -]*(\d{1,2})/i);
	//
	//     if (match) {
	//         matched.volumes = { match: match[0], matchIndex: match.index };
	//         result.volumes = array(integer)(match[1]);
	//         return { rawMatch: match[0], matchIndex: match.index, remove: true };
	//     }
	//     return null;
	// });
	{
		Field:     "volumes",
		Pattern:   regexp.MustCompile(`(?i)\bvol(?:s|umes?)?[. -]*(?:\d{1,2}[., +/\\&-]+)+\d{1,2}\b`),
		Transform: to_int_range(),
		Remove:    true,
	},
	{
		Field: "volumes",
		Process: func() hProcessor {
			re := regexp.MustCompile(`(?i)\bvol(?:ume)?[. -]*(\d{1,2})`)
			return func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
				startIndex := 0
				if yr, ok := result["year"]; ok {
					startIndex = min(yr.mIndex, len(title))
				}
				mIdxs := re.FindStringSubmatchIndex(title[startIndex:])
				if len(mIdxs) == 0 {
					return m
				}
				mStr := title[startIndex+mIdxs[2] : startIndex+mIdxs[3]]
				if num, err := strconv.Atoi(mStr); err == nil {
					m.mIndex = mIdxs[0]
					m.mValue = title[startIndex+mIdxs[0] : startIndex+mIdxs[1]]
					m.value = []int{num}
					m.remove = true
				}
				return m
			}
		}(),
	},

	// parser.add_handler("country", regex.compile(r"\b(US|UK|AU|NZ)\b"), value("$1"))
	{
		Field:   "country",
		Pattern: regexp.MustCompile(`\b(US|UK|AU|NZ)\b`),
	},

	// parser.add_handler("languages", regex.compile(r"\b(temporadas?|completa)\b", regex.IGNORECASE), uniq_concat(value("es")), {"skipIfAlreadyFound": False})
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(temporadas?|completa)\b`),
		Transform:    to_value_set("spanish"),
		KeepMatching: true,
	},

	// parser.addHandler("complete", /(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bbox[ .-]?set\b/i, boolean);
	// parser.addHandler("complete", /(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bmini[ .-]?series\b/i, boolean);
	// parser.addHandler("complete", /(?:\bthe\W)?(?:\bcomplete|full|all)\b.*\b(?:series|seasons|collection|episodes|set|pack|movies)\b/i, boolean);
	// parser.addHandler("complete", /\b(?:series|seasons|movies?)\b.*\b(?:complete|collection)\b/i, boolean);
	// parser.addHandler("complete", /(?:\bthe\W)?\bultimate\b[ .]\bcollection\b/i, boolean, { skipIfAlreadyFound: false });
	// parser.addHandler("complete", /\bcollection\b.*\b(?:set|pack|movies)\b/i, boolean);
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bbox[ .-]?set\b`),
		Transform: to_boolean(),
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete|collection|dvd)?\b[ .]?\bmini[ .-]?series\b`),
		Transform: to_boolean(),
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:\bthe\W)?(?:\bcomplete|full|all)\b.*\b(?:series|seasons|collection|episodes|set|pack|movies)\b`),
		Transform: to_boolean(),
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\b(?:series|seasons|movies?)\b.*\b(?:complete|collection)\b`),
		Transform: to_boolean(),
	},
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(?i)(?:\bthe\W)?\bultimate\b[ .]\bcollection\b`),
		Transform:    to_boolean(),
		KeepMatching: true,
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcollection\b.*\b(?:set|pack|movies)\b`),
		Transform: to_boolean(),
	},
	// parser.add_handler("complete", regex.compile(r"\bcollection(?:(\s\[|\s\())", regex.IGNORECASE), boolean, {"remove": True})
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcollection(?:(\s\[|\s\())`),
		Transform: to_boolean(),
		Remove:    true,
	},
	// x parser.addHandler("complete", /\b(collection|completa)\b/i, boolean, { skipFromTitle: true });
	// {
	// 	Field:         "complete",
	// 	Pattern:       regexp.MustCompile(`(?i)\b(collection|completa)\b`),
	// 	Transform:     to_boolean(),
	// 	Remove:        true,
	// 	SkipFromTitle: true,
	// },
	// parser.addHandler("complete", /\bkolekcja\b(?:\Wfilm(?:y|ów|ow)?)?/i, boolean, { remove: true });
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bkolekcja\b(?:\Wfilm(?:y|ów|ow)?)?`),
		Transform: to_boolean(),
		Remove:    true,
	},
	// x parser.addHandler("complete", /duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology|saga/i, boolean, { skipIfAlreadyFound: false });
	// {
	// 	Field:        "complete",
	// 	Pattern:      regexp.MustCompile(`(?i)duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology|saga`),
	// 	Transform:    to_boolean(),
	// 	KeepMatching: true,
	// },
	// parser.add_handler("complete", regex.compile(r"duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology", regex.IGNORECASE), boolean, {"skipIfAlreadyFound": False})
	// parser.add_handler("complete", regex.compile(r"\bcompleta\b", regex.IGNORECASE), boolean, {"remove": True})
	// parser.add_handler("complete", regex.compile(r"\bsaga\b", regex.IGNORECASE), boolean, {"skipFromTitle": True, "skipIfAlreadyFound": True})
	// parser.add_handler("complete", regex.compile(r"(The.)?\[?\bComplete\b\]?", regex.IGNORECASE), boolean, {"remove": True})
	{
		Field:        "complete",
		Pattern:      regexp.MustCompile(`(?i)duology|trilogy|quadr[oi]logy|tetralogy|pentalogy|hexalogy|heptalogy|anthology`),
		Transform:    to_boolean(),
		KeepMatching: true,
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)\bcompleta\b`),
		Transform: to_boolean(),
		Remove:    true,
	},
	{
		Field:         "complete",
		Pattern:       regexp.MustCompile(`(?i)\bsaga\b`),
		Transform:     to_boolean(),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:     "complete",
		Pattern:   regexp.MustCompile(`(?i)(?:The.)?\[?\bComplete\b\]?`),
		Transform: to_boolean(),
		Remove:    true,
	},

	// parser.addHandler("seasons", /(?:complete\W|seasons?\W|\W|^)((?:s\d{1,2}[., +/\\&-]+)+s\d{1,2}\b)/i, range, { remove: true });
	// parser.addHandler("seasons", /(?:complete\W|seasons?\W|\W|^)[([]?(s\d{2,}-\d{2,}\b)[)\]]?/i, range, { remove: true });
	// parser.addHandler("seasons", /(?:complete\W|seasons?\W|\W|^)[([]?(s[1-9]-[2-9]\b)[)\]]?/i, range, { remove: true });
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)((?:s\d{1,2}[., +/\\&-]+)+s\d{1,2}\b)`),
		Transform: to_int_range(),
		Remove:    true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)[([]?(s\d{2,}-\d{2,}\b)[)\]]?`),
		Transform: to_int_range(),
		Remove:    true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:complete\W|seasons?\W|\W|^)[([]?(s[1-9]-[2-9]\b)[)\]]?`),
		Transform: to_int_range(),
		Remove:    true,
	},
	// parser.add_handler("seasons", regex.compile(r"\d+ª(?:.+)?(?:a.?)?\d+ª(?:(?:.+)?(?:temporadas?))", regex.IGNORECASE), range_func, {"remove": True})
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\d+ª(?:.+)?(?:a.?)?\d+ª(?:(?:.+)?(?:temporadas?))`),
		Transform: to_int_range(),
		Remove:    true,
	},
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|sezon|temporadas?|stagioni)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[, /\\&]+)+\d{1,2}\b)[)\]]?/i, range, { remove: true });
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons|[Сс]езони?|sezon|temporadas?|stagioni)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[. -]+)+0?[1-9]\d?\b)[)\]]?/i, range, { remove: true });
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?season[. ]?[([]?((?:\d{1,2}[. -]+)+[1-9]\d?\b)[)\]]?(?!.*\.\w{2,4}$)/i, range, { remove: true });
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?\bseasons?\b[. -]?(\d{1,2}[. -]?(?:to|thru|and|\+|:)[. -]?\d{1,2})\b/i, range, { remove: true });
	// parser.addHandler("seasons", /(\d{1,2})(?:-?й)?[. _]?(?:[Сс]езон|sez(?:on)?)(?:\W?\D|$)/i, array(integer));
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?(?:saison|seizoen|sezon(?:SO?)?|stagione|season|series|temp(?:orada)?):?[. ]?(\d{1,2})/i, array(integer));
	// parser.addHandler("seasons", /[Сс]езон:?[. _]?№?(\d{1,2})(?!\d)/i, array(integer));
	// parser.addHandler("seasons", /(?:\D|^)(\d{1,2})Â?[°ºªa]?[. ]*temporada/i, array(integer), { remove: true });
	// parser.addHandler("seasons", /t(\d{1,3})(?:[ex]+|$)/i, array(integer), { remove: true });
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete)?(?:\W|^)so?([01]?[0-5]?[1-9])(?:[\Wex]|\d{2}\b)/i, array(integer), { skipIfAlreadyFound: false });
	// parser.addHandler("seasons", /(?:so?|t)(\d{1,2})[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?\d{1,4}(?:[abc]|v0?[1-4]|\D|$)/i, array(integer));
	// parser.addHandler("seasons", /(?:(?:\bthe\W)?\bcomplete\W)?(?:\W|^)(\d{1,2})[. ]?(?:st|nd|rd|th)[. ]*season/i, array(integer));
	// parser.addHandler("seasons", /(?:\D|^)(\d{1,2})[Xxх]\d{1,3}(?:\D|$)/, array(integer));
	// parser.addHandler("seasons", /\bSn([1-9])(?:\D|$)/, array(integer));
	// parser.addHandler("seasons", /[[(](\d{1,2})\.\d{1,3}[)\]]/, array(integer));
	// parser.addHandler("seasons", /-\s?(\d{1,2})\.\d{2,3}\s?-/, array(integer));
	// parser.addHandler("seasons", /(?:^|\/)(?!20-20)(\d{1,2})-\d{2}\b(?!-\d)/, array(integer));
	// parser.addHandler("seasons", /[^\w-](\d{1,2})-\d{2}(?=\.\w{2,4}$)/, array(integer));
	// parser.addHandler("seasons", /(?<!\bEp?(?:isode)? ?\d+\b.*)\b(\d{2})[ ._]\d{2}(?:.F)?\.\w{2,4}$/, array(integer));
	// parser.addHandler("seasons", /\bEp(?:isode)?\W+(\d{1,2})\.\d{1,3}\b/i, array(integer));
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|sezon|temporadas?|stagioni)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[, /\\&]+)+\d{1,2}\b)[)\]]?`),
		Transform: to_int_range(),
		// Remove:    true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons|[Сс]езони?|sezon|temporadas?|stagioni)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[. -]+)+0?[1-9]\d?\b)[)\]]?`),
		Transform: to_int_range(),
		Remove:    true,
	},
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?season[. ]?[([]?((?:\d{1,2}[. -]+)+[1-9]\d?\b)[)\]]?(?:.*\.\w{2,4}$)?`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:.*\.\w{2,4}$)`)),
		Transform:     to_int_range(),
		Remove:        true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?\bseasons?\b[. -]?(\d{1,2}[. -]?(?:to|thru|and|\+|:)[. -]?\d{1,2})\b`),
		Transform: to_int_range(),
		Remove:    true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(\d{1,2})(?:-?й)?[. _]?(?:[Сс]езон|sez(?:on)?)(?:\W?\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:saison|seizoen|sezon(?:SO?)?|stagione|season|series|temp(?:orada)?):?[. ]?(\d{1,2})`),
		Transform: to_int_array(),
	},
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?i)[Сс]езон:?[. _]?№?(\d{1,2})(?:\d)?`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\d{3}`)),
		Transform:     to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:\D|^)(\d{1,2})Â?[°ºªa]?[. ]*temporada`),
		Transform: to_int_array(),
		Remove:    true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)t(\d{1,3})(?:[ex]+|$)`),
		Transform: to_int_array(),
		Remove:    true,
	},
	{
		Field:        "seasons",
		Pattern:      regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete)?(?:\W|^)so?([01]?[0-5]?[1-9])(?:[\Wex]|\d{2}\b)`),
		Transform:    to_int_array(),
		KeepMatching: true,
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:so?|t)(\d{1,2})[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?\d{1,4}(?:[abc]|v0?[1-4]|\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:\W|^)(\d{1,2})[. ]?(?:st|nd|rd|th)[. ]*season`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?:\D|^)(\d{1,2})[Xxх]\d{1,3}(?:\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`\bSn([1-9])(?:\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`[[(](\d{1,2})\.\d{1,3}[)\]]`),
		Transform: to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`-\s?(\d{1,2})\.\d{2,3}\s?-`),
		Transform: to_int_array(),
	},
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?:^|\/)(?:20-20)?(\d{1,2})-\d{2}\b(?:-\d)?`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`^(?:20-20)|(\d{1,2})-\d{2}\b(?:-\d)`)),
		Transform:     to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`[^\w-](\d{1,2})-\d{2}(?:\.\w{2,4}$)`),
		Transform: to_int_array(),
	},
	{
		Field:         "seasons",
		Pattern:       regexp.MustCompile(`(?:\bEp?(?:isode)? ?\d+\b.*)?\b(\d{2})[ ._]\d{2}(?:.F)?\.\w{2,4}$`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?:\bEp?(?:isode)? ?\d+\b.*)`)),
		Transform:     to_int_array(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\bEp(?:isode)?\W+(\d{1,2})\.\d{1,3}\b`),
		Transform: to_int_array(),
	},
	// parser.add_handler("seasons", regex.compile(r"\bSeasons?\b.*\b(\d{1,2}-\d{1,2})\b", regex.IGNORECASE), range_func)
	// parser.add_handler("seasons", regex.compile(r"(?:\W|^)(\d{1,2})(?:e|ep)\d{1,3}(?:\W|$)", regex.IGNORECASE), array(integer))
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)\bSeasons?\b.*\b(\d{1,2}-\d{1,2})\b`),
		Transform: to_int_range(),
	},
	{
		Field:     "seasons",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(\d{1,2})(?:e|ep)\d{1,3}(?:\W|$)`),
		Transform: to_int_array(),
	},

	// parser.addHandler("episodes", /(?:[\W\d]|^)e[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|e){1,2}[ .]?\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /(?:[\W\d]|^)ep[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|ep){1,2}[ .]?\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /(?:[\W\d]|^)\d+[xх][ .]?[([]?(\d{1,3}(?:[ .]?[xх][ .]?\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /(?:[\W\d]|^)(?:episodes?|[Сс]ерии:?)[ .]?[([]?(\d{1,3}(?:[ .+]*[&+][ .]?\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /[([]?(?:\D|^)(\d{1,3}[ .]?ao[ .]?\d{1,3})[)\]]?(?:\W|$)/i, range);
	// parser.addHandler("episodes", /(?:[\W\d]|^)(?:e|eps?|episodes?|[Сс]ерии:?|\d+[xх])[ .]*[([]?(\d{1,3}(?:-\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /(?:so?|t)\d{1,2}[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\D|$)/i, array(integer));
	// parser.addHandler("episodes", /\b(?:so?|t)\d{2}(\d{2})\b/i, array(integer));
	// parser.addHandler("episodes", /(?:\W|^)(\d{1,3}(?:[ .]*~[ .]*\d{1,3})+)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?!-\d)(?:\W|$)/i, range);
	// parser.addHandler("episodes", /s\d{1,2}\s?\((\d{1,3}[ .]*-[ .]*\d{1,3})\)/i, range);
	// parser.addHandler("episodes", /(?:^|\/)(?!20-20)\d{1,2}-(\d{2})\b(?!-\d)/, array(integer));
	// parser.addHandler("episodes", /(?<!\d-)\b\d{1,2}-(\d{2})(?=\.\w{2,4}$)/, array(integer));
	// parser.addHandler("episodes", /(?<=^\[.+].+)[. ]+-[. ]+(\d{1,4})[. ]+(?=\W)/i, array(integer));
	// parser.addHandler("episodes", /(?<!(?:seasons?|[Сс]езони?)\W*)(?:[ .([-]|^)(\d{1,3}(?:[ .]?[,&+~][ .]?\d{1,3})+)(?:[ .)\]-]|$)/i, range);
	// parser.addHandler("episodes", /(?<!(?:seasons?|[Сс]езони?)\W*)(?!20-20)(?:[ .([-]|^)(\d{1,3}(?:-\d{1,3})+)(?:[ .)(\]]|-\D|$)/i, range);
	// parser.addHandler("episodes", /\bEp(?:isode)?\W+\d{1,2}\.(\d{1,3})\b/i, array(integer));
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)e[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|e){1,2}[ .]?\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)ep[ .]?[([]?(\d{1,3}(?:[ .-]*(?:[&+]|ep){1,2}[ .]?\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)\d+[xх][ .]?[([]?(\d{1,3}(?:[ .]?[xх][ .]?\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)(?:episodes?|[Сс]ерии:?)[ .]?[([]?(\d{1,3}(?:[ .+]*[&+][ .]?\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)[([]?(?:\D|^)(\d{1,3}[ .]?ao[ .]?\d{1,3})[)\]]?(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:[\W\d]|^)(?:e|eps?|episodes?|[Сс]ерии:?|\d+[xх])[ .]*[([]?(\d{1,3}(?:-\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:so?|t)\d{1,2}[. ]?[xх-]?[. ]?(?:e|x|х|ep|-|\.)[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b(?:so?|t)\d{2}(\d{2})\b`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(\d{1,3}(?:[ .]*~[ .]*\d{1,3})+)(?:\W|$)`),
		Transform: to_int_range(),
	},
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)?(?:\W|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)-\s(\d{1,3}[ .]*-[ .]*\d{1,3})(?:-\d*)`)),
		Transform:     to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)s\d{1,2}\s?\((\d{1,3}[ .]*-[ .]*\d{1,3})\)`),
		Transform: to_int_range(),
	},
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?:^|\/)(?:20-20)?\d{1,2}-(\d{2})\b(?:-\d)?`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`^(?:20-20)|\d{1,2}-(\d{2})\b(?:-\d)`)),
		Transform:     to_int_array(),
	},
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?:\d-)?\b\d{1,2}-(\d{2})(?:\.\w{2,4}$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?:\d-)\b\d{1,2}-(\d{2})`)),
		Transform:     to_int_array(),
	},
	{
		Field:      "episodes",
		Pattern:    regexp.MustCompile(`(?i)(?:^\[.+].+)([. ]+-[. ]+(\d{1,4})[. ]+)(?:\W)`),
		Transform:  to_int_array(),
		ValueGroup: 2,
		MatchGroup: 1,
	},
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:seasons?|[Сс]езони?)\W*)?(?:[ .([-]|^)(\d{1,3}(?:[ .]?[,&+~][ .]?\d{1,3})+)(?:[ .)\]-]|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:(?:seasons?|[Сс]езони?)\W*)`)),
		Transform:     to_int_range(),
	},
	{
		Field:         "episodes",
		Pattern:       regexp.MustCompile(`(?i)(?:(?:seasons?|[Сс]езони?)\W*)?(?:20-20)?(?:[ .([-]|^)(\d{1,3}(?:-\d{1,3})+)(?:[ .)(\]]|-\D|$)`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:(?:seasons?|[Сс]езони?)\W*|^)(?:20-20)`)),
		Transform:     to_int_range(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\bEp(?:isode)?\W+\d{1,2}\.(\d{1,3})\b`),
		Transform: to_int_array(),
	},
	// parser.add_handler("episodes", regex.compile(r"Ep.\d+.-.\d+", regex.IGNORECASE), range_func, {"remove": True})
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)Ep.\d+.-.\d+`),
		Transform: to_int_range(),
		Remove:    true,
	},
	// parser.addHandler("episodes", /(?:\b[ée]p?(?:isode)?|[Ээ]пизод|[Сс]ер(?:ии|ия|\.)?|caa?p(?:itulo)?|epis[oó]dio)[. ]?[-:#№]?[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\W|$)/i, array(integer));
	// parser.addHandler("episodes", /\b(\d{1,3})(?:-?я)?[ ._-]*(?:ser(?:i?[iyj]a|\b)|[Сс]ер(?:ии|ия|\.)?)/i, array(integer));
	// parser.addHandler("episodes", /(?:\D|^)\d{1,2}[. ]?[Xxх][. ]?(\d{1,3})(?:[abc]|v0?[1-4]|\D|$)/, array(integer));
	// parser.addHandler("episodes", /[[(]\d{1,2}\.(\d{1,3})[)\]]/, array(integer));
	// parser.addHandler("episodes", /\b[Ss](?:eason\W?)?\d{1,2}[ .](\d{1,2})\b/, array(integer));
	// parser.addHandler("episodes", /-\s?\d{1,2}\.(\d{2,3})\s?-/, array(integer));
	// parser.addHandler("episodes", /(?<=\D|^)(\d{1,3})[. ]?(?:of|из|iz)[. ]?\d{1,3}(?=\D|$)/i, array(integer));
	// parser.addHandler("episodes", /\b\d{2}[ ._-](\d{2})(?:.F)?\.\w{2,4}$/, array(integer));
	// parser.addHandler("episodes", /(?<!^)\[(\d{2,3})](?!(?:\.\w{2,4})?$)/, array(integer));
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\b[ée]p?(?:isode)?|[Ээ]пизод|[Сс]ер(?:ии|ия|\.)?|caa?p(?:itulo)?|epis[oó]dio)[. ]?[-:#№]?[. ]?(\d{1,4})(?:[abc]|v0?[1-4]|\W|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b(\d{1,3})(?:-?я)?[ ._-]*(?:ser(?:i?[iyj]a|\b)|[Сс]ер(?:ии|ия|\.)?)`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\D|^)\d{1,2}[. ]?[Xxх][. ]?(\d{1,3})(?:[abc]|v0?[1-4]|\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)[[(]\d{1,2}\.(\d{1,3})[)\]]`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b[Ss](?:eason\W?)?\d{1,2}[ .](\d{1,2})\b`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)-\s?\d{1,2}\.(\d{2,3})\s?-`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\D|^)(\d{1,3})[. ]?(?:of|из|iz)[. ]?\d{1,3}(?:\D|$)`),
		Transform: to_int_array(),
	},
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\b\d{2}[ ._-](\d{2})(?:.F)?\.\w{2,4}$`),
		Transform: to_int_array(),
	},
	{
		Field:   "episodes",
		Pattern: regexp.MustCompile(`(?i)(?:^)?\[(\d{2,3})](?:(?:\.\w{2,4})?$)?`),
		ValidateMatch: validate_and(
			validate_not_at_start(),
			validate_not_at_end(),
			validate_not_match(regexp.MustCompile(`(?i)(?:720|1080)|\[(\d{2,3})](?:(?:\.\w{2,4})$)`)),
		),
		Transform: to_int_array(),
	},
	// parser.add_handler("episodes", regex.compile(r"(?<![xh])\b264\b|\b265\b", regex.IGNORECASE), array(integer), {"remove": True})
	{
		Field:   "episodes",
		Pattern: regexp.MustCompile(`(?i)\b264\b|\b265\b`),
		ValidateMatch: func() hMatchValidator {
			re := regexp.MustCompile(`(?i)\b[xh]\b`)
			return func(input string, match []int) bool {
				return !re.MatchString(input[:match[0]])
			}
		}(),
		Transform: to_int_array(),
		Remove:    true,
	},
	// parser.add_handler("episodes", regex.compile(r"(?:\W|^)(?:\d+)?(?:e|ep)(\d{1,3})(?:\W|$)", regex.IGNORECASE), array(integer), {"remove": True})
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)(?:\W|^)(?:\d+)?(?:e|ep)(\d{1,3})(?:\W|$)`),
		Transform: to_int_array(),
		Remove:    true,
	},
	// parser.add_handler("episodes", regex.compile(r"\d+.-.\d+TV", regex.IGNORECASE), range_func, {"remove": True})
	{
		Field:     "episodes",
		Pattern:   regexp.MustCompile(`(?i)\d+.-.\d+TV`),
		Transform: to_int_range(),
		Remove:    true,
	},
	// // can be both absolute episode and season+episode in format 101
	// parser.addHandler("episodes", ({ title, result, matched }) => {
	//     if (!result.episodes) {
	//         const startIndexes = [matched.year, matched.seasons]
	//             .filter(component => component)
	//             .map(component => component.matchIndex)
	//             .filter(index => index > 0);
	//         const endIndexes = [matched.resolution, matched.source, matched.codec, matched.audio]
	//             .filter(component => component)
	//             .map(component => component.matchIndex)
	//             .filter(index => index > 0);
	//         const startIndex = startIndexes.length ? Math.min(...startIndexes) : 0;
	//         const endIndex = Math.min(...endIndexes, title.length);
	//         const beginningTitle = title.slice(0, endIndex);
	//         const middleTitle = title.slice(startIndex, endIndex);
	//
	//         // try to match the episode inside the title with a separator, if not found include the start of the title as well
	//         const matches = Array.from(beginningTitle.matchAll(/(?<!movie\W*|film\W*|^)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:\W|$)(?!movie|film|\d+)/gi)).pop() ||
	//             middleTitle.match(/^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?!\Wmovie|\Wfilm|-\d)(?:\W|$)/i);
	//
	//         if (matches) {
	//             result.episodes = [matches[matches.length - 1]]
	//                 .map(group => group.replace(/\D/g, ""))
	//                 .map(group => parseInt(group, 10));
	//             return { matchIndex: title.indexOf(matches[0]) };
	//         }
	//     }
	//     return null;
	// });
	// beginning_pattern = regex.compile(r"(?<!movie\W*|film\W*|^)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:\W|$)(?!movie|film|\d+)(?<!\[(?:480|720|1080)\])", regex.IGNORECASE)
	// middle_pattern = regex.compile(r"^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?:\W|$)(?!movie|film)(?!\[(480|720|1080)\])", regex.IGNORECASE)
	{
		Field: "episodes",
		Process: func() hProcessor {
			btRe := regexp.MustCompile(`(?i)(?:movie\W*|film\W*|^)?(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})(?:a|b|v\d|\.\d)?(?:\W|$)(?:movie|film|\d+)?`)
			btReNegBefore := regexp.MustCompile(`(?i)(?:movie\W*|film\W*)(?:[ .]+-[ .]+|[([][ .]*)(\d{1,4})`)
			btReNegAfter := regexp.MustCompile(`(?i)(?:movie|film)|(\d{1,4})(?:a|b|v\d|\.\d)(?:\W)(?:\d+)`)
			mtRe := regexp.MustCompile(`(?i)^(?:[([-][ .]?)?(\d{1,4})(?:a|b|v\d)?(?:\Wmovie|\Wfilm|-\d)?(?:\W|$)`)
			mtReNegAfter := regexp.MustCompile(`(?i)(\d{1,4})(?:a|b|v\d)?(?:\Wmovie|\Wfilm|-\d)`)
			commonResolutionNeg := regexp.MustCompile(`\[(?:480|720|1080)\]`)
			commonFPSNeg := regexp.MustCompile(`(?i)\d+(?:fps|帧率?)`)
			return func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
				if m.value != nil {
					return m
				}
				startIndex := 0
				for _, component := range []string{"year", "seasons"} {
					if cm, ok := result[component]; ok {
						if cm.mIndex > 0 && (startIndex == 0 || cm.mIndex < startIndex) {
							startIndex = cm.mIndex
						}
					}
				}
				endIndex := len(title)
				for _, component := range []string{"resolution", "quality", "codec", "audio"} {
					if cm, ok := result[component]; ok {
						if cm.mIndex > 0 && cm.mIndex < endIndex {
							endIndex = cm.mIndex
						}
					}
				}
				beginningTitle := title[:endIndex]
				startIndex = min(startIndex, len(title))
				middleTitle := title[startIndex:max(endIndex, startIndex)]
				mIdxs := btRe.FindStringSubmatchIndex(beginningTitle)
				mStr := ""
				if len(mIdxs) != 0 {
					mStr = beginningTitle[mIdxs[0]:mIdxs[1]]
					if mIdxs[0] == 0 || btReNegBefore.MatchString(mStr) || btReNegAfter.MatchString(mStr) || commonResolutionNeg.MatchString(mStr) || commonFPSNeg.MatchString(mStr) {
						mIdxs, mStr = nil, ""
					} else if len(mIdxs) > 2 {
						mStr = beginningTitle[mIdxs[2]:mIdxs[3]]
					}
				}
				if mStr == "" {
					mIdxs = mtRe.FindStringSubmatchIndex(middleTitle)
					if len(mIdxs) != 0 {
						if mtReNegAfter.MatchString(middleTitle[mIdxs[2]:]) || commonResolutionNeg.MatchString(mStr) {
							mIdxs, mStr = nil, ""
						} else {
							mStr = middleTitle[mIdxs[2]:mIdxs[3]]
						}
					}
				}
				if mStr != "" {
					mStr = non_digit_regex.ReplaceAllString(mStr, "")
					if ep, err := strconv.Atoi(mStr); err == nil {
						m.mIndex = strings.Index(title, mStr)
						m.mValue = mStr
						m.value = []int{ep}
					}
				}
				return m
			}
		}(),
	},

	// parser.addHandler("languages", /\bmulti(?:ple)?[ .-]*(?:su?$|sub\w*|dub\w*)\b|msub/i, uniqConcat(value("multi subs")), { skipIfAlreadyFound: false, remove: true });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bmulti(?:ple)?[ .-]*(?:su?$|sub\w*|dub\w*)\b|msub`),
		Transform:    to_value_set("multi subs"),
		KeepMatching: true,
		Remove:       true,
	},
	// parser.addHandler("languages", /\bmulti(?:ple)?[ .-]*(?:lang(?:uages?)?|audio|VF2)?\b/i, uniqConcat(value("multi audio")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\btri(?:ple)?[ .-]*(?:audio|dub\w*)\b/i, uniqConcat(value("multi audio")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bmulti(?:ple)?[ .-]*(?:lang(?:uages?)?|audio|VF2)?\b`),
		Transform:    to_value_set("multi audio"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\btri(?:ple)?[ .-]*(?:audio|dub\w*)\b`),
		Transform:    to_value_set("multi audio"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\bdual[ .-]*(?:au?$|[aá]udio|line)\b/i, uniqConcat(value("dual audio")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bdual\b(?![ .-]*sub)/i, uniqConcat(value("dual audio")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bdual[ .-]*(?:au?$|[aá]udio|line)\b`),
		Transform:    to_value_set("dual audio"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bdual\b(?:[ .-]*sub)?`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:[ .-]*sub)`)),
		Transform:     to_value_set("dual audio"),
		KeepMatching:  true,
	},
	// parser.addHandler("languages", /\bengl?(?:sub[A-Z]*)?\b/i, uniqConcat(value("english")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\beng?sub[A-Z]*\b/i, uniqConcat(value("english")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bing(?:l[eéê]s)?\b/i, uniqConcat(value("english")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bengl?(?:sub[A-Z]*)?\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\beng?sub[A-Z]*\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bing(?:l[eéê]s)?\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
	},
	// parser.add_handler("languages", regex.compile(r"\besub\b", regex.IGNORECASE), uniq_concat(value("en")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\besub\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
		Remove:       true,
	},
	// parser.addHandler("languages", /\benglish\W+(?:subs?|sdh|hi)\b/i, uniqConcat(value("english")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bEN\b/i, uniqConcat(value("english")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\benglish?\b/i, uniqConcat(value("english")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\benglish\W+(?:subs?|sdh|hi)\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bEN\b`),
		Transform:     to_value_set("english"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\benglish?\b`),
		Transform:    to_value_set("english"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:JP|JAP|JPN)\b/i, uniqConcat(value("japanese")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(japanese|japon[eê]s)\b/i, uniqConcat(value("japanese")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:JP|JAP|JPN)\b`),
		Transform:    to_value_set("japanese"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)(japanese|japon[eê]s)\b`),
		Transform:    to_value_set("japanese"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:KOR|kor[ .-]?sub)\b/i, uniqConcat(value("korean")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(korean|coreano)\b/i, uniqConcat(value("korean")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:KOR|kor[ .-]?sub)\b`),
		Transform:    to_value_set("korean"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)(korean|coreano)\b`),
		Transform:    to_value_set("korean"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:traditional\W*chinese|chinese\W*traditional)(?:\Wchi)?\b/i, uniqConcat(value("taiwanese")), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("languages", /\bzh-hant\b/i, uniqConcat(value("taiwanese")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:traditional\W*chinese|chinese\W*traditional)(?:\Wchi)?\b`),
		Transform:    to_value_set("taiwanese"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bzh-hant\b`),
		Transform:    to_value_set("taiwanese"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\b(?:mand[ae]rin|ch[sn])\b/i, uniqConcat(value("chinese")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:mand[ae]rin|ch[sn])\b`),
		Transform:    to_value_set("chinese"),
		KeepMatching: true,
	},
	// parser.add_handler("languages", regex.compile(r"(?<!shang-?)\bCH(?:I|T)\b", regex.IGNORECASE), uniq_concat(value("zh")), {"skipFromTitle": True, "skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)(?:shang-?)?\bCH(?:I|T)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)shang-?`)),
		Transform:     to_value_set("chinese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// x parser.addHandler("languages", /\bCH[IT]\b/, uniqConcat(value("chinese")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// {
	// 	Field:         "languages",
	// 	Pattern:       regexp.MustCompile(`(?i)\bCH[IT]\b`),
	// 	Transform:     to_value_set("chinese"),
	// 	KeepMatching:  true,
	// 	SkipFromTitle: true,
	// },
	// parser.addHandler("languages", /\b(chinese|chin[eê]s|chi)\b/i, uniqConcat(value("chinese")), { skipIfFirst: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bzh-hans\b/i, uniqConcat(value("chinese")), { skipIfAlreadyFound: false });
	{
		Field: "languages",
		// Pattern:      regexp.MustCompile(`(?i)(chinese|chin[eê]s|chi)\b`),
		Pattern:      regexp.MustCompile(`(?i)(chinese|chin[eê]s)\b`),
		Transform:    to_value_set("chinese"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bzh-hans\b`),
		Transform:    to_value_set("chinese"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\bFR(?:ench|a|e|anc[eê]s)?\b/i, uniqConcat(value("french")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(Truefrench|VF[FI])\b/i, uniqConcat(value("french")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(VOST(?:FR?|A)?|SUBFRENCH)\b/i, uniqConcat(value("french")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bFR(?:ench|a|e|anc[eê]s)?\b`),
		Transform:    to_value_set("french"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(Truefrench|VF[FI])\b`),
		Transform:    to_value_set("french"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(VOST(?:FR?|A)?|SUBFRENCH)\b`),
		Transform:    to_value_set("french"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\bspanish\W?latin|american\W*(?:spa|esp?)/i, uniqConcat(value("latino")), { skipFromTitle: true, skipIfAlreadyFound: false, remove: true });
	// x parser.addHandler("languages", /\b(?:audio.)?lat(?:i|ino)?\b/i, uniqConcat(value("latino")), { skipIfAlreadyFound: false });
	// parser.add_handler("languages", regex.compile(r"\b(?:audio.)?lat(?:in?|ino)?\b", regex.IGNORECASE), uniq_concat(value("la")), {"skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bspanish\W?latin|american\W*(?:spa|esp?)`),
		Transform:     to_value_set("latino"),
		KeepMatching:  true,
		Remove:        true,
		SkipFromTitle: true,
	},
	// {
	// 	Field:        "languages",
	// 	Pattern:      regexp.MustCompile(`(?i)\b(?:audio.)?lat(?:i|ino)?\b`),
	// 	Transform:    to_value_set("latino"),
	// 	KeepMatching: true,
	// },
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:audio.)?lat(?:in?|ino)?\b`),
		Transform:    to_value_set("latino"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\b(?:audio.)?(?:ESP|spa|(en[ .]+)?espa[nñ]ola?|castellano)\b/i, uniqConcat(value("spanish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bes(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b/i, uniqConcat(value("spanish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})es\b/i, uniqConcat(value("spanish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)es(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\b/i, uniqConcat(value("spanish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bes(?=\.(?:ass|ssa|srt|sub|idx)$)/i, uniqConcat(value("spanish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bspanish\W+subs?\b/i, uniqConcat(value("spanish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(spanish|espanhol)\b/i, uniqConcat(value("spanish")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:audio.)?(?:ESP|spa|(?:en[ .]+)?espa[nñ]ola?|castellano)\b`),
		Transform:    to_value_set("spanish"),
		KeepMatching: true,
	},
	// {
	// 	Field:       "languages",
	// 	Pattern:     regexp.MustCompile(`(?i)\bes(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b`),
	// 	Transform:   to_multiple_value("spanish"),
	// 	KeepParsing: true,
	//  SkipFromTitle: true,
	// },
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})es\b`),
		Transform:     to_value_set("spanish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:[ .,/-]*[A-Z]{2}[ .,/-]+)es(?:[ .,/-]+[A-Z]{2}[ .,/-]+)\b`),
		Transform:     to_value_set("spanish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bes(?:\.(?:ass|ssa|srt|sub|idx)$)`),
		Transform:     to_value_set("spanish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bspanish\W+subs?\b`),
		Transform:    to_value_set("spanish"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(spanish|espanhol)\b`),
		Transform:    to_value_set("spanish"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:p[rt]|en|port)[. (\\/-]*BR\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("languages", /\bbr(?:a|azil|azilian)\W+(?:pt|por)\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("languages", /\b(?:leg(?:endado|endas?)?|dub(?:lado)?|portugu[eèê]se?)[. -]*BR\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bleg(?:endado|endas?)\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bportugu[eèê]s[ea]?\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bPT[. -]*(?:PT|ENG?|sub(?:s|titles?))\b/i, uniqConcat(value("portuguese")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bpt(?=\.(?:ass|ssa|srt|sub|idx)$)/i, uniqConcat(value("portuguese")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bpor\b/i, uniqConcat(value("portuguese")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:p[rt]|en|port)[. (\\/-]*BR\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bbr(?:a|azil|azilian)\W+(?:pt|por)\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:leg(?:endado|endas?)?|dub(?:lado)?|portugu[eèê]se?)[. -]*BR\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bleg(?:endado|endas?)\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bportugu[eèê]s[ea]?\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bPT[. -]*(?:PT|ENG?|sub(?:s|titles?))\b`),
		Transform:    to_value_set("portuguese"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bpt(?:\.(?:ass|ssa|srt|sub|idx)$)`),
		Transform:     to_value_set("portuguese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bpor\b`),
		Transform:     to_value_set("portuguese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\bITA\b/i, uniqConcat(value("italian")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(?<!w{3}\.\w+\.)IT(?=[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,})\b/, uniqConcat(value("italian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bit(?=\.(?:ass|ssa|srt|sub|idx)$)/i, uniqConcat(value("italian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bitaliano?\b/i, uniqConcat(value("italian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bITA\b`),
		Transform:    to_value_set("italian"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`\b(?:w{3}\.\w+\.)?IT(?:[ .,/-]+(?:[a-zA-Z]{2}[ .,/-]+){2,})\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?:w{3}\.\w+\.)IT`)),
		Transform:     to_value_set("italian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bit(?:\.(?:ass|ssa|srt|sub|idx)$)`),
		Transform:     to_value_set("italian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bitaliano?\b`),
		Transform:    to_value_set("italian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bgreek[ .-]*(?:audio|lang(?:uage)?|subs?(?:titles?)?)?\b/i, uniqConcat(value("greek")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bgreek[ .-]*(?:audio|lang(?:uage)?|subs?(?:titles?)?)?\b`),
		Transform:    to_value_set("greek"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:GER|DEU)\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bde(?=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(?<=[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})de\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)de(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\b/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bde(?=\.(?:ass|ssa|srt|sub|idx)$)/i, uniqConcat(value("german")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(german|alem[aã]o)\b/i, uniqConcat(value("german")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:GER|DEU)\b`),
		Transform:     to_value_set("german"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bde(?:[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})\b`),
		Transform:     to_value_set("german"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:[ .,/-]+(?:[A-Z]{2}[ .,/-]+){2,})de\b`),
		Transform:     to_value_set("german"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// {
	// 	Field:       "languages",
	// 	Pattern:     regexp.MustCompile(`(?i)\b(?<=[ .,/-]+[A-Z]{2}[ .,/-]+)de(?=[ .,/-]+[A-Z]{2}[ .,/-]+)\b`),
	// 	Transform:   to_multiple_value("german"),
	// 	KeepParsing: true,
	//  SkipFromTitle: true,
	// },
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bde(?:\.(?:ass|ssa|srt|sub|idx)$)`),
		Transform:     to_value_set("german"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(german|alem[aã]o)\b`),
		Transform:    to_value_set("german"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bRUS?\b/i, uniqConcat(value("russian")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(russian|russo)\b/i, uniqConcat(value("russian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bRUS?\b`),
		Transform:    to_value_set("russian"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)(russian|russo)\b`),
		Transform:    to_value_set("russian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bUKR\b/i, uniqConcat(value("ukrainian")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bukrainian\b/i, uniqConcat(value("ukrainian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bUKR\b`),
		Transform:    to_value_set("ukrainian"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bukrainian\b`),
		Transform:    to_value_set("ukrainian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bhin(?:di)?\b/i, uniqConcat(value("hindi")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bhin(?:di)?\b`),
		Transform:    to_value_set("hindi"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)tel(?!\W*aviv)|telugu)\b/i, uniqConcat(value("telugu")), { skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?tel(?:\W*aviv)?|telugu)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:(?:w{3}\.\w+\.)tel)|(?:tel(?:\W*aviv))`)),
		Transform:     to_value_set("telugu"),
		KeepMatching:  true,
	},
	// parser.addHandler("languages", /\bt[aâ]m(?:il)?\b/i, uniqConcat(value("tamil")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bt[aâ]m(?:il)?\b`),
		Transform:    to_value_set("tamil"),
		KeepMatching: true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)MAL(?:ay)?|malayalam)\b", regex.IGNORECASE), uniq_concat(value("ml")), {"remove": True, "skipIfFirst": True, "skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?MAL(?:ay)?|malayalam)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)MAL)\b`)),
		Transform:     to_value_set("malayalam"),
		KeepMatching:  true,
		Remove:        true,
		SkipIfFirst:   true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)KAN(?:nada)?|kannada)\b", regex.IGNORECASE), uniq_concat(value("kn")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?KAN(?:nada)?|kannada)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)KAN)\b`)),
		Transform:     to_value_set("kannada"),
		KeepMatching:  true,
		Remove:        true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)MAR(?:a(?:thi)?)?|marathi)\b", regex.IGNORECASE), uniq_concat(value("mr")), {"skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?MAR(?:a(?:thi)?)?|marathi)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)MAR)\b`)),
		Transform:     to_value_set("marathi"),
		KeepMatching:  true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)GUJ(?:arati)?|gujarati)\b", regex.IGNORECASE), uniq_concat(value("gu")), {"skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?GUJ(?:arati)?|gujarati)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)GUJ)\b`)),
		Transform:     to_value_set("gujarati"),
		KeepMatching:  true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)PUN(?:jabi)?|punjabi)\b", regex.IGNORECASE), uniq_concat(value("pa")), {"skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?PUN(?:jabi)?|punjabi)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)PUN)\b`)),
		Transform:     to_value_set("punjabi"),
		KeepMatching:  true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(?:(?<!w{3}\.\w+\.)BEN(?!.\bThe|and|of\b)(?:gali)?|bengali)\b", regex.IGNORECASE), uniq_concat(value("bn")), {"skipIfFirst": True, "skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?BEN(?:.\bThe|and|of\b)?(?:gali)?|bengali)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)BEN)|(?:BEN)(?:.\bThe|and|of\b)\b`)),
		Transform:     to_value_set("bengali"),
		KeepMatching:  true,
		SkipIfFirst:   true,
	},
	// parser.addHandler("languages", /\b(?<!YTS\.)LT\b/, uniqConcat(value("lithuanian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\blithuanian\b/i, uniqConcat(value("lithuanian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:YTS\.)?LT\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:YTS\.)`)),
		Transform:     to_value_set("lithuanian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\blithuanian\b`),
		Transform:    to_value_set("lithuanian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\blatvian\b/i, uniqConcat(value("latvian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\blatvian\b`),
		Transform:    to_value_set("latvian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bestonian\b/i, uniqConcat(value("estonian")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bestonian\b`),
		Transform:    to_value_set("estonian"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\b(?:PLDUB|Dubbing.PL|Lektor.PL|Film.Polski)\b/i, uniqConcat(value("polish")), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("languages", /\b(?:Napisy.PL|PLSUB(?:BED)?)\b/i, uniqConcat(value("polish")), { skipIfAlreadyFound: false, remove: true });
	// parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)PL|pol)\b/i, uniqConcat(value("polish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(polish|polon[eê]s|polaco)\b/i, uniqConcat(value("polish")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:PLDUB|Dubbing.PL|Lektor.PL|Film.Polski)\b`),
		Transform:    to_value_set("polish"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:Napisy.PL|PLSUB(?:BED)?)\b`),
		Transform:    to_value_set("polish"),
		KeepMatching: true,
		Remove:       true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?PL|pol)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:w{3}\.\w+\.)`)),
		Transform:     to_value_set("polish"),
		KeepMatching:  true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(polish|polon[eê]s|polaco)\b`),
		Transform:    to_value_set("polish"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bCZ[EH]?\b/i, uniqConcat(value("czech")), { skipIfFirst: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bczech\b/i, uniqConcat(value("czech")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bCZ[EH]?\b`),
		Transform:    to_value_set("czech"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bczech\b`),
		Transform:    to_value_set("czech"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	// parser.addHandler("languages", /\bslo(?:vak|vakian|subs|[\]_)]?\.\w{2,4}$)\b/i, uniqConcat(value("slovakian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bslo(?:vak|vakian|subs|[\]_)]?\.\w{2,4}$)\b`),
		Transform:     to_value_set("slovakian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\bHU\b/, uniqConcat(value("hungarian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bHUN(?:garian)?\b/i, uniqConcat(value("hungarian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bHU\b`),
		Transform:     to_value_set("hungarian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bHUN(?:garian)?\b`),
		Transform:     to_value_set("hungarian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\bROM(?:anian)?\b/i, uniqConcat(value("romanian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bRO(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)/i, uniqConcat(value("romanian")), { skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bROM(?:anian)?\b`),
		Transform:     to_value_set("romanian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bRO(?:[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)`),
		Transform:    to_value_set("romanian"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\bbul(?:garian)?\b/i, uniqConcat(value("bulgarian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bbul(?:garian)?\b`),
		Transform:     to_value_set("bulgarian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:srp|serbian)\b/i, uniqConcat(value("serbian")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:srp|serbian)\b`),
		Transform:    to_value_set("serbian"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\b(?:HRV|croatian)\b/i, uniqConcat(value("croatian")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bHR(?=[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub)\b/i, uniqConcat(value("croatian")), { skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:HRV|croatian)\b`),
		Transform:    to_value_set("croatian"),
		KeepMatching: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bHR(?:[ .,/-]*(?:[A-Z]{2}[ .,/-]+)*sub\w*)\b`),
		Transform:    to_value_set("croatian"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\bslovenian\b/i, uniqConcat(value("slovenian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bslovenian\b`),
		Transform:     to_value_set("slovenian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)NL|dut|holand[eê]s)\b/i, uniqConcat(value("dutch")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bdutch\b/i, uniqConcat(value("dutch")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bflemish\b/i, uniqConcat(value("dutch")), { skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?NL|dut|holand[eê]s)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:w{3}\.\w+\.)NL`)),
		Transform:     to_value_set("dutch"),
		KeepMatching:  true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bdutch\b`),
		Transform:     to_value_set("dutch"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\bflemish\b`),
		Transform:    to_value_set("dutch"),
		KeepMatching: true,
	},
	// parser.addHandler("languages", /\b(?:DK|danska|dansub|nordic)\b/i, uniqConcat(value("danish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(danish|dinamarqu[eê]s)\b/i, uniqConcat(value("danish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bdan\b(?=.*\.(?:srt|vtt|ssa|ass|sub|idx)$)/i, uniqConcat(value("danish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:DK|danska|dansub|nordic)\b`),
		Transform:    to_value_set("danish"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(danish|dinamarqu[eê]s)\b`),
		Transform:     to_value_set("danish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bdan\b(?:.*\.(?:srt|vtt|ssa|ass|sub|idx)$)`),
		Transform:     to_value_set("danish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)FI|finsk|finsub|nordic)\b/i, uniqConcat(value("finnish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bfinnish\b/i, uniqConcat(value("finnish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?FI|finsk|finsub|nordic)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:w{3}\.\w+\.)FI`)),
		Transform:     to_value_set("finnish"),
		KeepMatching:  true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bfinnish\b`),
		Transform:     to_value_set("finnish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:(?<!w{3}\.\w+\.)SE|swe|swesubs?|sv(?:ensk)?|nordic)\b/i, uniqConcat(value("swedish")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(swedish|sueco)\b/i, uniqConcat(value("swedish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:(?:w{3}\.\w+\.)?SE|swe|swesubs?|sv(?:ensk)?|nordic)\b`),
		ValidateMatch: validate_not_match(regexp.MustCompile(`(?i)(?:w{3}\.\w+\.)SE`)),
		Transform:     to_value_set("swedish"),
		KeepMatching:  true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(swedish|sueco)\b`),
		Transform:     to_value_set("swedish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:NOR|norsk|norsub|nordic)\b/i, uniqConcat(value("norwegian")), { skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?=[\]_)]?\.\w{2,4}$))\b/i, uniqConcat(value("norwegian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:NOR|norsk|norsub|nordic)\b`),
		Transform:    to_value_set("norwegian"),
		KeepMatching: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(norwegian|noruegu[eê]s|bokm[aå]l|nob|nor(?:[\]_)]?\.\w{2,4}$))\b`),
		Transform:     to_value_set("norwegian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:arabic|[aá]rabe|ara)\b/i, uniqConcat(value("arabic")), { skipIfFirst: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\barab.*(?:audio|lang(?:uage)?|sub(?:s|titles?)?)\b/i, uniqConcat(value("arabic")), { skipFromTitle: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\bar(?=\.(?:ass|ssa|srt|sub|idx)$)/i, uniqConcat(value("arabic")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:arabic|[aá]rabe|ara)\b`),
		Transform:    to_value_set("arabic"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\barab.*(?:audio|lang(?:uage)?|sub(?:s|titles?)?)\b`),
		Transform:     to_value_set("arabic"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bar(?:\.(?:ass|ssa|srt|sub|idx)$)`),
		Transform:     to_value_set("arabic"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:turkish|tur(?:co)?)\b/i, uniqConcat(value("turkish")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(?:turkish|tur(?:co)?)\b`),
		Transform:     to_value_set("turkish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.add_handler("languages", regex.compile(r"\b(TİVİBU|tivibu|bitturk(.net)?|turktorrent)\b", regex.IGNORECASE), uniq_concat(value("tr")), {"skipFromTitle": True, "skipIfAlreadyFound": False})
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(TİVİBU|tivibu|bitturk(?:\.net)?|turktorrent)\b`),
		Transform:     to_value_set("turkish"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\bvietnamese\b|\bvie(?=[\]_)]?\.\w{2,4}$)/i, uniqConcat(value("vietnamese")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bvietnamese\b|\bvie(?:[\]_)]?\.\w{2,4}$)`),
		Transform:     to_value_set("vietnamese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\bind(?:onesian)?\b/i, uniqConcat(value("indonesian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bind(?:onesian)?\b`),
		Transform:     to_value_set("indonesian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(thai|tailand[eê]s)\b/i, uniqConcat(value("thai")), { skipIfFirst: true, skipIfAlreadyFound: false });
	// parser.addHandler("languages", /\b(THA|tha)\b/, uniqConcat(value("thai")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(thai|tailand[eê]s)\b`),
		Transform:    to_value_set("thai"),
		KeepMatching: true,
		SkipIfFirst:  true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`\b(THA|tha)\b`),
		Transform:     to_value_set("thai"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(?:malay|may(?=[\]_)]?\.\w{2,4}$)|(?<=subs?\([a-z,]+)may)\b/i, uniqConcat(value("malay")), { skipIfFirst: true, skipIfAlreadyFound: false });
	{
		Field:        "languages",
		Pattern:      regexp.MustCompile(`(?i)\b(?:malay|may(?:[\]_)]?\.\w{2,4}$)|(?:subs?\([a-z,]+)may)\b`),
		Transform:    to_value_set("malay"),
		KeepMatching: true,
		// SkipIfFirst: true,
	},
	// parser.addHandler("languages", /\bheb(?:rew|raico)?\b/i, uniqConcat(value("hebrew")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\bheb(?:rew|raico)?\b`),
		Transform:     to_value_set("hebrew"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", /\b(persian|persa)\b/i, uniqConcat(value("persian")), { skipFromTitle: true, skipIfAlreadyFound: false });
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)\b(persian|persa)\b`),
		Transform:     to_value_set("persian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.add_handler("languages", regex.compile(r"[\u3040-\u30ff]+", regex.IGNORECASE), uniq_concat(value("ja")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # japanese
	// parser.add_handler("languages", regex.compile(r"[\u3400-\u4dbf]+", regex.IGNORECASE), uniq_concat(value("zh")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # chinese
	// parser.add_handler("languages", regex.compile(r"[\u4e00-\u9fff]+", regex.IGNORECASE), uniq_concat(value("zh")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # chinese
	// parser.add_handler("languages", regex.compile(r"[\uf900-\ufaff]+", regex.IGNORECASE), uniq_concat(value("zh")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # chinese
	// parser.add_handler("languages", regex.compile(r"[\uff66-\uff9f]+", regex.IGNORECASE), uniq_concat(value("ja")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # japanese
	// parser.add_handler("languages", regex.compile(r"[\u0400-\u04ff]+", regex.IGNORECASE), uniq_concat(value("ru")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # russian
	// parser.add_handler("languages", regex.compile(r"[\u0600-\u06ff]+", regex.IGNORECASE), uniq_concat(value("ar")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # arabic
	// parser.add_handler("languages", regex.compile(r"[\u0750-\u077f]+", regex.IGNORECASE), uniq_concat(value("ar")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # arabic
	// parser.add_handler("languages", regex.compile(r"[\u0c80-\u0cff]+", regex.IGNORECASE), uniq_concat(value("kn")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # kannada
	// parser.add_handler("languages", regex.compile(r"[\u0d00-\u0d7f]+", regex.IGNORECASE), uniq_concat(value("ml")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # malayalam
	// parser.add_handler("languages", regex.compile(r"[\u0e00-\u0e7f]+", regex.IGNORECASE), uniq_concat(value("th")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # thai
	// parser.add_handler("languages", regex.compile(r"[\u0900-\u097f]+", regex.IGNORECASE), uniq_concat(value("hi")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # hindi
	// parser.add_handler("languages", regex.compile(r"[\u0980-\u09ff]+", regex.IGNORECASE), uniq_concat(value("bn")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # bengali
	// parser.add_handler("languages", regex.compile(r"[\u0a00-\u0a7f]+", regex.IGNORECASE), uniq_concat(value("gu")), {"skipFromTitle": True, "skipIfAlreadyFound": False})  # gujarati
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{3040}-\x{30ff}]+`),
		Transform:     to_value_set("japanese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{3400}-\x{4dbf}]+`),
		Transform:     to_value_set("chinese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{4e00}-\x{9fff}]+`),
		Transform:     to_value_set("chinese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{f900}-\x{faff}]+`),
		Transform:     to_value_set("chinese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{ff66}-\x{ff9f}]+`),
		Transform:     to_value_set("japanese"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0400}-\x{04ff}]+`),
		Transform:     to_value_set("russian"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0600}-\x{06ff}]+`),
		Transform:     to_value_set("arabic"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0750}-\x{077f}]+`),
		Transform:     to_value_set("arabic"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0c80}-\x{0cff}]+`),
		Transform:     to_value_set("kannada"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0d00}-\x{0d7f}]+`),
		Transform:     to_value_set("malayalam"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0e00}-\x{0e7f}]+`),
		Transform:     to_value_set("thai"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0900}-\x{097f}]+`),
		Transform:     to_value_set("hindi"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0980}-\x{09ff}]+`),
		Transform:     to_value_set("bengali"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	{
		Field:         "languages",
		Pattern:       regexp.MustCompile(`(?i)[\x{0a00}-\x{0a7f}]+`),
		Transform:     to_value_set("gujarati"),
		KeepMatching:  true,
		SkipFromTitle: true,
	},
	// parser.addHandler("languages", ({ title, result, matched }) => {
	//     if (!result.languages || ["portuguese", "spanish"].every(l => !result.languages.includes(l))) {
	//         if ((matched.episodes && matched.episodes.rawMatch.match(/capitulo|ao/i)) ||
	//             title.match(/dublado/i)) {
	//             result.languages = (result.languages || []).concat("portuguese");
	//         }
	//     }
	//     return { matchIndex: 0 };
	// });
	{
		Field: "languages",
		Process: func() hProcessor {
			ere := regexp.MustCompile(`(?i)capitulo|ao`)
			tre := regexp.MustCompile(`(?i)dublado`)
			return func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
				m.mIndex = 0
				m.mValue = ""
				vs, ok := m.value.(*value_set[any])
				if ok && (vs.exists("portuguese") || vs.exists("spanish")) {
					return m
				}
				em, found := result["episodes"]
				if (found && em.mValue != "" && ere.MatchString(em.mValue)) || tre.MatchString(title) {
					if !ok {
						vs = &value_set[any]{existMap: map[any]struct{}{}, values: []any{}}
					}
					m.value = vs.append("portuguese")
				}
				return m
			}
		}(),
	},

	// parser.add_handler("subbed", regex.compile(r"\bmulti(?:ple)?[ .-]*(?:su?$|sub\w*|dub\w*)\b|msub", regex.IGNORECASE), boolean, {"remove": True})
	// parser.add_handler("subbed", regex.compile(r"\b(?:Official.*?|Dual-?)?sub(s|bed)?\b", regex.IGNORECASE), boolean, {"remove": True})
	{
		Field:     "subbed",
		Pattern:   regexp.MustCompile(`(?i)\b(?:Official.*?|Dual-?)?sub(?:s|bed)?\b`),
		Transform: to_boolean(),
		Remove:    true,
	},
	{
		Field: "subbed",
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			lm, found := result["languages"]
			if !found {
				return m
			}
			if s, ok := lm.value.(*value_set[any]); ok && s.exists("multi subs") {
				m.value = true
			}
			return m
		},
	},

	// parser.addHandler("dubbed", /\b(?:DUBBED|dublado|dubbing|DUBS?)\b/i, boolean);
	// parser.addHandler("dubbed", ({ result }) => {
	//     if (result.languages && ["multi audio", "dual audio"].some(l => result.languages.includes(l))) {
	//         result.dubbed = true;
	//     }
	//     return { matchIndex: 0 };
	// });
	{
		Field:     "dubbed",
		Pattern:   regexp.MustCompile(`(?i)\b(?:DUBBED|dublado|dubbing|DUBS?)\b`),
		Transform: to_boolean(),
	},
	{
		Field: "dubbed",
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			lm, found := result["languages"]
			if !found {
				return m
			}
			if s, ok := lm.value.(*value_set[any]); ok && (s.exists("multi audio") || s.exists("dual audio")) {
				m.value = true
			}
			return m
		},
	},

	// parser.add_handler("size", regex.compile(r"\b(\d+(\.\d+)?\s?(MB|GB|TB))\b", regex.IGNORECASE), none, {"remove": True})
	{
		Field:   "size",
		Pattern: regexp.MustCompile(`(?i)\b(\d+(\.\d+)?\s?(MB|GB|TB))\b`),
		Remove:  true,
	},

	// parser.add_handler("site", regex.compile(r"\[([^\]]+\.[^\]]+)\](?=\.\w{2,4}$|\s)", regex.IGNORECASE), value("$1"), {"remove": True})
	// parser.add_handler("site", regex.compile(r"\bwww.\w*.\w+\b", regex.IGNORECASE), value("$1"), {"remove": True})
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)(\[([^\]]+\.[^\]]+)\])(?:\.\w{2,4}$|\s)`),
		Transform:     to_trimmed(),
		Remove:        true,
		SkipFromTitle: true,

		MatchGroup: 1,
		ValueGroup: 2,
	},
	{
		Field:         "site",
		Pattern:       regexp.MustCompile(`(?i)\bwww.\w*.\w+\b`),
		Remove:        true,
		SkipFromTitle: true,
	},

	// parser.add_handler("network", regex.compile(r"\bATVP?\b", regex.IGNORECASE), value("Apple TV"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bAMZN\b", regex.IGNORECASE), value("Amazon"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bNF|Netflix\b", regex.IGNORECASE), value("Netflix"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bNICK(elodeon)?\b", regex.IGNORECASE), value("Nickelodeon"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bDSNY?P?\b", regex.IGNORECASE), value("Disney"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bH(MAX|BO)\b", regex.IGNORECASE), value("HBO"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bHULU\b", regex.IGNORECASE), value("Hulu"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bCBS\b", regex.IGNORECASE), value("CBS"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bNBC\b", regex.IGNORECASE), value("NBC"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bAMC\b", regex.IGNORECASE), value("AMC"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bPBS\b", regex.IGNORECASE), value("PBS"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\b(Crunchyroll|[. -]CR[. -])\b", regex.IGNORECASE), value("Crunchyroll"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bVICE\b"), value("VICE"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bSony\b", regex.IGNORECASE), value("Sony"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bHallmark\b", regex.IGNORECASE), value("Hallmark"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bAdult.?Swim\b", regex.IGNORECASE), value("Adult Swim"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bAnimal.?Planet|ANPL\b", regex.IGNORECASE), value("Animal Planet"), {"remove": True})
	// parser.add_handler("network", regex.compile(r"\bCartoon.?Network(.TOONAMI.BROADCAST)?\b", regex.IGNORECASE), value("Cartoon Network"), {"remove": True})
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bATVP?\b`),
		Transform: to_value("Apple TV"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAMZN\b`),
		Transform: to_value("Amazon"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNF|Netflix\b`),
		Transform: to_value("Netflix"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNICK(elodeon)?\b`),
		Transform: to_value("Nickelodeon"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bDSNY?P?\b`),
		Transform: to_value("Disney"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bH(MAX|BO)\b`),
		Transform: to_value("HBO"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bHULU\b`),
		Transform: to_value("Hulu"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bCBS\b`),
		Transform: to_value("CBS"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bNBC\b`),
		Transform: to_value("NBC"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAMC\b`),
		Transform: to_value("AMC"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bPBS\b`),
		Transform: to_value("PBS"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\b(Crunchyroll|[. -]CR[. -])\b`),
		Transform: to_value("Crunchyroll"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`\bVICE\b`),
		Transform: to_value("VICE"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bSony\b`),
		Transform: to_value("Sony"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bHallmark\b`),
		Transform: to_value("Hallmark"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAdult.?Swim\b`),
		Transform: to_value("Adult Swim"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bAnimal.?Planet|ANPL\b`),
		Transform: to_value("Animal Planet"),
		Remove:    true,
	},
	{
		Field:     "network",
		Pattern:   regexp.MustCompile(`(?i)\bCartoon.?Network(?:.TOONAMI.BROADCAST)?\b`),
		Transform: to_value("Cartoon Network"),
		Remove:    true,
	},

	// parser.add_handler("group", regex.compile(r"\b(INFLATE|DEFLATE)\b"), value("$1"), {"remove": True})
	// parser.add_handler("group", regex.compile(r"\b(?:Erai-raws|Erai-raws\.com)\b", regex.IGNORECASE), value("Erai-raws"), {"remove": True})
	{
		Field:   "group",
		Pattern: regexp.MustCompile(`\b(INFLATE|DEFLATE)\b`),
		Remove:  true,
	},
	{
		Field:     "group",
		Pattern:   regexp.MustCompile(`(?i)\b(?:Erai-raws|Erai-raws\.com)\b`),
		Transform: to_value("Erai-raws"),
		Remove:    true,
	},
	// parser.addHandler("group", /^\[([^[\]]+)]/);
	// parser.addHandler("group", /\(([\w-]+)\)(?:$|\.\w{2,4}$)/);
	// parser.addHandler("group", ({ result, matched }) => {
	//     if (matched.group && matched.group.rawMatch.match(/^\[.+]$/)) {
	//         const endIndex = matched.group && matched.group.matchIndex + matched.group.rawMatch.length || 0;
	//
	//         // remove anime group match if some other parameter is contained in it, since it's a false positive.
	//         if (Object.keys(matched)
	//             .some(key => matched[key].matchIndex && matched[key].matchIndex < endIndex)) {
	//             delete result.group;
	//         }
	//     }
	//     return { matchIndex: 0 };
	// });
	{
		Field:   "group",
		Pattern: regexp.MustCompile(`^\[([^[\]]+)]`),
	},
	{
		Field:   "group",
		Pattern: regexp.MustCompile(`\(([\w-]+)\)(?:$|\.\w{2,4}$)`),
	},
	{
		Field: "group",
		Process: func(title string, m *parseMeta, result map[string]*parseMeta) *parseMeta {
			re := regexp.MustCompile(`^\[.+]$`)
			if m.mValue != "" && re.MatchString(m.mValue) {
				endIndex := m.mIndex + len(m.mValue)
				// remove anime group match if some other parameter is contained in it, since it's a false positive.
				for key := range result {
					if km, found := result[key]; found && km.mIndex > 0 && km.mIndex < endIndex {
						m.value = ""
						return m
					}
				}
			}
			m.mIndex = 0
			m.mValue = ""
			return m
		},
	},

	// parser.addHandler("extension", /\.(3g2|3gp|avi|flv|mkv|mk3d|mov|mp2|mp4|m4v|mpe|mpeg|mpg|mpv|webm|wmv|ogm|divx|ts|m2ts|iso|vob|sub|idx|ttxt|txt|smi|srt|ssa|ass|vtt|nfo|html")$/i, lowercase);
	{
		Field:     "extension",
		Pattern:   regexp.MustCompile(`(?i)\.(3g2|3gp|avi|flv|mkv|mk3d|mov|mp2|mp4|m4v|mpe|mpeg|mpg|mpv|webm|wmv|ogm|divx|ts|m2ts|iso|vob|sub|idx|ttxt|txt|smi|srt|ssa|ass|vtt|nfo|html)$`),
		Transform: to_lowercase(),
	},
	// parser.add_handler("audio", regex.compile(r"\bMP3\b", regex.IGNORECASE), uniq_concat(value("MP3")), {"remove": True, "skipIfAlreadyFound": False})
	{
		Field:        "audio",
		Pattern:      regexp.MustCompile(`(?i)\bMP3\b`),
		Transform:    to_value_set("MP3"),
		Remove:       true,
		KeepMatching: true,
	},
}
