package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type MediaProberQueueItem struct {
	Hash string
	Path string
	Link string
}

var MediaProberQueue = WorkerQueue[MediaProberQueueItem]{
	debounceTime: 10 * time.Second,
	getKey: func(item MediaProberQueueItem) string {
		return item.Hash + ":" + item.Path
	},
	transform: func(item *MediaProberQueueItem) *MediaProberQueueItem {
		return item
	},
	Disabled: !config.Feature.IsEnabled(config.FeatureMediaProbe),
}
