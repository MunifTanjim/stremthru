package stremio_shared

import (
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/context"
	"github.com/MunifTanjim/stremthru/internal/stremio/configure"
	"github.com/MunifTanjim/stremthru/store"
)

var P2PEnabled = config.Feature.IsEnabled(config.FeatureStremioP2P)

func GetStoreCodeOptions(includeP2P bool) []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "ad", Label: "AllDebrid"},
		{Value: "dr", Label: "⚠️ Debrider"},
		{Value: "dl", Label: "DebridLink"},
		{Value: "ed", Label: "⚠️ EasyDebrid"},
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
	if P2PEnabled && includeP2P {
		options = append(options, configure.ConfigOption{
			Value: "p2p",
			Label: "P2P 🧪",
		})
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
		error := core.NewStoreError("torrent failed to reach status: " + string(status) + ", last status: " + string(m.Status))
		error.StoreName = string(ctx.Store.GetName())
		return m, error
	}
	return m, nil
}

func GetStoreCodeOptionsForNewz() []configure.ConfigOption {
	options := []configure.ConfigOption{
		{Value: "", Label: "StremThru"},
		{Value: "tb", Label: "TorBox"},
	}
	if config.IsPublicInstance {
		options[0].Disabled = true
		options[0].Label = ""
	}
	return options
}

func WaitForNewzStatus(ctx *context.StoreContext, data *store.GetNewzData, status store.NewzStatus, maxRetry int, retryInterval time.Duration) (*store.GetNewzData, error) {
	retry := 0
	for data.Status != status && retry < maxRetry {
		params := &store.GetNewzParams{
			Id:       data.Id,
			ClientIP: ctx.ClientIP,
		}
		params.APIKey = ctx.StoreAuthToken
		newz, err := ctx.Store.(store.NewzStore).GetNewz(params)
		if err != nil {
			return data, err
		}
		data = newz
		time.Sleep(retryInterval)
		retry++
	}
	if data.Status != status {
		error := core.NewStoreError("newz failed to reach status: " + string(status) + ", last status: " + string(data.Status))
		error.StoreName = string(ctx.Store.GetName())
		return data, error
	}
	return data, nil
}
