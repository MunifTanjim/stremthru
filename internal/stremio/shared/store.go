package stremio_shared

import (
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
)

func GetStoreCodeOptions() []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "ad", Label: "AllDebrid"},
		{Value: "dl", Label: "DebridLink"},
		{Value: "ed", Label: "EasyDebrid"},
		{Value: "oc", Label: "Offcloud"},
		{Value: "pm", Label: "Premiumize"},
		{Value: "pp", Label: "PikPak"},
		{Value: "rd", Label: "RealDebrid"},
		{Value: "tb", Label: "TorBox"},
	}
	if config.IsPublicInstance {
		options[0].Disabled = true
		options[0].Label = ""
	}
	return options
}

func WaitForMagnetStatus(ctx *context.StoreContext, m *store.GetMagnetData, status store.MagnetStatus, maxRetry int, retryInterval time.Duration) (*store.GetMagnetData, error) {
	retry := 0
	for m.Status != status && retry < maxRetry {
		gmParams := &store.GetMagnetParams{
			Id:       m.Id,
			ClientIP: ctx.ClientIP,
		}
		gmParams.APIKey = ctx.StoreAuthToken
		magnet, err := ctx.Store.GetMagnet(gmParams)
		if err != nil {
			return m, err
		}
		m = magnet
		time.Sleep(retryInterval)
		retry++
	}
	if m.Status != status {
		error := core.NewStoreError("torrent failed to reach status: " + string(status))
		error.StoreName = string(ctx.Store.GetName())
		return m, error
	}
	return m, nil
}
