package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type MagnetCachePullerQueueItem struct {
	ClientIP   string
	Hash       string
	SId        string
	StoreCode  string
	StoreToken string
}

var MagnetCachePullerQueue = WorkerQueue[MagnetCachePullerQueueItem]{
	debounceTime: 5 * time.Minute,
	getKey: func(item MagnetCachePullerQueueItem) string {
		return item.StoreCode + ":" + item.SId + ":" + item.Hash
	},
	getGroupKey: func(item MagnetCachePullerQueueItem) string {
		return item.StoreCode + ":" + item.SId
	},
	transform: func(item *MagnetCachePullerQueueItem) *MagnetCachePullerQueueItem {
		return item
	},
	Disabled: !config.LazyPeer,
}
