package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type StoreCrawlerQueueItem struct {
	StoreCode  string
	StoreToken string
}

var StoreCrawlerQueue = WorkerQueue[StoreCrawlerQueueItem]{
	debounceTime: 15 * time.Minute,
	getKey: func(item StoreCrawlerQueueItem) string {
		return item.StoreCode + ":" + item.StoreToken
	},
	transform: func(item *StoreCrawlerQueueItem) *StoreCrawlerQueueItem {
		return item
	},
	Disabled: !config.Feature.HasTorrentInfo(),
}
