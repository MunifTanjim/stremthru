package stremio_wrap

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MunifTanjim/stremthru/internal/context"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
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

func (ud UserData) fetchCatalog(ctx *context.StoreContext, w http.ResponseWriter, r *http.Request, rType, id, extra string) (*stremio.CatalogHandlerResponse, error) {
	idx, catalogId, err := parseCatalogId(id, &ud)
	if err != nil {
		SendError(w, r, err)
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

	rpdbPosterBaseUrl := ""
	if ud.RPDBAPIKey != "" {
		rpdbPosterBaseUrl = "https://api.ratingposterdb.com/" + ud.RPDBAPIKey + "/imdb/poster-default/"
	}

	for i := range res.Data.Metas {
		item := &res.Data.Metas[i]
		if rpdbPosterBaseUrl != "" && strings.HasPrefix(item.Id, "tt") {
			item.Poster = rpdbPosterBaseUrl + item.Id + ".jpg?fallback=true"
		}
	}

	return &res.Data, nil
}
