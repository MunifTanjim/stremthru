package stremio_list

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/anilist"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/letterboxd"
	"github.com/MunifTanjim/stremthru/internal/mdblist"
	"github.com/MunifTanjim/stremthru/internal/oauth"
	stremio_shared "github.com/MunifTanjim/stremthru/internal/stremio/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	"github.com/MunifTanjim/stremthru/internal/tmdb"
	"github.com/MunifTanjim/stremthru/internal/trakt"
	"github.com/MunifTanjim/stremthru/internal/tvdb"
	"github.com/MunifTanjim/stremthru/internal/util"
)

type UserData struct {
	Lists        []string `json:"lists"`
	ListNames    []string `json:"list_names"`
	ListTypes    []string `json:"list_types"`
	ListShuffle  []int    `json:"list_shuffle"`
	list_urls    []string `json:"-"`
	MDBListLists []int    `json:"mdblist_lists,omitempty"` // deprecated

	MDBListAPIkey string                   `json:"mdblist_api_key,omitempty"`
	mdblistUser   *mdblist.GetMyLimitsData `json:"-"`

	TMDBTokenId string            `json:"tmdb_token_id,omitempty"`
	tmdbToken   *oauth.OAuthToken `json:"-"`

	TraktTokenId string            `json:"trakt_token_id,omitempty"`
	traktToken   *oauth.OAuthToken `json:"-"`

	RPDBAPIKey       string `json:"rpdb_api_key,omitempty"`
	TopPostersAPIKey string `json:"top_posters_api_key,omitempty"`

	MetaIdMovie  string `json:"meta_id_movie,omitempty"`
	MetaIdSeries string `json:"meta_id_series,omitempty"`
	MetaIdAnime  string `json:"meta_id_anime,omitempty"`

	Shuffle bool `json:"shuffle,omitempty"`

	encoded string `json:"-"` // correctly configured

	mdblistById    map[string]mdblist.MDBListList       `json:"-"`
	anilistById    map[string]anilist.AniListList       `json:"-"`
	traktById      map[string]trakt.TraktList           `json:"-"`
	tmdbById       map[string]tmdb.TMDBList             `json:"-"`
	tvdbById       map[string]tvdb.TVDBList             `json:"-"`
	letterboxdById map[string]letterboxd.LetterboxdList `json:"-"`
}

func (ud UserData) StripSecrets() UserData {
	ud.MDBListAPIkey = ""
	ud.TMDBTokenId = ""
	ud.TraktTokenId = ""
	ud.RPDBAPIKey = ""
	ud.TopPostersAPIKey = ""
	return ud
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "list",
})

func (ud UserData) HasRequiredValues() bool {
	return len(ud.Lists) != 0
}

func (ud UserData) GetEncoded() string {
	return ud.encoded
}

func (ud *UserData) SetEncoded(encoded string) {
	ud.encoded = encoded
}

func (ud *UserData) Ptr() *UserData {
	return ud
}

type userDataError struct {
	mdblist struct {
		api_key string
	}
	list_urls           []string
	tmdb_token_id       string
	trakt_token_id      string
	meta_id_movie       string
	meta_id_series      string
	meta_id_anime       string
	rpdb_api_key        string
	top_posters_api_key string
}

func (uderr userDataError) HasError() bool {
	if uderr.mdblist.api_key != "" {
		return true
	}
	for i := range uderr.list_urls {
		if uderr.list_urls[i] != "" {
			return true
		}
	}
	if uderr.rpdb_api_key != "" {
		return true
	}
	if uderr.top_posters_api_key != "" {
		return true
	}
	return false
}

func (uderr userDataError) Error() string {
	var str strings.Builder
	if uderr.mdblist.api_key != "" {
		str.WriteString("mdblist.api_key: " + uderr.mdblist.api_key + "\n")
	}
	for i, err := range uderr.list_urls {
		if err != "" {
			str.WriteString("mdblist.list[" + strconv.Itoa(i) + "].url: " + err + "\n")
		}
	}
	return str.String()
}

func getUserData(r *http.Request, isAuthed bool) (*UserData, error) {
	ud := &UserData{}
	ud.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(ud); err != nil {
			return nil, err
		}
		if ud.encoded == "" {
			return ud, nil
		}

		if len(ud.MDBListLists) > 0 {
			for _, id := range ud.MDBListLists {
				ud.Lists = append(ud.Lists, "mdblist:"+strconv.Itoa(id))
			}

			ud.MDBListLists = nil

			if err := udManager.Sync(ud); err != nil {
				return nil, err
			}
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		isExecutingAction := stremio_shared.GetConfigureAction(r) != ""

		udErr := userDataError{}

		ud.MDBListAPIkey = r.Form.Get("mdblist_api_key")
		ud.TMDBTokenId = r.Form.Get("tmdb_token_id")
		ud.TraktTokenId = r.Form.Get("trakt_token_id")

		ud.RPDBAPIKey = r.Form.Get("rpdb_api_key")
		ud.TopPostersAPIKey = r.Form.Get("top_posters_api_key")

		if ud.RPDBAPIKey != "" && ud.TopPostersAPIKey != "" {
			err := "Only one poster provider can be used at a time"
			udErr.rpdb_api_key = err
			udErr.top_posters_api_key = err
		}

		ud.MetaIdMovie = r.Form.Get("meta_id_movie")
		ud.MetaIdSeries = r.Form.Get("meta_id_series")
		ud.MetaIdAnime = r.Form.Get("meta_id_anime")

		ud.Shuffle = r.Form.Get("shuffle") == "on"

		lists_length := 0
		if v := r.Form.Get("lists_length"); v != "" {
			if lists_length, err = strconv.Atoi(v); err != nil {
				return nil, err
			}
		}

		if lists_length == 0 {
			err := userDataError{}
			err.list_urls = []string{"Missing List URL"}
			return ud, err
		}

		isLetterboxdEnabled := LetterboxdEnabled
		isMDBListEnabled := ud.MDBListAPIkey != ""
		isTMDBConfigured := TMDBEnabled && ud.TMDBTokenId != ""
		isTraktTvConfigured := TraktEnabled && ud.TraktTokenId != ""
		isTVDBConfigured := TVDBEnabled

		if isMDBListEnabled {
			if _, err := ud.getMDBListUser(); err != nil {
				udErr.mdblist.api_key = "Invalid API Key: " + err.Error()
			}
		}

		if ud.TopPostersAPIKey != "" && udErr.top_posters_api_key == "" {
			if err := verifyTopPostersAPIKey(ud.TopPostersAPIKey); err != nil {
				udErr.top_posters_api_key = "Failed to Verify: " + err.Error()
			}
		}

		if isTMDBConfigured {
			ud.tmdbToken, err = ud.getTMDBToken()
			if err != nil {
				udErr.tmdb_token_id = err.Error()
			}
			isTMDBConfigured = ud.TMDBTokenId != ""
		}

		if isTraktTvConfigured {
			ud.traktToken, err = ud.getTraktToken()
			if err != nil {
				udErr.trakt_token_id = err.Error()
			}
			isTraktTvConfigured = ud.TraktTokenId != ""
		}

		ud.Lists = make([]string, 0, lists_length)
		ud.ListNames = make([]string, 0, lists_length)
		ud.ListTypes = make([]string, 0, lists_length)
		ud.ListShuffle = make([]int, 0, lists_length)

		ud.list_urls = make([]string, 0, lists_length)
		udErr.list_urls = make([]string, 0, lists_length)

		idx := -1
		for i := range lists_length {
			listId := r.Form.Get("lists[" + strconv.Itoa(i) + "].id")
			listUrlStr := r.Form.Get("lists[" + strconv.Itoa(i) + "].url")
			if !isExecutingAction && listId == "" && listUrlStr == "" {
				continue
			}

			idx++

			ud.Lists = append(ud.Lists, listId)
			ud.ListNames = append(ud.ListNames, r.Form.Get("lists["+strconv.Itoa(i)+"].name"))
			ud.ListTypes = append(ud.ListTypes, r.Form.Get("lists["+strconv.Itoa(i)+"].type"))
			if r.Form.Get("lists["+strconv.Itoa(i)+"].shuffle") == "on" {
				ud.ListShuffle = append(ud.ListShuffle, 1)
			} else {
				ud.ListShuffle = append(ud.ListShuffle, 0)
			}

			ud.list_urls = append(ud.list_urls, listUrlStr)
			udErr.list_urls = append(udErr.list_urls, "")

			if listUrlStr == "" {
				continue
			}

			listUrl, err := url.Parse(listUrlStr)
			if err != nil {
				udErr.list_urls[idx] = "Invalid List URL: " + err.Error()
				continue
			}

			switch listUrl.Hostname() {
			case "anilist.co":
				if !AnimeEnabled {
					udErr.list_urls[idx] = "Unsupported List URL"
					continue
				}

				list := anilist.AniListList{}
				if restPath, ok := strings.CutPrefix(listUrl.Path, "/user/"); ok {
					parts := strings.SplitN(restPath, "/", 3)
					if len(parts) != 3 || parts[1] != "animelist" {
						udErr.list_urls[idx] = "Invalid AniList URL"
						continue
					}
					userName, listName := parts[0], parts[2]
					if userName == "" || listName == "" {
						udErr.list_urls[idx] = "Invalid AniList URL"
						continue
					}
					list.Id = userName + ":" + listName
				} else if restPath, ok := strings.CutPrefix(listUrl.Path, "/search/anime/"); ok {
					name := restPath
					if !anilist.IsValidSearchList(name) {
						udErr.list_urls[idx] = "Unsupported AniList URL"
						continue
					}
					list.Id = "~:" + name
				} else {
					udErr.list_urls[idx] = "Unsupported AniList URL"
					continue
				}

				err := ud.FetchAniListList(&list, true)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.Lists[idx] = "anilist:" + list.Id

			case "letterboxd.com":
				if !isLetterboxdEnabled {
					udErr.list_urls[idx] = "Unsupported List URL"
					continue
				}

				list := letterboxd.LetterboxdList{}
				parts := strings.Split(strings.Trim(listUrl.Path, "/"), "/")
				switch {
				case len(parts) == 3 && parts[1] == "list":
					username, slug := parts[0], parts[2]
					if username == "" || slug == "" {
						udErr.list_urls[idx] = "Invalid List URL"
						continue
					}
					listId, err := letterboxd.FetchLetterboxdListIdentifier(username, slug)
					if err != nil {
						udErr.list_urls[idx] = "Failed to fetch list identifier: " + err.Error()
						continue
					}
					userId, err := letterboxd.FetchLetterboxdUserIdentifier(username)
					if err != nil {
						udErr.list_urls[idx] = "Failed to fetch user identifier: " + err.Error()
						continue
					}
					list.Id = listId
					list.UserId = userId
					list.UserName = username
					list.Slug = slug
				case len(parts) == 2 && parts[1] == "watchlist":
					username, slug := parts[0], parts[1]
					if username == "" || slug == "" {
						udErr.list_urls[idx] = "Invalid List URL"
						continue
					}
					userId, err := letterboxd.FetchLetterboxdUserIdentifier(username)
					if err != nil {
						udErr.list_urls[idx] = "Failed to fetch user identifier: " + err.Error()
						continue
					}
					list.Id = letterboxd.ID_PREFIX_USER_WATCHLIST + userId
					list.UserId = userId
					list.UserName = username
					list.Slug = slug
				default:
					udErr.list_urls[idx] = "Invalid List URL"
					continue
				}

				err := ud.FetchLetterboxdList(&list)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.Lists[idx] = "letterboxd:" + list.Id

			case "mdblist.com":
				if !isMDBListEnabled {
					udErr.list_urls[idx] = "MDBList API Key is required"
					continue
				}

				query := listUrl.Query()
				list := mdblist.MDBListList{}
				if idStr := query.Get("list"); idStr != "" {
					list.Id = idStr
				} else if restPath, ok := strings.CutPrefix(listUrl.Path, "/lists/"); ok {
					username, slug, _ := strings.Cut(restPath, "/")
					if username != "" && slug != "" && !strings.Contains(slug, "/") {
						list.UserName = username
						list.Slug = slug
					} else {
						udErr.list_urls[idx] = "Invalid List URL"
						continue
					}
				} else if restPath, ok := strings.CutPrefix(listUrl.Path, "/watchlist/"); ok {
					username := restPath
					if user, err := ud.getMDBListUser(); err != nil {
						udErr.list_urls[idx] = "Failed to fetch user: " + err.Error()
						continue
					} else {
						if !strings.EqualFold(username, user.Username) {
							udErr.list_urls[idx] = "Invalid URL: not own list"
							continue
						}
					}
					list.Id = "~:watchlist:" + username
					list.UserName = username
					list.Slug = "watchlist/" + username
				} else {
					udErr.list_urls[idx] = "Invalid List URL"
					continue
				}

				err := ud.FetchMDBListList(&list)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.Lists[idx] = "mdblist:" + list.Id

			case "www.themoviedb.org", "themoviedb.org":
				if !isTMDBConfigured {
					if TMDBEnabled {
						udErr.list_urls[idx] = "TMDB Auth Code is required"
					} else {
						udErr.list_urls[idx] = "Unsupported List URL"
					}
					continue
				}

				list := tmdb.TMDBList{}
				switch {
				case strings.HasPrefix(listUrl.Path, "/company/"):
					parts := strings.SplitN(strings.Trim(strings.TrimPrefix(listUrl.Path, "/company/"), "/"), "/", 2)
					if len(parts) != 2 {
						udErr.list_urls[idx] = "Invalid TMDB URL"
						continue
					}
					companyId, _, _ := strings.Cut(parts[0], "-")
					if !util.IsNumericString(companyId) {
						udErr.list_urls[idx] = "Invalid TMDB URL"
						continue
					}
					listType := parts[1]
					list.Id = tmdb.ID_PREFIX_DYNAMIC_COMPANY + companyId + ":" + listType
				case strings.HasPrefix(listUrl.Path, "/network/"):
					identifier := strings.Trim(strings.TrimPrefix(listUrl.Path, "/network/"), "/")
					if strings.Contains(identifier, "/") {
						udErr.list_urls[idx] = "Invalid TMDB URL"
						continue
					}
					networkId, _, _ := strings.Cut(identifier, "-")
					if !util.IsNumericString(networkId) {
						udErr.list_urls[idx] = "Invalid TMDB URL"
						continue
					}
					list.Id = tmdb.ID_PREFIX_DYNAMIC_NETWORK + networkId
				case strings.HasPrefix(listUrl.Path, "/list/"):
					parts := strings.SplitN(strings.TrimPrefix(listUrl.Path, "/list/"), "-", 2)
					if !util.IsNumericString(parts[0]) {
						udErr.list_urls[idx] = "Invalid TMDB URL"
						continue
					}
					list.Id = parts[0]
				case strings.HasPrefix(listUrl.Path, "/movie") || strings.HasPrefix(listUrl.Path, "/tv"):
					meta := tmdb.GetDynamicListMeta(listUrl.Path)
					if meta == nil {
						udErr.list_urls[idx] = "Unsupported TMDB URL"
						continue
					}

					list.Id = "~:" + strings.TrimPrefix(listUrl.Path, "/")
				case strings.HasPrefix(listUrl.Path, "/u/"):
					parts := strings.SplitN(strings.TrimPrefix(listUrl.Path, "/u/"), "/", 3)
					username := parts[0]
					if strings.ToLower(username) != strings.ToLower(ud.tmdbToken.UserName) {
						udErr.list_urls[idx] = "Invalid URL: not own list"
						continue
					}
					switch parts[1] {
					case "favorites", "recommendations", "ratings", "watchlist":
						listType := "movie"
						if len(parts) == 3 {
							listType = parts[2]
						}
						list.Id = "~:u:" + parts[1] + "/" + listType
						list.Username = username
					default:
						udErr.list_urls[idx] = "Unsupported TMDB URL"
						continue
					}
				default:
					udErr.list_urls[idx] = "Unsupported TMDB URL"
					continue
				}

				err := ud.FetchTMDBList(&list)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.Lists[idx] = "tmdb:" + list.Id

			case "trakt.tv":
				if !isTraktTvConfigured {
					if TraktEnabled {
						udErr.list_urls[idx] = "Trakt.tv Auth Code is required"
					} else {
						udErr.list_urls[idx] = "Unsupported List URL"
					}
					continue
				}

				list := trakt.TraktList{}
				switch {
				case strings.HasPrefix(listUrl.Path, "/users/"):
					parts := strings.SplitN(strings.TrimPrefix(listUrl.Path, "/users/"), "/", 3)
					switch {
					case len(parts) == 3 && parts[1] == "lists":
						userSlug, listSlug := parts[0], parts[2]
						if userSlug == "" || listSlug == "" {
							udErr.list_urls[idx] = "Invalid Trakt.tv URL"
							continue
						}
						list.UserId = userSlug
						list.Slug = listSlug

					case len(parts) == 2:
						switch parts[1] {
						case "collection", "favorites", "watchlist":
							list.Id = "~:" + parts[1] + ":" + parts[0]
							list.UserId = parts[0]
						case "progress":
							list.Id = trakt.ID_PREFIX_DYNAMIC_USER_SPECIFIC + "users/" + parts[0] + "/" + parts[1]
							list.UserId = parts[0]
							if list.UserId != ud.traktToken.UserId {
								udErr.list_urls[idx] = "Invalid URL: not own list"
								continue
							}
						default:
							udErr.list_urls[idx] = "Unsupported Trakt.tv URL"
							continue
						}
					default:
						udErr.list_urls[idx] = "Unsupported Trakt.tv URL"
						continue
					}

				default:
					meta := trakt.GetDynamicListMeta(listUrl.Path)
					if meta == nil {
						udErr.list_urls[idx] = "Unsupported Trakt.tv URL"
						continue
					}

					list.Id = meta.Id
					if list.Id == "" {
						list.Id = "~:" + strings.TrimPrefix(listUrl.Path, "/")
					}
					list.Slug = strings.TrimPrefix(listUrl.Path, "/")
				}

				err := ud.FetchTraktList(&list)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				if util.IsNumericString(list.Id) && list.UserId != "" && list.Slug != "" {
					ud.Lists[idx] = "trakt:" + list.UserId + "." + list.Slug
				} else {
					ud.Lists[idx] = "trakt:" + list.Id
				}

			case "www.thetvdb.com", "thetvdb.com":
				if !isTVDBConfigured {
					udErr.list_urls[idx] = "Unsupported List URL"
					continue
				}

				list := tvdb.TVDBList{}
				switch {
				case strings.HasPrefix(listUrl.Path, "/lists/"):
					idOrSlug := strings.TrimPrefix(listUrl.Path, "/lists/")
					if util.IsNumericString(idOrSlug) {
						list.Id = idOrSlug
					} else {
						list.Slug = idOrSlug
					}
				default:
					udErr.list_urls[idx] = "Unsupported TVDB URL"
					continue
				}

				err := ud.FetchTVDBList(&list)
				if err != nil {
					udErr.list_urls[idx] = "Failed to fetch List: " + err.Error()
					continue
				}
				ud.Lists[idx] = "tvdb:" + list.Id
			}
		}

		if udErr.HasError() {
			return ud, udErr
		}
	}

	if IsPublicInstance && len(ud.Lists) > MaxPublicInstanceListCount {
		ud.Lists = ud.Lists[0:MaxPublicInstanceListCount]
	}

	return ud, nil
}

func (ud *UserData) getTraktToken() (*oauth.OAuthToken, error) {
	if ud.TraktTokenId == "" {
		return nil, nil
	}

	if ud.traktToken != nil {
		return ud.traktToken, nil
	}

	otok, err := oauth.GetOAuthTokenById(ud.TraktTokenId)
	if err != nil {
		ud.TraktTokenId = ""
		return nil, errors.New("failed to retrieve token: " + err.Error())
	} else if otok != nil && otok.IsExpired() {
		traktClient := trakt.GetAPIClient(otok.Id)
		settings, err := traktClient.RetrieveSettings(&trakt.RetrieveSettingsParams{})
		if err != nil || settings.Data.User.Ids.Slug != otok.UserId {
			otok.AccessToken = ""
			otok.RefreshToken = ""
			err = oauth.SaveOAuthToken(otok)
			if err != nil {
				log.Error("failed to delete trakt token", "error", err, "id", otok.Id)
			}
			otok = nil
		}
	}
	if otok == nil {
		ud.TraktTokenId = ""
		return nil, errors.New("Invalid or Revoked")
	}

	ud.traktToken = otok
	return ud.traktToken, nil
}

func (ud *UserData) getTMDBToken() (*oauth.OAuthToken, error) {
	if ud.TMDBTokenId == "" {
		return nil, nil
	}

	if ud.tmdbToken != nil {
		return ud.tmdbToken, nil
	}

	otok, err := oauth.GetOAuthTokenById(ud.TMDBTokenId)
	if err != nil {
		ud.TMDBTokenId = ""
		return nil, errors.New("failed to retrieve token: " + err.Error())
	} else if otok != nil && otok.IsExpired() {
		tmdbClient := tmdb.GetAPIClient(otok.Id)
		details, err := tmdbClient.GetAccountDetails(&tmdb.GetAccountDetailsParams{})
		if err != nil || strconv.FormatInt(details.Data.Id, 10) != otok.UserId {
			otok.AccessToken = ""
			otok.RefreshToken = ""
			err = oauth.SaveOAuthToken(otok)
			if err != nil {
				log.Error("failed to delete tmdb token", "error", err, "id", otok.Id)
			}
			otok = nil
		}
	}
	if otok == nil {
		ud.TMDBTokenId = ""
		return nil, errors.New("Invalid or Revoked")
	}

	ud.tmdbToken = otok
	return ud.tmdbToken, nil
}

func (ud *UserData) FetchLetterboxdList(list *letterboxd.LetterboxdList) error {
	if ud.letterboxdById == nil {
		ud.letterboxdById = map[string]letterboxd.LetterboxdList{}
	}
	if list.Id != "" {
		if l, ok := ud.letterboxdById[list.Id]; ok {
			*list = l
			return nil
		}
	}
	if err := list.Fetch(); err != nil {
		return err
	}

	ud.letterboxdById[list.Id] = *list
	return nil
}

func verifyTopPostersAPIKey(apiKey string) error {
	resp, err := config.DefaultHTTPClient.Get("https://api.top-streaming.stream/auth/verify/" + apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("Invalid API key")
	}
	return nil
}

func (ud *UserData) getMDBListUser() (*mdblist.GetMyLimitsData, error) {
	if ud.mdblistUser == nil {
		userParams := mdblist.GetMyLimitsParams{}
		userParams.APIKey = ud.MDBListAPIkey
		res, err := mdblistClient.GetMyLimits(&userParams)
		if err != nil {
			return nil, err
		}
		ud.mdblistUser = &res.Data
	}
	return ud.mdblistUser, nil
}

func (ud *UserData) FetchMDBListList(list *mdblist.MDBListList) error {
	if ud.mdblistById == nil {
		ud.mdblistById = map[string]mdblist.MDBListList{}
	}
	if list.Id != "" {
		if l, ok := ud.mdblistById[list.Id]; ok {
			*list = l
			return nil
		}
	}
	if err := list.Fetch(ud.MDBListAPIkey); err != nil {
		return err
	}
	ud.mdblistById[list.Id] = *list
	return nil
}

func (ud *UserData) FetchAniListList(list *anilist.AniListList, scheduleIdMapSync bool) error {
	if ud.anilistById == nil {
		ud.anilistById = map[string]anilist.AniListList{}
	}
	if list.Id != "" {
		if l, ok := ud.anilistById[list.Id]; ok {
			*list = l
			return nil
		}
	}
	if err := list.Fetch(); err != nil {
		return err
	}

	if scheduleIdMapSync {
		anilist.ScheduleIdMapSync(list.Medias)
	}

	ud.anilistById[list.Id] = *list
	return nil
}

func (ud *UserData) FetchTMDBList(list *tmdb.TMDBList) error {
	if ud.TMDBTokenId == "" {
		return errors.New("TMDB Auth Code missing")
	}
	if ud.tmdbById == nil {
		ud.tmdbById = map[string]tmdb.TMDBList{}
	}
	if list.Id != "" {
		if l, ok := ud.tmdbById[list.Id]; ok {
			*list = l
			return nil
		}
		if list.IsUserSpecific() && list.Username == "" {
			tok, err := ud.getTMDBToken()
			if err != nil {
				return err
			}
			list.Username = tok.UserName
		}
	}
	if err := list.Fetch(ud.TMDBTokenId); err != nil {
		return err
	}

	ud.tmdbById[list.Id] = *list
	return nil
}

func (ud *UserData) FetchTraktList(list *trakt.TraktList) error {
	if ud.TraktTokenId == "" {
		return errors.New("Trakt Auth Code missing")
	}
	if ud.traktById == nil {
		ud.traktById = map[string]trakt.TraktList{}
	}
	if list.Id != "" && !list.IsDynamic() {
		if userSlug, listSlug, ok := strings.Cut(list.Id, "."); ok {
			list.Id = ""
			list.UserId = userSlug
			list.Slug = listSlug
		}
	}
	key := list.Id
	if !list.IsDynamic() && list.UserId != "" && list.Slug != "" {
		key = list.UserId + "." + list.Slug
	}
	if key != "" {
		if l, ok := ud.traktById[key]; ok {
			*list = l
			return nil
		}
	}
	if err := list.Fetch(ud.TraktTokenId); err != nil {
		return err
	}

	ud.traktById[key] = *list
	return nil
}

func (ud *UserData) FetchTVDBList(list *tvdb.TVDBList) error {
	if ud.tvdbById == nil {
		ud.tvdbById = map[string]tvdb.TVDBList{}
	}
	if list.Id != "" {
		if l, ok := ud.tvdbById[list.Id]; ok {
			*list = l
			return nil
		}
	}
	if err := list.Fetch(); err != nil {
		return err
	}

	ud.tvdbById[list.Id] = *list
	return nil
}
