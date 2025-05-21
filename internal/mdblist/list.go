package mdblist

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/MunifTanjim/stremthru/core"
)

type MediaType string

const (
	MediaTypeShow  = "show"
	MediaTypeMovie = "movie"
)

type List struct {
	Id          int       `json:"id"`
	UserId      int       `json:"user_id"`
	UserName    string    `json:"user_name"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Mediatype   MediaType `json:"mediatype"`
	Items       int       `json:"items"`
	Likes       int       `json:"likes"`
	Dynamic     bool      `json:"dynamic"`
	Private     bool      `json:"private"`
	Updated     time.Time `json:"updated"`
	Expires     time.Time `json:"expires"`
	Movies      int       `json:"movies"`
	Shows       int       `json:"shows"`

	is_fetched   bool      `json:"-"`
	refreshed_at time.Time `json:"-"`
}

func (l *List) GetURL() string {
	if l.UserName != "" && l.Slug != "" {
		return "https://mdblist.com/lists/" + l.UserName + "/" + l.Slug
	}
	if l.Id != 0 {
		return "https://mdblist.com/?list=" + strconv.Itoa(l.Id)
	}
	return ""
}

type fetchListData struct {
	ResponseContainer
	data List
}

func (d *fetchListData) UnmarshalJSON(data []byte) error {
	var rerr ResponseContainer

	if err := json.Unmarshal(data, &rerr); err == nil {
		d.ResponseContainer = rerr
		return nil
	}

	var items []List
	if err := json.Unmarshal(data, &items); err == nil {
		d.data = items[0]
		return nil
	}

	return core.NewAPIError("failed to parse response")
}

type FetchListByIdParams struct {
	Ctx
	ListId int
}

func (c APIClient) FetchListById(params *FetchListByIdParams) (APIResponse[List], error) {
	response := &fetchListData{}
	res, err := c.Request("GET", "/lists/"+strconv.Itoa(params.ListId), params, response)
	return newAPIResponse(res, response.data), err
}

type FetchListByNameParams struct {
	Ctx
	UserName string
	Slug     string
}

func (c APIClient) FetchListByName(params *FetchListByNameParams) (APIResponse[List], error) {
	response := &fetchListData{}
	res, err := c.Request("GET", "/lists/"+params.UserName+"/"+params.Slug, params, response)
	return newAPIResponse(res, response.data), err
}
