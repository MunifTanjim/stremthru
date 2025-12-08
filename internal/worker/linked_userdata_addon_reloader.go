package worker

import (
	"strings"

	stremio_account "github.com/MunifTanjim/stremthru/internal/stremio/account"
	stremio_addon "github.com/MunifTanjim/stremthru/internal/stremio/addon"
	stremio_api "github.com/MunifTanjim/stremthru/internal/stremio/api"
	stremio_userdata_account "github.com/MunifTanjim/stremthru/internal/stremio/userdata/account"
	"github.com/MunifTanjim/stremthru/internal/worker/worker_queue"
	"github.com/MunifTanjim/stremthru/stremio"
)

func InitLinkedUserdataAddonReloaderWorker(conf *WorkerConfig) *Worker {
	stremioClient := stremio_api.NewClient(&stremio_api.ClientConfig{})
	addonClient := stremio_addon.NewClient(&stremio_addon.ClientConfig{})

	conf.Executor = func(w *Worker) error {
		log := w.Log

		worker_queue.LinkedUserdataAddonReloaderQueue.Process(func(item worker_queue.UserdataAddonReloaderQueueItem) error {
			accountIds, err := stremio_userdata_account.GetAccountIds(item.Addon, item.Key)
			if err != nil {
				log.Error("failed to get account ids", "error", err, "addon", item.Addon, "key", item.Key)
				return err
			}

			if len(accountIds) == 0 {
				log.Debug("no linked accounts", "addon", item.Addon, "key", item.Key)
				return nil
			}

			for _, accountId := range accountIds {
				addon, key := item.Addon, item.Key
				addonTransportURLPattern := "/stremio/" + addon + "/k." + key + "/"

				account, err := stremio_account.GetById(accountId)
				if err != nil {
					log.Error("failed to get account", "error", err, "account_id", accountId)
					continue
				}
				if account == nil {
					log.Debug("account not found", "account_id", accountId)
					continue
				}

				token, err := account.GetValidToken()
				if err != nil {
					log.Warn("failed to get valid token", "error", err, "account_id", accountId)
					continue
				}

				params := &stremio_api.GetAddonsParams{}
				params.APIKey = token
				getRes, err := stremioClient.GetAddons(params)
				if err != nil {
					log.Error("failed to get addons for account", "error", err, "addon", addon, "key", key, "account_id", accountId)
					continue
				}

				addons := getRes.Data.Addons

				idx := -1
				for i := range addons {
					if strings.Contains(addons[i].TransportUrl, addonTransportURLPattern) {
						idx = i
						break
					}
				}

				if idx == -1 {
					log.Debug("addon not found in account", "addon", addon, "key", key, "account_id", accountId)
					err := stremio_userdata_account.Unlink(addon, key, accountId)
					if err != nil {
						log.Error("failed to unlink addon from account", "error", err, "addon", addon, "key", key, "account_id", accountId)
					}
					continue
				}

				currentAddon := &addons[idx]
				manifestUrl := currentAddon.TransportUrl

				baseUrl, err := stremio_addon.ExtractBaseURL(manifestUrl)
				if err != nil {
					log.Error("failed to extract base url", "error", err)
					continue
				}

				manifest, err := addonClient.GetManifest(&stremio_addon.GetManifestParams{BaseURL: baseUrl})
				if err != nil {
					log.Error("failed to get manifest", "error", err)
					continue
				}

				refreshedAddon := stremio.Addon{
					TransportUrl:  manifestUrl,
					TransportName: currentAddon.TransportName,
					Manifest:      manifest.Data,
					Flags:         currentAddon.Flags,
				}

				if currentAddon.Manifest.BehaviorHints != nil && !currentAddon.Manifest.BehaviorHints.Configurable {
					if refreshedAddon.Manifest.BehaviorHints == nil {
						refreshedAddon.Manifest.BehaviorHints = &stremio.BehaviorHints{}
					}
					refreshedAddon.Manifest.BehaviorHints.Configurable = false
				}

				addons[idx] = refreshedAddon

				setParams := &stremio_api.SetAddonsParams{Addons: addons}
				setParams.APIKey = token
				setRes, err := stremioClient.SetAddons(setParams)
				if err != nil {
					log.Error("failed to set addons for account", "error", err, "addon", addon, "key", key, "account_id", accountId)
					continue
				}

				if !setRes.Data.Success {
					log.Error("failed to set addons for account", "addon", addon, "key", key, "account_id", accountId)
					continue
				}

				log.Info("reloaded addon", "addon", addon, "key", key, "account_id", accountId)
			}

			return nil
		})

		return nil
	}

	worker := NewWorker(conf)

	return worker
}
