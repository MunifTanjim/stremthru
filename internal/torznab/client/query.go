package torznab_client

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/util"
)

type SearchParam = string

const (
	SearchParamT        SearchParam = "t"
	SearchParamAPIKey   SearchParam = "apikey"
	SearchParamCat      SearchParam = "cat"
	SearchParamAttrs    SearchParam = "attrs"
	SearchParamExtended SearchParam = "extended"
	SearchParamOffset   SearchParam = "offset"
	SearchParamLimit    SearchParam = "limit"
	SearchParamQ        SearchParam = "q"
	SearchParamEp       SearchParam = "ep"
	SearchParamSeason   SearchParam = "season"
	SearchParamYear     SearchParam = "year"
	SearchParamIMDBId   SearchParam = "imdbid"
	SearchParamTVDBId   SearchParam = "tvdbid"
	SearchParamTVMazeId SearchParam = "tvmazeid"
	SearchParamTraktId  SearchParam = "traktid"
)

type Query struct {
	caps   *Caps
	t      Function
	values url.Values
}

func (q Query) Clone() *Query {
	values := url.Values{}
	for key := range q.values {
		values[key] = slices.Clone(q.values[key])
	}
	q.values = values
	return &q
}

func (q *Query) IsSupported(param SearchParam) bool {
	switch param {
	case SearchParamT, SearchParamAPIKey:
		return false
	case SearchParamCat, SearchParamAttrs, SearchParamExtended, SearchParamOffset, SearchParamLimit:
		return true
	default:
		return q.caps.SupportsParam(q.t, param)
	}
}

func (q *Query) GetT() Function {
	return q.t
}

func (q *Query) SetT(t Function) *Query {
	if !q.caps.SupportsFunction(t) {
		t = FunctionSearch
	}
	q.t = t
	q.values.Set(SearchParamT, string(t))
	return q
}

func (q *Query) GetCat() []int {
	v := q.values.Get(SearchParamCat)
	if v == "" {
		return []int{}
	}
	parts := strings.Split(v, ",")
	cats := make([]int, 0, len(parts))
	for _, part := range parts {
		cat, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		cats = append(cats, cat)
	}
	return cats
}

func (q *Query) SetCat(cat ...int) *Query {
	var builder strings.Builder
	if v := q.values.Get(SearchParamCat); v != "" {
		builder.WriteString(v)
		builder.WriteString(",")
	}
	for i, c := range cat {
		if i > 1 {
			builder.WriteString(",")
		}
		builder.WriteString(strconv.Itoa(c))
	}
	q.values.Set(SearchParamCat, builder.String())
	return q
}

func (q *Query) GetAttrs() []string {
	v := q.values.Get(SearchParamAttrs)
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}

func (q *Query) SetAttrs(attrs ...string) *Query {
	var builder strings.Builder
	if v := q.values.Get(SearchParamAttrs); v != "" {
		builder.WriteString(v)
		builder.WriteString(",")
	}
	for i, a := range attrs {
		if i > 1 {
			builder.WriteString(",")
		}
		builder.WriteString(a)
	}
	q.values.Set(SearchParamAttrs, builder.String())
	return q
}

func (q *Query) GetExtended() bool {
	v := q.values.Get(SearchParamExtended)
	return v == "1"
}

func (q *Query) SetExtended(value bool) *Query {
	if value {
		q.values.Set(SearchParamExtended, "1")
	} else {
		q.values.Del(SearchParamExtended)
	}
	return q
}

func (q *Query) GetOffset() int {
	return util.SafeParseInt(q.values.Get(SearchParamOffset), 0)
}

func (q *Query) SetOffset(value int) *Query {
	q.values.Set(SearchParamOffset, strconv.Itoa(value))
	return q
}

func (q *Query) GetLimit() int {
	return util.SafeParseInt(q.values.Get(SearchParamLimit), 0)
}

func (q *Query) SetLimit(value int) *Query {
	if value <= 0 {
		value = q.caps.Limits.Max
	}
	q.values.Set(SearchParamLimit, strconv.Itoa(value))
	return q
}

func (q *Query) Set(param SearchParam, value string) *Query {
	q.values.Set(param, value)
	return q
}

func (q *Query) Get(param SearchParam, value string) {
	q.values.Set(param, value)
}

func (q *Query) Values() url.Values {
	return q.values
}

func (q *Query) Encode() string {
	return q.Values().Encode()
}

func (q *Query) String() string {
	return q.Encode()
}
