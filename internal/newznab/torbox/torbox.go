package newznab_torbox

import (
	"net/url"
	"sync"
	"time"

	newznab_client "github.com/MunifTanjim/stremthru/internal/newznab/client"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/internal/znab"
	"github.com/MunifTanjim/stremthru/store"
	"github.com/MunifTanjim/stremthru/store/torbox"
)

var (
	_ newznab_client.Indexer = (*Indexer)(nil)
)

type Indexer struct {
	api *torbox.APIClient
}

func NewIndexer(api *torbox.APIClient) *Indexer {
	return &Indexer{api: api}
}

var getCaps = sync.OnceValue(func() *znab.Caps {
	return &znab.Caps{
		Server: &znab.CapsServer{
			Title: "Torbox",
		},
		Searching: &znab.CapsSearching{
			MovieSearch: &znab.CapsSearchingItem{
				Available: true,
				SupportedParams: znab.CapsSearchingItemSupportedParams{
					znab.SearchParamIMDBId,
				},
			},
			TVSearch: &znab.CapsSearchingItem{
				Available: true,
				SupportedParams: znab.CapsSearchingItemSupportedParams{
					znab.SearchParamIMDBId,
					znab.SearchParamSeason,
					znab.SearchParamEp,
				},
			},
		},
		Categories: []znab.CapsCategory{},
	}
})

func (i *Indexer) GetId() string {
	return string(store.StoreNameTorBox)
}

func (i *Indexer) NewSearchQuery(fn func(caps *znab.Caps) newznab_client.Function) (*newznab_client.Query, error) {
	caps := getCaps()
	return newznab_client.NewQuery(caps).SetT(fn(caps)), nil
}

func convertNZBToNewz(nzb *torbox.UsenetSearchByIDDataNZB) newznab_client.Newz {
	now := time.Now()
	var age time.Duration
	if v, err := time.ParseDuration(nzb.Age); err == nil {
		age = v
	}
	date := now.Add(-age)

	categories := util.SliceMapIntToString(nzb.Categories)

	return newznab_client.Newz{
		Title:          nzb.RawTitle,
		GUID:           nzb.Hash,
		Size:           nzb.Size,
		Files:          nzb.Files,
		Categories:     categories,
		DownloadLink:   nzb.NZB,
		Date:           date,
		PublishDate:    date,
		Hash:           nzb.Hash,
		LockedDownload: true,
		LockedProvider: string(store.StoreNameTorBox),
	}
}

func (i *Indexer) Search(query url.Values) ([]newznab_client.Newz, error) {
	imdbId := query.Get(znab.SearchParamIMDBId)
	if imdbId == "" {
		return nil, nil
	}

	params := &torbox.SearchUsenetByIDParams{
		IDType:            "imdb",
		ID:                imdbId,
		Season:            util.SafeParseInt(query.Get(znab.SearchParamSeason), 0),
		Episode:           util.SafeParseInt(query.Get(znab.SearchParamEp), 0),
		SearchUserEngines: true,
	}

	resp, err := i.api.SearchUsenetByID(params)
	if err != nil {
		return nil, err
	}

	nzbs := resp.Data.NZBs
	result := make([]newznab_client.Newz, 0, len(nzbs))
	for idx := range nzbs {
		nzb := &nzbs[idx]
		result = append(result, convertNZBToNewz(nzb))
	}
	return result, nil
}
