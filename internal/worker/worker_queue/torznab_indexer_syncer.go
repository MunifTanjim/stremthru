package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type TorznabIndexerSyncerQueueItem struct {
	SId string
}

var TorznabIndexerSyncerQueue = WorkerQueue[TorznabIndexerSyncerQueueItem]{
	debounceTime: 5 * time.Minute,
	getKey: func(item TorznabIndexerSyncerQueueItem) string {
		return item.SId
	},
	transform: func(item *TorznabIndexerSyncerQueueItem) *TorznabIndexerSyncerQueueItem {
		return item
	},
	Disabled: !config.Feature.HasVault(),
}
