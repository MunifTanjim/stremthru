package stremio_wrap

import (
	"sync"

	"github.com/MunifTanjim/stremthru/internal/context"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	"github.com/MunifTanjim/stremthru/stremio"
)

func (ud UserData) fetchSubtitles(ctx *context.StoreContext, rType, id, extra string) (*stremio.SubtitlesHandlerResponse, error) {
	log := ctx.Log

	upstreams, err := ud.getUpstreams(ctx, stremio.ResourceNameSubtitles, rType, id)
	if err != nil {
		return nil, err
	}

	upstreamsCount := len(upstreams)
	log.Debug("found addons for subtitles", "count", upstreamsCount)

	chunks := make([][]stremio.Subtitle, upstreamsCount)
	errs := make([]error, len(upstreams))

	var wg sync.WaitGroup
	for i := range upstreams {
		wg.Go(func() {
			res, err := addon.FetchSubtitles(&stremio_addon.FetchSubtitlesParams{
				BaseURL:  upstreams[i].baseUrl,
				Type:     rType,
				Id:       id,
				Extra:    extra,
				ClientIP: ctx.ClientIP,
			})
			chunks[i] = res.Data.Subtitles
			errs[i] = err
		})
	}
	wg.Wait()

	subtitles := []stremio.Subtitle{}
	for i := range chunks {
		if errs[i] != nil {
			log.Error("failed to fetch subtitles", "error", errs[i])
			continue
		}
		subtitles = append(subtitles, chunks[i]...)
	}

	return &stremio.SubtitlesHandlerResponse{
		Subtitles: subtitles,
	}, nil
}
