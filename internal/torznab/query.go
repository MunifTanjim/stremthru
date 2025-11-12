package torznab

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Query struct {
	Type             string
	Q, Series, Movie string
	Ep, Season       string
	Year             int
	Limit, Offset    int
	Extended         bool
	Categories       []int
	APIKey           string

	// identifier types
	TVDBId   string
	TVRageId string
	IMDBId   string
	TVMazeId string
	TraktId  string
}

func (query Query) HasTVShows() bool {
	for _, cat := range query.Categories {
		if 5000 <= cat && cat < 6000 {
			return true
		}
	}
	return false
}

func (query Query) HasMovies() bool {
	for _, cat := range query.Categories {
		if 2000 <= cat && cat < 3000 {
			return true
		}
	}
	return false
}

func (query Query) ToValues() *url.Values {
	v := url.Values{}

	if query.Type != "" {
		v.Set("t", query.Type)
	} else {
		v.Set("t", "search")
	}

	if query.Q != "" {
		v.Set("q", query.Q)
	}

	if query.Ep != "" {
		v.Set("ep", query.Ep)
	}

	if query.Season != "" {
		v.Set("season", query.Season)
	}

	if query.Movie != "" {
		v.Set("movie", query.Movie)
	}

	if query.Year != 0 {
		v.Set("year", strconv.Itoa(query.Year))
	}

	if query.Series != "" {
		v.Set("series", query.Series)
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

	if query.TVDBId != "" {
		v.Set("tvdbid", query.TVDBId)
	}

	if query.TVRageId != "" {
		v.Set("rid", query.TVRageId)
	}

	if query.TVMazeId != "" {
		v.Set("tvmazeid", query.TVMazeId)
	}

	if query.TraktId != "" {
		v.Set("traktid", query.TraktId)
	}

	if query.IMDBId != "" {
		v.Set("imdbid", strings.TrimPrefix(query.IMDBId, "tt"))
	}

	return &v
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
		case "t":
			if len(vals) > 1 {
				return query, errors.New("Multiple t parameters not allowed")
			}
			query.Type = vals[0]

		case "q":
			query.Q = strings.Join(vals, " ")

		case "year":
			if len(vals) > 1 {
				return query, errors.New("Multiple year parameters not allowed")
			}
			year, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, errors.New("Invalid year")
			}
			query.Year = year

		case "ep":
			if len(vals) > 1 {
				return query, errors.New("Multiple ep parameters not allowed")
			}
			if _, err := strconv.Atoi(vals[0]); err != nil {
				return query, errors.New("Invalid ep")
			}
			query.Ep = vals[0]

		case "season":
			if len(vals) > 1 {
				return query, errors.New("Multiple season parameters not allowed")
			}
			if _, err := strconv.Atoi(vals[0]); err != nil {
				return query, errors.New("Invalid season")
			}
			query.Season = vals[0]

		case "apikey":
			if len(vals) > 1 {
				return query, errors.New("Multiple apikey parameters not allowed")
			}
			query.APIKey = vals[0]

		case "limit":
			if len(vals) > 1 {
				return query, errors.New("Multiple limit parameters not allowed")
			}
			limit, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, err
			}
			query.Limit = limit

		case "offset":
			if len(vals) > 1 {
				return query, errors.New("Multiple offset parameters not allowed")
			}
			offset, err := strconv.Atoi(vals[0])
			if err != nil {
				return query, err
			}
			query.Offset = offset

		case "cat":
			query.Categories = []int{}
			for _, val := range vals {
				ints, err := splitInts(val, ",")
				if err != nil {
					return Query{}, fmt.Errorf("Unable to parse cats %q", vals[0])
				}
				query.Categories = append(query.Categories, ints...)
			}

		case "imdbid":
			if len(vals) > 1 {
				return query, errors.New("Multiple imdbid parameters not allowed")
			}
			query.IMDBId = vals[0]
			if !strings.HasPrefix(query.IMDBId, "tt") {
				query.IMDBId = "tt" + query.IMDBId
			}
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
