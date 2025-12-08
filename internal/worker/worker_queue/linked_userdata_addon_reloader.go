package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type UserdataAddonReloaderQueueItem struct {
	Addon string
	Key   string
}

var LinkedUserdataAddonReloaderQueue = WorkerQueue[UserdataAddonReloaderQueueItem]{
	debounceTime: 1 * time.Minute,
	getKey: func(item UserdataAddonReloaderQueueItem) string {
		return item.Addon + ":" + item.Key
	},
	transform: func(item *UserdataAddonReloaderQueueItem) *UserdataAddonReloaderQueueItem {
		return item
	},
	Disabled: !config.Feature.HasVault(),
}
