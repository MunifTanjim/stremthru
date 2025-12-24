package stremio_wrap

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/MunifTanjim/stremthru/internal/context"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/internal/util"
	"github.com/MunifTanjim/stremthru/stremio"
)

func parseCatalogId(id string, ud *UserData) (idx int, catalogId string, err error) {
	if len(ud.Upstreams) == 1 {
		return 0, id, nil
	}

	idxStr, catalogId, ok := strings.Cut(id, "::")
	if !ok {
		return -1, "", errors.New("invalid id")
	}
	idx, err = strconv.Atoi(idxStr)
	if err != nil {
		return -1, "", err
	}
	if len(ud.Upstreams) <= idx {
		return -1, "", errors.New("invalid id")
	}
	return idx, catalogId, nil
}

func (ud UserData) fetchAddonCatalog(ctx *context.StoreContext, w http.ResponseWriter, r *http.Request, rType, id string) {
	idx, catalogId, err := parseCatalogId(id, &ud)
	if err != nil {
		SendError(w, r, err)
		return
	}
	addon.ProxyResource(w, r, &stremio_addon.ProxyResourceParams{
		BaseURL:  ud.Upstreams[idx].baseUrl,
		Resource: string(stremio.ResourceNameAddonCatalog),
		Type:     rType,
		Id:       catalogId,
		ClientIP: ctx.ClientIP,
	})
}

func (ud UserData) fetchCatalog(ctx *context.StoreContext, rType, id, extra string) (*stremio.CatalogHandlerResponse, error) {
	if id == catalog_id_calendar_videos || id == catalog_id_last_videos {
		return ud.fetchCatalogWithMetasDetails(ctx, rType, id, extra)
	}

	idx, catalogId, err := parseCatalogId(id, &ud)
	if err != nil {
		return nil, err
	}

	res, err := addon.FetchCatalog(&stremio_addon.FetchCatalogParams{
		BaseURL:  ud.Upstreams[idx].baseUrl,
		Type:     rType,
		Id:       catalogId,
		Extra:    extra,
		ClientIP: ctx.ClientIP,
	})
	if err != nil {
		return nil, err
	}

	posterBaseUrl := ""
	posterQueryParams := ""
	if ud.RPDBAPIKey != "" {
		posterBaseUrl = "https://api.ratingposterdb.com/" + ud.RPDBAPIKey + "/imdb/poster-default/"
		posterQueryParams = "?fallback=true"
	} else if ud.TopPostersAPIKey != "" {
		posterBaseUrl = "https://api.top-streaming.stream/" + ud.TopPostersAPIKey + "/imdb/poster-default/"
	}

	for i := range res.Data.Metas {
		item := &res.Data.Metas[i]
		if posterBaseUrl != "" && strings.HasPrefix(item.Id, "tt") {
			item.Poster = posterBaseUrl + item.Id + ".jpg" + posterQueryParams
		}
	}

	return &res.Data, nil
}

func (ud UserData) fetchCatalogWithMetasDetails(ctx *context.StoreContext, rType, id, extra string) (*stremio.CatalogHandlerResponse, error) {
	log := ctx.Log

	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameCatalog, rType, id)
	if err != nil {
		return nil, err
	}

	upstreamsCount := len(upstreams)
	log.Debug("found addons for catalog", "count", upstreamsCount)

	if upstreamsCount == 0 {
		return &stremio.CatalogHandlerResponse{}, nil
	}

	chunks := make([][]stremio.Meta, upstreamsCount)
	errs := make([]error, upstreamsCount)

	var wg sync.WaitGroup
	for i := range upstreams {
		wg.Go(func() {
			res, err := addon.FetchCatalog(&stremio_addon.FetchCatalogParams{
				BaseURL:  upstreams[i].baseUrl,
				Type:     rType,
				Id:       id,
				Extra:    extra,
				ClientIP: ctx.ClientIP,
			})
			chunks[i] = res.Data.MetasDetailed
			errs[i] = err
		})
	}
	wg.Wait()

	result := &stremio.CatalogHandlerResponse{
		MetasDetailed: []stremio.Meta{},
	}

	seenIds := util.NewSet[string]()

	for i, chunk := range chunks {
		if errs[i] != nil {
			log.Error("failed to fetch catalog", "error", errs[i], "hostname", upstreams[i].baseUrl.Hostname())
			continue
		}
		for j := range chunk {
			id := chunk[j].Id
			if seenIds.Has(id) {
				continue
			}
			seenIds.Add(id)
			result.MetasDetailed = append(result.MetasDetailed, chunk[j])
		}
	}

	return result, nil
}
