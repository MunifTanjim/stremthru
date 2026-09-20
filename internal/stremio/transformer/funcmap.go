package stremio_transformer

import (
	"html/template"
	"strconv"
	"strings"
)

var lang_code_to_emoji = map[string]string{
	"dub":         "🗣️",
	"daud":        "🔉",
	"dual audio":  "🔉",
	"maud":        "🔊",
	"multi audio": "🔊",
	"msub":        "🔤",
	"multi subs":  "🔤",

	"en":     "🇬🇧",
	"ja":     "🇯🇵",
	"ru":     "🇷🇺",
	"it":     "🇮🇹",
	"pt":     "🇵🇹",
	"pt-pt":  "🇵🇹",
	"pt-br":  "🇧🇷",
	"es":     "🇪🇸",
	"es-419": "🇲🇽",
	"es-mx":  "🇲🇽",
	"ko":     "🇰🇷",
	"zh":     "🇨🇳",
	"zh-tw":  "🇹🇼",
	"fr":     "🇫🇷",
	"de":     "🇩🇪",
	"nl":     "🇳🇱",
	"hi":     "🇮🇳",
	"te":     "🇮🇳",
	"ta":     "🇮🇳",
	"ml":     "🇮🇳",
	"kn":     "🇮🇳",
	"mr":     "🇮🇳",
	"gu":     "🇮🇳",
	"pa":     "🇮🇳",
	"bn":     "🇧🇩",
	"ur":     "🇵🇰",
	"ne":     "🇳🇵",
	"si":     "🇱🇰",
	"ps":     "🇦🇫",
	"pl":     "🇵🇱",
	"lt":     "🇱🇹",
	"lv":     "🇱🇻",
	"et":     "🇪🇪",
	"cs":     "🇨🇿",
	"sk":     "🇸🇰",
	"sl":     "🇸🇮",
	"hu":     "🇭🇺",
	"ro":     "🇷🇴",
	"bg":     "🇧🇬",
	"sr":     "🇷🇸",
	"hr":     "🇭🇷",
	"sq":     "🇦🇱",
	"mk":     "🇲🇰",
	"bs":     "🇧🇦",
	"uk":     "🇺🇦",
	"el":     "🇬🇷",
	"da":     "🇩🇰",
	"fi":     "🇫🇮",
	"sv":     "🇸🇪",
	"no":     "🇳🇴",
	"tr":     "🇹🇷",
	"ar":     "🇸🇦",
	"fa":     "🇮🇷",
	"he":     "🇮🇱",
	"hy":     "🇦🇲",
	"ka":     "🇬🇪",
	"az":     "🇦🇿",
	"kk":     "🇰🇿",
	"uz":     "🇺🇿",
	"tg":     "🇹🇯",
	"vi":     "🇻🇳",
	"id":     "🇮🇩",
	"ms":     "🇲🇾",
	"th":     "🇹🇭",
	"km":     "🇰🇭",
	"lo":     "🇱🇦",
	"mn":     "🇲🇳",
	"tl":     "🇵🇭",
	"so":     "🇸🇴",
	"am":     "🇪🇹",
	"af":     "🇿🇦",
	"zu":     "🇿🇦",
	"xh":     "🇿🇦",
	"is":     "🇮🇸",
}

func langToEmoji(lang string) string {
	if emoji, ok := lang_code_to_emoji[lang]; ok {
		return emoji
	}
	if _, ok := lang_code_to_iso[lang]; ok {
		return "🏳️"
	}
	return lang
}

var lang_code_to_text = map[string]string{
	"dub":         "Dubbed",
	"daud":        "Dual Audio",
	"dual audio":  "Dual Audio",
	"maud":        "Multi Audio",
	"multi audio": "Multi Audio",
	"msub":        "Multi Subs",
	"multi subs":  "Multi Subs",

	"en":     "English",
	"ja":     "Japanese",
	"ru":     "Russian",
	"it":     "Italian",
	"pt":     "Portuguese",
	"pt-pt":  "Portuguese",
	"pt-br":  "Portuguese (Brazil)",
	"es":     "Spanish",
	"es-419": "Latino",
	"es-mx":  "Spanish (Mexico)",
	"ko":     "Korean",
	"zh":     "Chinese",
	"zh-tw":  "Taiwanese",
	"fr":     "French",
	"de":     "German",
	"nl":     "Dutch",
	"hi":     "Hindi",
	"te":     "Telugu",
	"ta":     "Tamil",
	"ml":     "Malayalam",
	"kn":     "Kannada",
	"mr":     "Marathi",
	"gu":     "Gujarati",
	"pa":     "Punjabi",
	"bn":     "Bengali",
	"ur":     "Urdu",
	"ne":     "Nepali",
	"si":     "Sinhala",
	"ps":     "Pashto",
	"pl":     "Polish",
	"lt":     "Lithuanian",
	"lv":     "Latvian",
	"et":     "Estonian",
	"cs":     "Czech",
	"sk":     "Slovakian",
	"sl":     "Slovenian",
	"hu":     "Hungarian",
	"ro":     "Romanian",
	"bg":     "Bulgarian",
	"sr":     "Serbian",
	"hr":     "Croatian",
	"sq":     "Albanian",
	"mk":     "Macedonian",
	"bs":     "Bosnian",
	"uk":     "Ukrainian",
	"el":     "Greek",
	"da":     "Danish",
	"fi":     "Finnish",
	"sv":     "Swedish",
	"no":     "Norwegian",
	"tr":     "Turkish",
	"ar":     "Arabic",
	"fa":     "Persian",
	"he":     "Hebrew",
	"hy":     "Armenian",
	"ka":     "Georgian",
	"az":     "Azerbaijani",
	"kk":     "Kazakh",
	"uz":     "Uzbek",
	"tg":     "Tajik",
	"vi":     "Vietnamese",
	"id":     "Indonesian",
	"ms":     "Malay",
	"th":     "Thai",
	"km":     "Khmer",
	"lo":     "Lao",
	"mn":     "Mongolian",
	"tl":     "Tagalog",
	"so":     "Somali",
	"am":     "Amharic",
	"af":     "Afrikaans",
	"zu":     "Zulu",
	"xh":     "Xhosa",
	"is":     "Icelandic",
	"cy":     "Welsh",
	"eu":     "Basque",
	"ca":     "Catalan",
	"ku":     "Kurdish",
	"yi":     "Yiddish",
}

var lang_code_to_iso = map[string]string{
	"dub":         "Dub",
	"daud":        "DAud",
	"dual audio":  "DAud",
	"maud":        "MAud",
	"multi audio": "MAud",
	"msub":        "MSubs",
	"multi subs":  "MSubs",

	"en":     "ENG",
	"ja":     "JPN",
	"ru":     "RUS",
	"it":     "ITA",
	"pt":     "POR",
	"pt-pt":  "POR",
	"pt-br":  "POR(BR)",
	"es":     "SPA",
	"es-419": "SPA(LA)",
	"es-mx":  "SPA(MX)",
	"ko":     "KOR",
	"zh":     "ZHO",
	"zh-tw":  "ZHO(TW)",
	"fr":     "FRA",
	"de":     "DEU",
	"nl":     "NLD",
	"hi":     "HIN",
	"te":     "TEL",
	"ta":     "TAM",
	"ml":     "MAL",
	"kn":     "KAN",
	"mr":     "MAR",
	"gu":     "GUJ",
	"pa":     "PAN",
	"bn":     "BEN",
	"ur":     "URD",
	"ne":     "NEP",
	"si":     "SIN",
	"ps":     "PUS",
	"pl":     "POL",
	"lt":     "LIT",
	"lv":     "LAV",
	"et":     "EST",
	"cs":     "CES",
	"sk":     "SLK",
	"sl":     "SLV",
	"hu":     "HUN",
	"ro":     "RON",
	"bg":     "BUL",
	"sr":     "SRP",
	"hr":     "HRV",
	"sq":     "SQI",
	"mk":     "MKD",
	"bs":     "BOS",
	"uk":     "UKR",
	"el":     "ELL",
	"da":     "DAN",
	"fi":     "FIN",
	"sv":     "SWE",
	"no":     "NOR",
	"tr":     "TUR",
	"ar":     "ARA",
	"fa":     "FAS",
	"he":     "HEB",
	"hy":     "HYE",
	"ka":     "KAT",
	"az":     "AZE",
	"kk":     "KAZ",
	"uz":     "UZB",
	"tg":     "TGK",
	"vi":     "VIE",
	"id":     "IND",
	"ms":     "MSA",
	"th":     "THA",
	"km":     "KHM",
	"lo":     "LAO",
	"mn":     "MON",
	"tl":     "TGL",
	"so":     "SOM",
	"am":     "AMH",
	"af":     "AFR",
	"zu":     "ZUL",
	"xh":     "XHO",
	"is":     "ISL",
	"cy":     "CYM",
	"eu":     "EUS",
	"ca":     "CAT",
	"ku":     "KUR",
	"yi":     "YID",
}

func langToText(lang string) string {
	if text, ok := lang_code_to_text[lang]; ok {
		return text
	}
	return lang
}

func langToISO(lang string) string {
	if iso, ok := lang_code_to_iso[lang]; ok {
		return iso
	}
	return lang
}

var funcMap = template.FuncMap{
	"str_join":   strings.Join,
	"int_to_str": strconv.Itoa,
	"lang_join": func(languages []string, sep string, format string) string {
		var fn func(string) string
		switch format {
		case "emoji":
			fn = langToEmoji
		case "text":
			fn = langToText
		case "iso":
			fn = langToISO
		default:
			return strings.Join(languages, sep)
		}
		langs := make([]string, len(languages))
		for i := range languages {
			langs[i] = fn(languages[i])
		}
		return strings.Join(langs, sep)
	},
}
