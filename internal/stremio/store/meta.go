package stremio_store

import (
	"net/url"

	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/stremio"
)

var client = func() *stremio_addon.Client {
	return stremio_addon.NewClient(&stremio_addon.ClientConfig{})
}()

var cinemetaBaseUrl = func() *url.URL {
	url, err := url.Parse("https://v3-cinemeta.strem.io/")
	if err != nil {
		panic(err)
	}
	return url
}()

func fetchMeta(sType, imdbId, clientIp string) (stremio.MetaHandlerResponse, error) {
	res, err := client.FetchMeta(&stremio_addon.FetchMetaParams{
		BaseURL:  cinemetaBaseUrl,
		Type:     sType,
		Id:       imdbId + ".json",
		ClientIP: clientIp,
	})
	return res.Data, err
}

func getPosterUrl(imdbId string) string {
	return "https://images.metahub.space/poster/small/" + imdbId + "/img"
}
