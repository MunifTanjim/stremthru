package util

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/agnivade/levenshtein"
	fuzzy "github.com/paul-mannino/go-fuzzywuzzy"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func RepeatJoin(s string, count int, sep string) string {
	if count == 0 {
		return ""
	}
	return strings.Repeat(s+sep, count-1) + s
}

func MustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

var numericRegex = regexp.MustCompile(`^[0-9]+$`)

func IsNumericString(s string) bool {
	return numericRegex.MatchString(s)
}

var quoteRegex = regexp.MustCompile(`['"]+`)
var separatorRegex = regexp.MustCompile(`[ ._-]+`)

type StringNormalizer struct {
	t transform.Transformer
}

func (sn *StringNormalizer) Normalize(input string) string {
	input = strings.ToLower(input)
	input = quoteRegex.ReplaceAllLiteralString(input, "")
	if result, _, err := transform.String(sn.t, input); err == nil {
		input = result
	}
	input = fuzzy.Cleanse(input, false)
	input = separatorRegex.ReplaceAllLiteralString(input, " ")
	return strings.TrimSpace(input)
}

func NewStringNormalizer() *StringNormalizer {
	sn := StringNormalizer{}
	sn.t = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	return &sn
}

func FuzzyTokenSetRatio(query, input string, normalizer *StringNormalizer) int {
	if normalizer == nil {
		normalizer = NewStringNormalizer()
	}
	return fuzzy.TokenSetRatio(normalizer.Normalize(query), normalizer.Normalize(input))
}

func MinFuzzyTokenSetRatio(ratio int, query, input string, normalizer *StringNormalizer) bool {
	if normalizer == nil {
		normalizer = NewStringNormalizer()
	}
	return fuzzy.TokenSetRatio(normalizer.Normalize(query), normalizer.Normalize(input)) >= ratio
}

func LevenshteinDistance(a, b string, normalizer *StringNormalizer) int {
	if normalizer == nil {
		normalizer = NewStringNormalizer()
	}
	return levenshtein.ComputeDistance(normalizer.Normalize(a), normalizer.Normalize(b))
}

func MaxLevenshteinDistance(distance int, a, b string, normalizer *StringNormalizer) bool {
	return LevenshteinDistance(a, b, normalizer) < distance
}
