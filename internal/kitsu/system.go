package kitsu

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	"github.com/MunifTanjim/stremthru/internal/util"
)

func GetSystemKitsu() *APIClient {
	if !config.Integration.Kitsu.HasDefaultCredentials() {
		return nil
	}
	otok, err := oauth.GetOAuthTokenByUserId(oauth.ProviderKitsu, config.Integration.Kitsu.Email)
	if err != nil {
		panic(err)
	}
	if otok == nil {
		tok, err := oauth.KitsuOAuthConfig.PasswordCredentialsToken(config.Integration.Kitsu.Email, config.Integration.Kitsu.Password)
		if err != nil {
			panic(err)
		}
		return GetAPIClient(tok.Extra("id").(string))
	}
	return GetAPIClient(otok.Id)
}

type getAnimeTypeByIdsData struct {
	ResponseError
	Data []struct {
		Attributes struct {
			Subtype AnimeSubtype `json:"subtype"`
		}
		Id string `json:"id"`
	}
}
type GetAnimeTypeByIdsParams struct {
	Ctx
	Ids []int
}

func (c APIClient) GetAnimeTypeByIds(params *GetAnimeTypeByIdsParams) (APIResponse[map[int]AnimeSubtype], error) {
	typeById := map[int]AnimeSubtype{}
	var res *http.Response
	var err error
	for cIds := range slices.Chunk(util.SliceMapIntToString(params.Ids), 20) {
		rParams := &Ctx{}
		query := url.Values{}
		query.Set("page[limit]", "20")
		query.Set("filter[id]", strings.Join(cIds, ","))
		query.Set("fields[anime]", "subtype")
		rParams.Query = &query
		response := getAnimeTypeByIdsData{}
		res, err = c.Request("GET", "/anime", rParams, &response)
		if err != nil {
			return newAPIResponse(res, typeById), err
		}
		for i := range response.Data {
			item := &response.Data[i]
			typeById[util.MustParseInt(item.Id)] = item.Attributes.Subtype
		}
	}
	return newAPIResponse(res, typeById), nil
}
