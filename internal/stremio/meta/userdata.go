package stremio_meta

import (
	"net/http"
	"strings"

	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
)

type UserData struct {
	Provider struct {
		Movie  string `json:"movie,omitempty"`
		Series string `json:"series,omitempty"`
		Anime  string `json:"anime,omitempty"`
	} `json:"provider"`
	StreamId struct {
		Movie  string `json:"movie,omitempty"`
		Series string `json:"series,omitempty"`
		Anime  string `json:"anime,omitempty"`
	} `json:"stream_id"`

	encoded string `json:"-"` // correctly configured
}

var udManager = stremio_userdata.NewManager[UserData](&stremio_userdata.ManagerConfig{
	AddonName: "meta",
})

func (ud UserData) HasRequiredValues() bool {
	return ud.Provider.Movie != "" ||
		ud.Provider.Series != "" ||
		ud.Provider.Anime != ""
}

func (ud *UserData) GetEncoded() string {
	return ud.encoded
}

func (ud *UserData) SetEncoded(encoded string) {
	ud.encoded = encoded
}

func (ud *UserData) Ptr() *UserData {
	return ud
}

type userDataError struct {
	provider struct {
		movie  string
		series string
		anime  string
	}
	stream_id struct {
		movie  string
		series string
		anime  string
	}
}

func (uderr userDataError) HasError() bool {
	return uderr.provider.movie != "" || uderr.provider.series != "" || uderr.provider.anime != ""
}

func (uderr userDataError) Error() string {
	var str strings.Builder
	if uderr.provider.movie != "" {
		str.WriteString("provider.movie: " + uderr.provider.movie + "\n")
	}
	if uderr.provider.series != "" {
		str.WriteString("provider.series: " + uderr.provider.series + "\n")
	}
	if uderr.provider.anime != "" {
		str.WriteString("provider.anime: " + uderr.provider.anime + "\n")
	}
	if uderr.stream_id.movie != "" {
		str.WriteString("stream_id.movie: " + uderr.stream_id.movie + "\n")
	}
	if uderr.stream_id.series != "" {
		str.WriteString("stream_id.series: " + uderr.stream_id.series + "\n")
	}
	if uderr.stream_id.anime != "" {
		str.WriteString("stream_id.anime: " + uderr.stream_id.anime + "\n")
	}
	return str.String()
}

func getUserData(r *http.Request) (*UserData, error) {
	ud := &UserData{}
	ud.SetEncoded(r.PathValue("userData"))

	if IsMethod(r, http.MethodGet) || IsMethod(r, http.MethodHead) {
		if err := udManager.Resolve(ud); err != nil {
			return nil, err
		}
		if ud.encoded == "" {
			return ud, nil
		}
	}

	if IsMethod(r, http.MethodPost) {
		err := r.ParseForm()
		if err != nil {
			return nil, err
		}

		udErr := userDataError{}

		if movie := r.Form.Get("provider.movie"); movie != "" {
			ud.Provider.Movie = movie
		}

		if series := r.Form.Get("provider.series"); series != "" {
			ud.Provider.Series = series
		}

		if anime := r.Form.Get("provider.anime"); anime != "" {
			ud.Provider.Anime = anime
		}

		if movie := r.Form.Get("stream_id.movie"); movie != "" {
			ud.StreamId.Movie = movie
		}

		if series := r.Form.Get("stream_id.series"); series != "" {
			ud.StreamId.Series = series
		}

		if anime := r.Form.Get("stream_id.anime"); anime != "" {
			ud.StreamId.Anime = anime
		}

		if udErr.HasError() {
			return ud, udErr
		}
	}

	return ud, nil
}
