package newznab_client

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/util"
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
		if i > 0 {
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
		if i > 0 {
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
	if value <= 0 && q.caps.Limits != nil {
		value = q.caps.Limits.Max
	}
	if value > 0 {
		q.values.Set(SearchParamLimit, strconv.Itoa(value))
	}
	return q
}

func (q *Query) Set(param SearchParam, value string) *Query {
	q.values.Set(param, value)
	return q
}

func (q *Query) Get(param SearchParam) string {
	return q.values.Get(param)
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

func (q *Query) GetHeader() http.Header {
	var h http.Header
	switch q.t {
	case FunctionSearchTV:
		h = config.Newz.IndexerRequestHeader.Query.Get(config.NewzIndexerRequestQueryTypeTV)
	case FunctionSearchMovie:
		h = config.Newz.IndexerRequestHeader.Query.Get(config.NewzIndexerRequestQueryTypeMovie)
	default:
		h = config.Newz.IndexerRequestHeader.Query.Get(config.NewzIndexerRequestQueryTypeAny)
	}
	return h
}
