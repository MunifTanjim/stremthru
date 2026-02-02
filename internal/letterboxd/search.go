package letterboxd

import (
	"net/url"
	"strconv"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type SearchMethod string

var (
	SearchMethodFullText         SearchMethod = "FullText"
	SearchMethodAutocomplete     SearchMethod = "Autocomplete"
	SearchMethodNamesAndKeywords SearchMethod = "NamesAndKeywords"
)

type SearchResultType string

var (
	SearchResultTypeContributorSearchItem SearchResultType = "ContributorSearchItem"
	SearchResultTypeFilmSearchItem        SearchResultType = "FilmSearchItem"
	SearchResultTypeListSearchItem        SearchResultType = "ListSearchItem"
	SearchResultTypeMemberSearchItem      SearchResultType = "MemberSearchItem"
	SearchResultTypeReviewSearchItem      SearchResultType = "ReviewSearchItem"
	SearchResultTypeTagSearchItem         SearchResultType = "TagSearchItem"
	SearchResultTypeStorySearchItem       SearchResultType = "StorySearchItem"
	SearchResultTypeArticleSearchItem     SearchResultType = "ArticleSearchItem"
	SearchResultTypePodcastSearchItem     SearchResultType = "PodcastSearchItem"
)

type ContributionType string

var (
	ContributionTypeDirector              ContributionType = "Director"
	ContributionTypeCoDirector            ContributionType = "CoDirector"
	ContributionTypeActor                 ContributionType = "Actor"
	ContributionTypeProducer              ContributionType = "Producer"
	ContributionTypeWriter                ContributionType = "Writer"
	ContributionTypeOriginalWriter        ContributionType = "OriginalWriter"
	ContributionTypeStory                 ContributionType = "Story"
	ContributionTypeCasting               ContributionType = "Casting"
	ContributionTypeEditor                ContributionType = "Editor"
	ContributionTypeCinematography        ContributionType = "Cinematography"
	ContributionTypeAssistantDirector     ContributionType = "AssistantDirector"
	ContributionTypeAdditionalDirecting   ContributionType = "AdditionalDirecting"
	ContributionTypeExecutiveProducer     ContributionType = "ExecutiveProducer"
	ContributionTypeLighting              ContributionType = "Lighting"
	ContributionTypeCameraOperator        ContributionType = "CameraOperator"
	ContributionTypeAdditionalPhotography ContributionType = "AdditionalPhotography"
	ContributionTypeProductionDesign      ContributionType = "ProductionDesign"
	ContributionTypeArtDirection          ContributionType = "ArtDirection"
	ContributionTypeSetDecoration         ContributionType = "SetDecoration"
	ContributionTypeSpecialEffects        ContributionType = "SpecialEffects"
	ContributionTypeVisualEffects         ContributionType = "VisualEffects"
	ContributionTypeTitleDesign           ContributionType = "TitleDesign"
	ContributionTypeStunts                ContributionType = "Stunts"
	ContributionTypeChoreography          ContributionType = "Choreography"
	ContributionTypeComposer              ContributionType = "Composer"
	ContributionTypeSongs                 ContributionType = "Songs"
	ContributionTypeSound                 ContributionType = "Sound"
	ContributionTypeCostumes              ContributionType = "Costumes"
	ContributionTypeCreator               ContributionType = "Creator"
	ContributionTypeMakeUp                ContributionType = "MakeUp"
	ContributionTypeHairstyling           ContributionType = "Hairstyling"
	ContributionTypeStudio                ContributionType = "Studio"
)

type SearchParams struct {
	Ctx
	Cursor                         string
	PerPage                        int // default 20, max 100
	Input                          string
	SearchMethod                   SearchMethod
	Include                        SearchResultType
	ContributionType               ContributionType
	Adult                          bool
	ExcludeMemberFilmRelationships bool
}

type AbstractSearchItem struct {
	Score  float32          `json:"score"`
	Type   SearchResultType `json:"type"`
	Member *MemberSummary   `json:"member,omitempty"`
	List   *ListSummary     `json:"list,omitempty"`
}

type SearchResponse struct {
	ResponseError
	Next      string               `json:"next"`
	Items     []AbstractSearchItem `json:"items"`
	ItemCount int                  `json:"itemCount"`
}

func (c *APIClient) Search(params *SearchParams) (request.APIResponse[SearchResponse], error) {
	query := url.Values{}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}
	if params.PerPage > 0 {
		if params.PerPage > 100 {
			panic("perPage maximum is 100")
		}
		query.Set("perPage", strconv.Itoa(params.PerPage))
	}
	if params.Input != "" {
		query.Set("input", params.Input)
	}
	if params.SearchMethod != "" {
		query.Set("searchMethod", string(params.SearchMethod))
	}
	if params.Include != "" {
		query.Set("include", string(params.Include))
	}
	if params.ContributionType != "" {
		query.Set("contributionType", string(params.ContributionType))
	}
	if params.Adult {
		query.Set("adult", "true")
	}
	if params.ExcludeMemberFilmRelationships {
		query.Set("excludeMemberFilmRelationships", "true")
	}
	params.Query = &query
	response := SearchResponse{}
	res, err := c.Request("GET", "/v0/search", params, &response)
	return request.NewAPIResponse(res, response), err
}
