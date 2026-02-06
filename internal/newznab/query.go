package newznab

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/znab"
)

type Query struct {
	Type          string
	Q             string
	Ep, Season    string
	Year          int
	Limit, Offset int
	Extended      bool
	Categories    []int
	APIKey        string

	IMDBId string
}

func (query Query) HasTVShows() bool {
	if query.Type == string(znab.FunctionSearchTV) {
		return true
	}

	for _, cat := range query.Categories {
		if CategoryTV.ID <= cat && cat < 6000 {
			return true
		}
	}
	return false
}

func (query Query) HasMovies() bool {
	if query.Type == string(znab.FunctionSearchMovie) {
		return true
	}

	for _, cat := range query.Categories {
		if CategoryMovies.ID <= cat && cat < 3000 {
			return true
		}
	}
	return false
}

func (query Query) ToValues() url.Values {
	v := url.Values{}

	v.Set("t", query.Type)

	if query.Q != "" {
		v.Set("q", query.Q)
	}

	if query.Ep != "" {
		v.Set("ep", query.Ep)
	}

	if query.Season != "" {
		v.Set("season", query.Season)
	}

	if query.Year != 0 {
		v.Set("year", strconv.Itoa(query.Year))
	}

	if query.Offset != 0 {
		v.Set("offset", strconv.Itoa(query.Offset))
	}

	if query.Limit != 0 {
		v.Set("limit", strconv.Itoa(query.Limit))
	}

	if query.Extended {
		v.Set("extended", "1")
	}

	if query.APIKey != "" {
		v.Set("apikey", query.APIKey)
	}

	if len(query.Categories) > 0 {
		cats := []string{}

		for _, cat := range query.Categories {
			cats = append(cats, strconv.Itoa(cat))
		}

		v.Set("cat", strings.Join(cats, ","))
	}

	if query.IMDBId != "" {
		v.Set("imdbid", strings.TrimPrefix(query.IMDBId, "tt"))
	}

	return v
}

func (query Query) Encode() string {
	return query.ToValues().Encode()
}

func (query Query) String() string {
	return query.Encode()
}

func ParseQuery(q url.Values) (Query, error) {
	query := Query{}

	for key, vals := range q {
		switch strings.ToLower(key) {
		case znab.SearchParamT:
			if len(vals) > 1 {
				return query, errors.New("multiple t parameters not allowed")
			}
			query.Type = vals[0]

		case znab.SearchParamQ:
			query.Q = strings.Join(vals, " ")

		case znab.SearchParamYear:
			if len(vals) > 1 {
				return query, errors.New("multiple year parameters not allowed")
			}
			year, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, errors.New("invalid year")
			}
			query.Year = year

		case znab.SearchParamEp:
			if len(vals) > 1 {
				return query, errors.New("multiple ep parameters not allowed")
			}
			if _, err := strconv.Atoi(vals[0]); err != nil {
				return query, errors.New("invalid ep")
			}
			query.Ep = vals[0]

		case znab.SearchParamSeason:
			if len(vals) > 1 {
				return query, errors.New("multiple season parameters not allowed")
			}
			if _, err := strconv.Atoi(vals[0]); err != nil {
				return query, errors.New("invalid season")
			}
			query.Season = vals[0]

		case znab.SearchParamAPIKey:
			if len(vals) > 1 {
				return query, errors.New("multiple apikey parameters not allowed")
			}
			query.APIKey = vals[0]

		case znab.SearchParamLimit:
			if len(vals) > 1 {
				return query, errors.New("multiple limit parameters not allowed")
			}
			limit, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, err
			}
			query.Limit = limit

		case znab.SearchParamOffset:
			if len(vals) > 1 {
				return query, errors.New("multiple offset parameters not allowed")
			}
			offset, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, err
			}
			query.Offset = offset

		case znab.SearchParamCat:
			query.Categories = []int{}
			for _, val := range vals {
				ints, err := splitInts(val, ",")
				if err != nil {
					return Query{}, fmt.Errorf("unable to parse cats %q", vals[0])
				}
				query.Categories = append(query.Categories, ints...)
			}

		case znab.SearchParamIMDBId:
			if len(vals) > 1 {
				return query, errors.New("multiple imdbid parameters not allowed")
			}
			query.IMDBId = vals[0]
			if !strings.HasPrefix(query.IMDBId, "tt") {
				query.IMDBId = "tt" + query.IMDBId
			}

		case znab.SearchParamExtended:
			query.Extended = vals[0] == "1"
		}
	}

	return query, nil
}

func splitInts(s, delim string) (i []int, err error) {
	for v := range strings.SplitSeq(s, delim) {
		vInt, err := strconv.Atoi(v)
		if err != nil {
			return i, err
		}
		i = append(i, vInt)
	}
	return i, err
}
