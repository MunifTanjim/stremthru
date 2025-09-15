package worker_queue

import (
	"time"

	"github.com/MunifTanjim/stremthru/internal/config"
)

type LetterboxdListSyncerQueueItem struct {
	ListId string
}

var LetterboxdListSyncerQueue = WorkerQueue[LetterboxdListSyncerQueueItem]{
	debounceTime: 60 * time.Second,
	getKey: func(item LetterboxdListSyncerQueueItem) string {
		return item.ListId
	},
	transform: func(item *LetterboxdListSyncerQueueItem) *LetterboxdListSyncerQueueItem {
		return item
	},
	Disabled: !config.Integration.Letterboxd.IsEnabled() && !config.Integration.Letterboxd.IsPiggybacked(),
}
