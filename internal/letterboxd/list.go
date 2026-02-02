package letterboxd

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/request"
)

type ListEntryOccurrence struct {
	Rank   int    `json:"rank"`
	FilmID string `json:"filmId"`
}

type ListSummary struct {
	Id                   string                `json:"id"`
	Name                 string                `json:"name"`
	Version              int64                 `json:"version,omitempty"`
	FilmCount            int                   `json:"filmCount"`
	Published            bool                  `json:"published"`
	Ranked               bool                  `json:"ranked"`
	DescriptionLbml      string                `json:"descriptionLbml,omitempty"`
	DescriptionTruncated bool                  `json:"descriptionTruncated,omitempty"`
	SharePolicy          SharePolicy           `json:"sharePolicy"`
	Owner                MemberSummary         `json:"owner"`
	ClonedFrom           *ListIdentifier       `json:"clonedFrom,omitempty"`
	PreviewEntries       []ListEntrySummary    `json:"previewEntries"`
	EntriesOfNote        []ListEntryOccurrence `json:"entriesOfNote,omitempty"`
	Description          string                `json:"description,omitempty"`
	WhenCreated          string                `json:"whenCreated"`
	WhenPublished        string                `json:"whenPublished,omitempty"`
	WhenUpdated          string                `json:"whenUpdated"`
}

type Tag struct {
	Code       string `json:"code"`
	DisplayTag string `json:"displayTag"`
}

type CommentPolicy string

const (
	CommentPolicyAnyone  CommentPolicy = "Anyone"
	CommentPolicyFriends CommentPolicy = "Friends"
	CommentPolicyYou     CommentPolicy = "You"
)

type SharePolicy string

const (
	SharePolicyAnyone  SharePolicy = "Anyone"
	SharePolicyFriends SharePolicy = "Friends"
	SharePolicyYou     SharePolicy = "You"
)

type Pronoun struct {
	Id                  string `json:"id"`
	Label               string `json:"label"`
	SubjectPronoun      string `json:"subjectPronoun"`
	ObjectPronoun       string `json:"objectPronoun"`
	PossessiveAdjective string `json:"possessiveAdjective"`
	PossessivePronoun   string `json:"possessivePronoun"`
	Reflexive           string `json:"reflexive"`
}

type ImageSize struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type Image struct {
	Sizes []ImageSize `json:"sizes"`
}

type ListIdentifier struct {
	Id string `json:"id"`
}

type ContributorSummary struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	CharacterName string `json:"characterName,omitempty"`
	TMDBId        string `json:"tmdbid,omitempty"`
	CustomPoster  *Image `json:"customPoster,omitempty"`
}

type PrivacyPolicy string

const (
	PrivacyPolicyAnyone  PrivacyPolicy = "Anyone"
	PrivacyPolicyFriends PrivacyPolicy = "Friends"
	PrivacyPolicyYou     PrivacyPolicy = "You"
	PrivacyPolicyDraft   PrivacyPolicy = "Draft"
)

type FilmRelationship struct {
	Watched                  bool          `json:"watched"`
	WhenWatched              string        `json:"whenWatched,omitempty"`
	Liked                    bool          `json:"liked"`
	WhenLiked                string        `json:"whenLiked,omitempty"`
	Favorited                bool          `json:"favorited"`
	Owned                    bool          `json:"owned,omitempty"`
	InWatchlist              bool          `json:"inWatchlist,omitempty"`
	WhenAddedToWatchlist     string        `json:"whenAddedToWatchlist,omitempty"`
	WhenCompletedInWatchlist string        `json:"whenCompletedInWatchlist,omitempty"`
	Rating                   float32       `json:"rating,omitempty"`
	Reviews                  []string      `json:"reviews"`
	DiaryEntries             []string      `json:"diaryEntries"`
	Rewatched                bool          `json:"rewatched,omitempty"`
	PrivacyPolicy            PrivacyPolicy `json:"privacyPolicy,omitempty"`
}

type MemberFilmRelationship struct {
	Member       MemberSummary    `json:"member"`
	Relationship FilmRelationship `json:"relationship"`
}

type Genre struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type ListEntrySummary struct {
	Rank int         `json:"rank,omitempty"`
	Film FilmSummary `json:"film"`
}

type LinkType string

const (
	LinkTypeLetterboxd LinkType = "letterboxd"
	LinkTypeBoxd       LinkType = "boxd"
	LinkTypeTMDB       LinkType = "tmdb"
	LinkTypeIMDB       LinkType = "imdb"
	LinkTypeJustWatch  LinkType = "justwatch"
	LinkTypeFacebook   LinkType = "facebook"
	LinkTypeInstagram  LinkType = "instagram"
	LinkTypeTwitter    LinkType = "twitter"
	LinkTypeYouTube    LinkType = "youtube"
	LinkTypeTickets    LinkType = "tickets"
	LinkTypeTikTok     LinkType = "tiktok"
	LinkTypeBluesky    LinkType = "bluesky"
	LinkTypeThreads    LinkType = "threads"
)

type Link struct {
	Type     LinkType `json:"type"`
	Id       string   `json:"id"`
	URL      string   `json:"url"`
	Label    string   `json:"label,omitempty"`
	CheckURL string   `json:"checkUrl,omitempty"`
}

type List struct {
	Id                  string             `json:"id"`
	Name                string             `json:"name"`
	Version             int64              `json:"version,omitempty"`
	FilmCount           int                `json:"filmCount"`
	Published           bool               `json:"published"`
	Ranked              bool               `json:"ranked"`
	HasEntriesWithNotes bool               `json:"hasEntriesWithNotes"`
	DescriptionLbml     string             `json:"descriptionLbml,omitempty"`
	Tags                []string           `json:"tags,omitempty"`
	Tags2               []Tag              `json:"tags2"`
	WhenCreated         string             `json:"whenCreated"`
	WhenPublished       string             `json:"whenPublished,omitempty"`
	WhenUpdated         string             `json:"whenUpdated"`
	CommentPolicy       CommentPolicy      `json:"commentPolicy,omitempty"`
	SharePolicy         SharePolicy        `json:"sharePolicy"`
	Owner               MemberSummary      `json:"owner"`
	ClonedFrom          *ListIdentifier    `json:"clonedFrom,omitempty"`
	PreviewEntries      []ListEntrySummary `json:"previewEntries"`
	Links               []Link             `json:"links"`
	Backdrop            *Image             `json:"backdrop,omitempty"`
	BackdropFocalPoint  float32            `json:"backdropFocalPoint,omitempty"`
	Description         string             `json:"description,omitempty"`
}

func (l List) getLetterboxdLink() string {
	for i := range l.Links {
		link := &l.Links[i]
		if link.Type == LinkTypeLetterboxd {
			return link.URL
		}
	}
	return ""
}

func (l List) GetLetterboxdSlug() string {
	link := l.getLetterboxdLink()
	if link == "" {
		return ""
	}
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"+strings.ToLower(l.Owner.Username)+"/list/"), "/")
}

type fetchListData struct {
	ResponseError
	List
}

type FetchListParams struct {
	Ctx
	Id string
}

func (c *APIClient) FetchList(params *FetchListParams) (request.APIResponse[List], error) {
	response := fetchListData{}
	res, err := c.Request("GET", "/v0/list/"+params.Id, params, &response)
	return request.NewAPIResponse(res, response.List), err
}

type ListEntry struct {
	Rank              int         `json:"rank,omitempty"`
	EntryId           string      `json:"entryId"`
	NotesLbml         string      `json:"notesLbml,omitempty"`
	PosterPickerURL   string      `json:"posterPickerUrl,omitempty"`   // FIRST PARTY
	BackdropPickerURL string      `json:"backdropPickerUrl,omitempty"` // FIRST PARTY
	ContainsSpoilers  bool        `json:"containsSpoilers,omitempty"`
	Film              FilmSummary `json:"film"`
	WhenAdded         string      `json:"whenAdded"`
	Notes             string      `json:"notes,omitempty"`
}

type FilmsMetadata struct {
	TotalFilmCount    int `json:"totalFilmCount"`
	FilteredFilmCount int `json:"filteredFilmCount"`
}

type FilmsRelationshipCounts struct {
	Watches int32 `json:"watches"`
	Likes   int32 `json:"likes"`
}

type FilmsRelationship struct {
	Counts FilmsRelationshipCounts `json:"counts"`
}

type FilmsMemberRelationship struct {
	Member       MemberSummary     `json:"member"`
	Relationship FilmsRelationship `json:"relationship"`
}

type ListEntriesResponse struct {
	ResponseError
	Next          string                    `json:"next,omitempty"`
	Items         []ListEntry               `json:"items"`
	ItemCount     int                       `json:"itemCount"`
	Metadata      FilmsMetadata             `json:"metadata"`
	Relationships []FilmsMemberRelationship `json:"relationships"`
}

type FetchListEntriesParams struct {
	Ctx
	Id      string
	Cursor  string
	PerPage int // default 20, max 100
}

func (c *APIClient) FetchListEntries(params *FetchListEntriesParams) (request.APIResponse[ListEntriesResponse], error) {
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
	params.Query = &query
	response := ListEntriesResponse{}
	res, err := c.Request("GET", "/v0/list/"+params.Id+"/entries", params, &response)
	return request.NewAPIResponse(res, response), err
}
